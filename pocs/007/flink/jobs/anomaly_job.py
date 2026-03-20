from pyflink.table import EnvironmentSettings, TableEnvironment, TableDescriptor, Schema, Row
from pyflink.table.types import DataTypes
from pyflink.table.udf import udf
from pyflink.table.expressions import call, col
from datetime import datetime

from core.constants import node_no_dict, channel_values, component_values, level_values, columns
from core.sub_function import parse_log_line, match_pattern, node_extractor, safe_int
from core.model_api import predict_api

env_settings = EnvironmentSettings.in_streaming_mode()
t_env = TableEnvironment.create(env_settings)


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

# postgres sink
pg_schema = Schema.new_builder() \
    .column('timestamp', DataTypes.STRING()) \
    .column('date', DataTypes.DATE()) \
    .column('node', DataTypes.STRING()) \
    .column('time', DataTypes.TIMESTAMP()) \
    .column('nodeRepeat', DataTypes.STRING()) \
    .column('type', DataTypes.STRING()) \
    .column('component', DataTypes.STRING()) \
    .column('level', DataTypes.STRING()) \
    .column('content', DataTypes.STRING()) \
    .column('label', DataTypes.STRING()) \
    .build()

t_env.create_temporary_table(
    'pg_sink',
    TableDescriptor.for_connector("jdbc")
    .schema(pg_schema)
    .option("url", "jdbc:postgresql://postgres:5432/postgres")
    .option("table-name", "event_logs")
    .option("username", "postgres")
    .option("password", "mysecret")
    .option("driver", "org.postgresql.Driver")
    .build()
)

print(f"CHEGUEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEI")

# udf gọi model api
@udf(result_type=DataTypes.ROW([DataTypes.FIELD("Label", DataTypes.STRING())]))
def predict_anomaly(log: str):
    id_, date, code1, time, code2, typee, comp1, level, content = parse_log_line(log)
    structured_log = {
        "Timestamp": id_,
        "Date": date,
        "Node": code1,
        "Time": time,
        "NodeRepeat": code2,
        "Type": typee,
        "Component": comp1,
        "Level": level,
        "Content": content
    }

    node_extract = node_extractor(structured_log['Node'])

    model_input = {
        'eventId': match_pattern(structured_log['Content']),
        'rack': safe_int(node_extract['rack']),
        'midplane': safe_int(node_extract['midplane']),
        'uid': safe_int(node_extract['uid']),
        'jid': safe_int(node_extract['jid']),
        'type': 1 if structured_log['Type'] == "RAS" else 0,
        'node_type': 0 if node_extract['node_type'] == "N" else (
            1 if node_extract['node_type'] == "L" else -1),
        'control_IO': 0 if node_extract['control_IO'] == "C" else (
            1 if node_extract['node_type'] == "I" else -1),
        'node_no': node_no_dict[node_extract['node_no']]
    }

    for channel_value in channel_values:
        model_input[f'channel_{channel_value}'] = 1 if node_extract['channel'] == channel_value else 0

    for component_value in component_values:
        model_input[f'component_{component_value}'] = 1 if structured_log['Component'] == component_value else 0

    for level_value in level_values:
        model_input[f'level_{level_value}'] = 1 if structured_log['Level'] == level_value else 0

    ordered_record = {k: model_input[k] for k in columns}
    label = predict_api(ordered_record)
    return Row(Label=label)


# dummy visualization udf
@udf(result_type=DataTypes.ROW([
    DataTypes.FIELD("Timestamp", DataTypes.STRING()),
    DataTypes.FIELD("Date", DataTypes.DATE()),
    DataTypes.FIELD("Node", DataTypes.STRING()),
    DataTypes.FIELD("Time", DataTypes.TIMESTAMP()),
    DataTypes.FIELD("NodeRepeat", DataTypes.STRING()),
    DataTypes.FIELD("Type", DataTypes.STRING()),
    DataTypes.FIELD("Component", DataTypes.STRING()),
    DataTypes.FIELD("Level", DataTypes.STRING()),
    DataTypes.FIELD("Content", DataTypes.STRING())
]))
def process_key(log):
    id_, date, code1, _, code2, typee, comp1, level, content = parse_log_line(log)
    return Row(
        Timestamp=id_,
        Date=datetime.strptime(date, '%Y.%m.%d').date(),
        Node=code1,
        Time=datetime.now(),
        NodeRepeat=code2,
        Type=typee,
        Component=comp1,
        Level=level,
        Content=content
    )


t_env.create_temporary_function('process_key', process_key)
t_env.create_temporary_function('predict_anomaly', predict_anomaly)

# pipeline flink
source = t_env.from_path('source')

with_anomaly = source.select(
    col("message"),
    call("predict_anomaly", col("message")).alias("anomaly_result")
)

with_visualization = with_anomaly.select(
    col("message"),
    col("anomaly_result"),
    call("process_key", col("message")).alias("viz_result")
)

final = with_visualization.select(
    col("viz_result").get("Timestamp").alias("timestamp"),
    col("viz_result").get("Date").alias("date"),
    col("viz_result").get("Node").alias("node"),
    col("viz_result").get("Time").alias("time"),
    col("viz_result").get("NodeRepeat").alias("nodeRepeat"),
    col("viz_result").get("Type").alias("type"),
    col("viz_result").get("Component").alias("component"),
    col("viz_result").get("Level").alias("level"),
    col("viz_result").get("Content").alias("content"),
    col("anomaly_result").get("Label").alias("label")
)

final.execute_insert('pg_sink')
