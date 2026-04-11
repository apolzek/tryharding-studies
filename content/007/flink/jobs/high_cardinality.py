from pyflink.table import EnvironmentSettings, TableEnvironment, TableDescriptor, Schema, Row
from pyflink.table.types import DataTypes
from pyflink.table.udf import udf
from pyflink.table.expressions import call, col
import re
from collections import defaultdict, Counter

env_settings = EnvironmentSettings.in_streaming_mode()
t_env = TableEnvironment.create(env_settings)

HIGH_CARDINALITY_THRESHOLD = 10  
CARDINALITY_WINDOW_SIZE = 100 

# Kafka source
source_schema = Schema.new_builder() \
    .column('message', DataTypes.STRING()) \
    .build()

t_env.create_temporary_table(
    'source',
    TableDescriptor.for_connector('kafka')
    .schema(source_schema)
    .option("topic", "raw-logs")
    .option("properties.bootstrap.servers", "kafka:29092")
    .option("scan.startup.mode", "latest-offset")
    .option("value.format", "raw")
    .build()
)

class CardinalityTracker:
    def __init__(self):
        # metric_name -> label_name -> set of values
        self.label_values = defaultdict(lambda: defaultdict(set))
        # metric_name -> set of high cardinality labels
        self.high_cardinality_labels = defaultdict(set)
        self.processed_count = 0
        self.total_high_cardinality_metrics = 0
    
    def add_metric(self, metric_name, labels):
        for label_name, label_value in labels.items():
            self.label_values[metric_name][label_name].add(label_value)
            
            # Verifica se a label ultrapassou o threshold
            if len(self.label_values[metric_name][label_name]) > HIGH_CARDINALITY_THRESHOLD:
                if label_name not in self.high_cardinality_labels[metric_name]:
                    self.high_cardinality_labels[metric_name].add(label_name)
                    self.total_high_cardinality_metrics += 1
                    print(f"🚨 HIGH CARDINALITY DETECTED:")
                    print(f"   Metric: {metric_name}")
                    print(f"   Label: {label_name}")
                    print(f"   Unique values: {len(self.label_values[metric_name][label_name])}")
                    print(f"   Threshold: {HIGH_CARDINALITY_THRESHOLD}")
                    print(f"   🔢 Total high-cardinality metrics so far: {self.total_high_cardinality_metrics}")
                    print()
    
    def should_remove_label(self, metric_name, label_name):
        return label_name in self.high_cardinality_labels[metric_name]

# Instância global do tracker
cardinality_tracker = CardinalityTracker()

def parse_prometheus_line(line):
    """Parse uma linha de métrica do Prometheus"""
    line = line.strip()
    
    # Ignora comentários e linhas vazias
    if not line or line.startswith('#'):
        return None
    
    # Regex para extrair nome da métrica, labels e valor
    pattern = r'^([a-zA-Z_:][a-zA-Z0-9_:]*?)(\{([^}]*)\})?\s+([^\s]+)(\s+\d+)?$'
    match = re.match(pattern, line)
    
    if not match:
        return None
    
    metric_name = match.group(1)
    labels_str = match.group(3) or ""
    value = match.group(4)
    
    # Parse das labels
    labels = {}
    if labels_str:
        # Regex para extrair pares key=value das labels
        label_pattern = r'([a-zA-Z_][a-zA-Z0-9_]*)="([^"]*)"'
        for label_match in re.finditer(label_pattern, labels_str):
            label_name = label_match.group(1)
            label_value = label_match.group(2)
            labels[label_name] = label_value
    
    return {
        'metric_name': metric_name,
        'labels': labels,
        'value': value,
        'original_line': line
    }

def format_metric_line(metric_name, labels, value):
    """Reconstrói a linha da métrica"""
    if labels:
        labels_str = ','.join([f'{k}="{v}"' for k, v in labels.items()])
        return f'{metric_name}{{{labels_str}}} {value}'
    else:
        return f'{metric_name} {value}'

@udf(result_type=DataTypes.ROW([
    DataTypes.FIELD("original_line", DataTypes.STRING()),
    DataTypes.FIELD("processed_line", DataTypes.STRING()),
    DataTypes.FIELD("removed_labels", DataTypes.STRING()),
    DataTypes.FIELD("has_high_cardinality", DataTypes.BOOLEAN())
]))
def process_prometheus_metrics(message):
    """UDF principal para processar métricas do Prometheus"""
    
    lines = message.split('\n')
    results = []
    
    for line in lines:
        parsed = parse_prometheus_line(line)
        
        if parsed is None:
            # Linha de comentário ou inválida, retorna sem modificação
            continue
        
        metric_name = parsed['metric_name']
        labels = parsed['labels']
        value = parsed['value']
        original_line = parsed['original_line']
        
        # Atualiza o tracker de cardinalidade
        cardinality_tracker.add_metric(metric_name, labels)
        
        # Filtra labels com alta cardinalidade
        filtered_labels = {}
        removed_labels = []
        has_high_cardinality = False
        
        for label_name, label_value in labels.items():
            if cardinality_tracker.should_remove_label(metric_name, label_name):
                removed_labels.append(f"{label_name}={label_value}")
                has_high_cardinality = True
            else:
                filtered_labels[label_name] = label_value
        
        # Reconstrói a linha processada
        processed_line = format_metric_line(metric_name, filtered_labels, value)
        
        # Se removeu labels, imprime o antes e depois
        if removed_labels:
            print("🔄 METRIC PROCESSED:")
            print(f"   BEFORE: {original_line}")
            print(f"   AFTER:  {processed_line}")
            print(f"   REMOVED LABELS: {', '.join(removed_labels)}")
            print()
        
        return Row(
            original_line=original_line,
            processed_line=processed_line,
            removed_labels=','.join(removed_labels) if removed_labels else "",
            has_high_cardinality=has_high_cardinality
        )
    
    # Se não houver métricas válidas, retorna linha vazia
    return Row(
        original_line="",
        processed_line="",
        removed_labels="",
        has_high_cardinality=False
    )

# UDF para processar lote de métricas (otimização)
@udf(result_type=DataTypes.ARRAY(DataTypes.ROW([
    DataTypes.FIELD("original_line", DataTypes.STRING()),
    DataTypes.FIELD("processed_line", DataTypes.STRING()),
    DataTypes.FIELD("removed_labels", DataTypes.STRING()),
    DataTypes.FIELD("has_high_cardinality", DataTypes.BOOLEAN())
])))
def process_prometheus_batch(message):
    """UDF otimizada para processar lotes de métricas"""
    
    lines = message.split('\n')
    results = []
    
    for line in lines:
        parsed = parse_prometheus_line(line)
        
        if parsed is None:
            continue
        
        metric_name = parsed['metric_name']
        labels = parsed['labels']
        value = parsed['value']
        original_line = parsed['original_line']
        
        # Atualiza o tracker de cardinalidade
        cardinality_tracker.add_metric(metric_name, labels)
        
        # Filtra labels com alta cardinalidade
        filtered_labels = {}
        removed_labels = []
        has_high_cardinality = False
        
        for label_name, label_value in labels.items():
            if cardinality_tracker.should_remove_label(metric_name, label_name):
                removed_labels.append(f"{label_name}={label_value}")
                has_high_cardinality = True
            else:
                filtered_labels[label_name] = label_value
        
        # Reconstrói a linha processada
        processed_line = format_metric_line(metric_name, filtered_labels, value)
        
        # Se removeu labels, imprime o antes e depois
        if removed_labels:
            print("🔄 METRIC PROCESSED:")
            print(f"   BEFORE: {original_line}")
            print(f"   AFTER:  {processed_line}")
            print(f"   REMOVED LABELS: {', '.join(removed_labels)}")
            print()
        
        results.append(Row(
            original_line=original_line,
            processed_line=processed_line,
            removed_labels=','.join(removed_labels) if removed_labels else "",
            has_high_cardinality=has_high_cardinality
        ))
    
    return results

# Registra as UDFs
t_env.create_temporary_function('process_prometheus_metrics', process_prometheus_metrics)
t_env.create_temporary_function('process_prometheus_batch', process_prometheus_batch)

print("🚀 Starting Prometheus High Cardinality Detector...")
print(f"📊 Configuration:")
print(f"   High Cardinality Threshold: {HIGH_CARDINALITY_THRESHOLD}")
print(f"   Window Size: {CARDINALITY_WINDOW_SIZE}")
print()

# Pipeline Flink
source = t_env.from_path('source')

# Processa as métricas
processed_metrics = source.select(
    col("message"),
    call("process_prometheus_metrics", col("message")).alias("result")
)

# Seleciona apenas métricas que foram processadas (não vazias)
final_metrics = processed_metrics.select(
    col("result").get("original_line").alias("original_line"),
    col("result").get("processed_line").alias("processed_line"),
    col("result").get("removed_labels").alias("removed_labels"),
    col("result").get("has_high_cardinality").alias("has_high_cardinality")
).filter(col("original_line") != "")

# Executa o pipeline (apenas processa, não grava em nenhum destino)
try:
    # Para desenvolvimento/teste, você pode usar print_table para ver os resultados
    # final_metrics.print_schema()
    
    # Execute o pipeline
    final_metrics.execute().print()
    
except Exception as e:
    print(f"❌ Error executing pipeline: {e}")

print("✅ Pipeline started successfully!")
print("   Monitoring Kafka topic 'prometheus-metrics' for high cardinality labels...")
print("   Press Ctrl+C to stop")