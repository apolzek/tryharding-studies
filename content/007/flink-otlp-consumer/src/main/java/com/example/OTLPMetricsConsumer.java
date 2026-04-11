package com.example.flink.otlp;

import io.opentelemetry.proto.collector.metrics.v1.ExportMetricsServiceRequest;
import io.opentelemetry.proto.metrics.v1.Metric;
import io.opentelemetry.proto.metrics.v1.NumberDataPoint;
import io.opentelemetry.proto.metrics.v1.ResourceMetrics;
import io.opentelemetry.proto.common.v1.KeyValue;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.sink.SinkFunction;
import org.apache.flink.api.common.functions.FlatMapFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import org.apache.flink.connector.prometheus.sink.PrometheusSink;
import org.apache.flink.connector.prometheus.sink.PrometheusSinkConfiguration.RetryConfiguration;
import org.apache.flink.connector.prometheus.sink.PrometheusSinkConfiguration.SinkWriterErrorHandlingBehaviorConfiguration;
import org.apache.flink.connector.prometheus.sink.PrometheusTimeSeries;
import org.apache.flink.connector.prometheus.sink.PrometheusTimeSeriesLabelsAndMetricNameKeySelector;
import static org.apache.flink.connector.prometheus.sink.PrometheusSinkConfiguration.OnErrorBehavior.DISCARD_AND_CONTINUE;
import org.apache.flink.api.connector.sink2.Sink;
import org.apache.flink.api.common.functions.MapFunction;

import java.io.IOException;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.Properties;
import java.util.List;
import java.util.stream.Collectors;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.regex.Pattern;
import java.util.ArrayList;
import java.util.concurrent.ThreadLocalRandom;

public class OTLPMetricsConsumer {
    
    private static final Logger LOGGER = LoggerFactory.getLogger(OTLPMetricsConsumer.class);
    
    private static final String KAFKA_BOOTSTRAP_SERVERS = "kafka:29092";
    private static final String KAFKA_TOPIC = "otel-metrics";
    private static final String KAFKA_GROUP_ID = "flink-otlp-consumer";
    private static final String PROMETHEUS_ENDPOINT = "http://prometheus:9090/api/v1/write";
    private static final int DEFAULT_MAX_RETRIES = 3;
    
    private static final int PARALLELISM = Math.max(1, Runtime.getRuntime().availableProcessors() / 2);

    public static void main(String[] args) throws Exception {
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        
        int actualParallelism = 1; // Start conservative
        
        if (args.length > 0) {
            try {
                actualParallelism = Integer.parseInt(args[0]);
                LOGGER.info("Using parallelism from command line argument: {}", actualParallelism);
            } catch (NumberFormatException e) {
                LOGGER.warn("Invalid parallelism argument '{}', using default: {}", args[0], actualParallelism);
            }
        } else {
            int availableCores = Runtime.getRuntime().availableProcessors();
            actualParallelism = Math.max(1, availableCores / 2);
            LOGGER.info("Auto-detected parallelism: {} (cores: {})", actualParallelism, availableCores);
        }
        
        try {
            env.setParallelism(actualParallelism);
        } catch (Exception e) {
            LOGGER.warn("Failed to set parallelism to {}, falling back to 1", actualParallelism);
            actualParallelism = 1;
            env.setParallelism(1);
        }
        
        env.getConfig().enableObjectReuse();
        env.setBufferTimeout(100);
        
        LOGGER.info("Starting OTLP Metrics Consumer with parallelism: {}", actualParallelism);
        
        KafkaSource<List<OTLPMetricData>> source = createKafkaSource();
        DataStream<List<OTLPMetricData>> metricsStream = env.fromSource(
            source, 
            org.apache.flink.api.common.eventtime.WatermarkStrategy.noWatermarks(), 
            "kafka-otlp-source"
        );

        DataStream<OTLPMetricData> flattenedStream = metricsStream
            .flatMap(new MetricsFlatMapper())
            .name("metrics-flattener");

        flattenedStream.addSink(new MetricsPrettyPrintSink());

        Properties prometheusProperties = createPrometheusProperties();
        
        flattenedStream
            .map(new OTLPToPrometheusMapper())
            .name("otlp-to-prometheus-mapper")
            .keyBy(new PrometheusTimeSeriesLabelsAndMetricNameKeySelector())
            .sinkTo(createPrometheusSink(prometheusProperties))
            .name("prometheus-sink")
            .uid("prometheus-sink");

        LOGGER.info("Executing Flink job with {} parallelism", actualParallelism);
        env.execute("OTLP Metrics Consumer - High Performance");
    }

    private static KafkaSource<List<OTLPMetricData>> createKafkaSource() {
        Properties kafkaProps = new Properties();
        kafkaProps.setProperty("fetch.min.bytes", "1048576");
        kafkaProps.setProperty("fetch.max.wait.ms", "100");
        kafkaProps.setProperty("max.partition.fetch.bytes", "10485760");
        kafkaProps.setProperty("receive.buffer.bytes", "1048576");
        
        return KafkaSource.<List<OTLPMetricData>>builder()
                .setBootstrapServers(KAFKA_BOOTSTRAP_SERVERS)
                .setTopics(KAFKA_TOPIC)
                .setGroupId(KAFKA_GROUP_ID)
                .setStartingOffsets(OffsetsInitializer.latest())
                .setValueOnlyDeserializer(new OptimizedOTLPMetricsDeserializer())
                .setProperties(kafkaProps)
                .build();
    }

    private static Properties createPrometheusProperties() {
        Properties properties = new Properties();
        properties.setProperty("endpoint.url", PROMETHEUS_ENDPOINT);
        properties.setProperty("max.request.retry", String.valueOf(DEFAULT_MAX_RETRIES));
        return properties;
    }

    private static Sink<PrometheusTimeSeries> createPrometheusSink(Properties properties) {
        String endpointUrl = properties.getProperty("endpoint.url");
        
        if (endpointUrl == null || endpointUrl.trim().isEmpty()) {
            throw new IllegalArgumentException("Endpoint URL not defined");
        }
        
        int maxRetries = Integer.parseInt(properties.getProperty("max.request.retry", String.valueOf(DEFAULT_MAX_RETRIES)));
        
        if (maxRetries <= 0) {
            throw new IllegalArgumentException("Max request retry must be greater than 0");
        }
        
        LOGGER.info("Configuring Prometheus sink with endpoint: {}", endpointUrl);
        
        return PrometheusSink.builder()
                .setPrometheusRemoteWriteUrl(endpointUrl)
                .setRetryConfiguration(RetryConfiguration.builder()
                        .setMaxRetryCount(maxRetries).build())
                .setErrorHandlingBehaviorConfiguration(SinkWriterErrorHandlingBehaviorConfiguration.builder()
                        .onMaxRetryExceeded(DISCARD_AND_CONTINUE)
                        .build())
                .setMetricGroupName("otlpMetrics")
                .build();
    }

    public static class MetricsFlatMapper implements FlatMapFunction<List<OTLPMetricData>, OTLPMetricData> {
        @Override
        public void flatMap(List<OTLPMetricData> metrics, Collector<OTLPMetricData> out) throws Exception {
            for (OTLPMetricData metric : metrics) {
                out.collect(metric);
            }
        }
    }

    public static class OTLPToPrometheusMapper implements MapFunction<OTLPMetricData, PrometheusTimeSeries> {
        
        private static final Map<String, String> METRIC_NAME_CACHE = new ConcurrentHashMap<>(1000);
        private static final Map<String, String> LABEL_NAME_CACHE = new ConcurrentHashMap<>(1000);
        private static final Pattern METRIC_NORMALIZE_PATTERN = Pattern.compile("[^a-zA-Z0-9_:]");
        private static final Pattern LABEL_NORMALIZE_PATTERN = Pattern.compile("[^a-zA-Z0-9_]");
        private static final Pattern METRIC_START_PATTERN = Pattern.compile("^[a-zA-Z_:].*");
        private static final Pattern LABEL_START_PATTERN = Pattern.compile("^[a-zA-Z_].*");
        
        private transient StringBuilder stringBuilder;
        
        @Override
        public PrometheusTimeSeries map(OTLPMetricData metric) throws Exception {
            if (metric.isError) {
                return createErrorTimeSeries();
            }

            String normalizedMetricName = normalizeMetricNameCached(metric.metricName);
            Map<String, String> labelsMap = buildLabelsMapOptimized(metric);
            double prometheusValue = parseValueToDoubleOptimized(metric.value);
            
            PrometheusTimeSeries.Label[] labels = convertToLabelsArray(labelsMap);
            PrometheusTimeSeries.Sample[] samples = {
                new PrometheusTimeSeries.Sample(prometheusValue, System.currentTimeMillis())
            };
            
            return new PrometheusTimeSeries(normalizedMetricName, labels, samples);
        }

        private PrometheusTimeSeries createErrorTimeSeries() {
            PrometheusTimeSeries.Label[] errorLabels = {
                new PrometheusTimeSeries.Label("error_type", "deserialization_error"),
                new PrometheusTimeSeries.Label("service_name", "otlp_consumer")
            };
            
            PrometheusTimeSeries.Sample[] errorSamples = {
                new PrometheusTimeSeries.Sample(1.0, System.currentTimeMillis())
            };
            
            return new PrometheusTimeSeries("otlp_processing_errors_total", errorLabels, errorSamples);
        }

        private Map<String, String> buildLabelsMapOptimized(OTLPMetricData metric) {
            Map<String, String> labelsMap = new HashMap<>(8);
            labelsMap.put("service_name", metric.serviceName);
            labelsMap.put("metric_type", normalizeMetricTypeOptimized(metric.metricType));
            
            if (isValidString(metric.unit)) {
                labelsMap.put("unit", metric.unit);
            }
            
            if (isValidString(metric.attributes)) {
                parseAttributesOptimized(metric.attributes, labelsMap);
            }
            
            return labelsMap;
        }

        private String normalizeMetricTypeOptimized(String metricType) {
            if (stringBuilder == null) {
                stringBuilder = new StringBuilder(256);
            }
            stringBuilder.setLength(0);
            for (int i = 0; i < metricType.length(); i++) {
                char c = metricType.charAt(i);
                if (c == ' ') {
                    stringBuilder.append('_');
                } else if (c != '(' && c != ')') {
                    stringBuilder.append(Character.toLowerCase(c));
                }
            }
            return stringBuilder.toString();
        }

        private PrometheusTimeSeries.Label[] convertToLabelsArray(Map<String, String> labelsMap) {
            return labelsMap.entrySet().stream()
                .map(entry -> new PrometheusTimeSeries.Label(entry.getKey(), entry.getValue()))
                .toArray(PrometheusTimeSeries.Label[]::new);
        }
        
        private String normalizeMetricNameCached(String name) {
            if (!isValidString(name)) {
                return "unknown_metric";
            }
            
            return METRIC_NAME_CACHE.computeIfAbsent(name, this::normalizeMetricNameDirect);
        }

        private String normalizeMetricNameDirect(String name) {
            String normalized = METRIC_NORMALIZE_PATTERN.matcher(name).replaceAll("_");
            if (!METRIC_START_PATTERN.matcher(normalized).matches()) {
                normalized = "_" + normalized;
            }
            return normalized;
        }

        private String normalizeLabelNameCached(String name) {
            if (!isValidString(name)) {
                return "unknown_label";
            }
            
            return LABEL_NAME_CACHE.computeIfAbsent(name, this::normalizeLabelNameDirect);
        }

        private String normalizeLabelNameDirect(String name) {
            String normalized = LABEL_NORMALIZE_PATTERN.matcher(name).replaceAll("_");
            if (!LABEL_START_PATTERN.matcher(normalized).matches()) {
                normalized = "_" + normalized;
            }
            return normalized;
        }
        
        private double parseValueToDoubleOptimized(String value) {
            if (!isValidString(value) || "N/A".equals(value)) {
                return 0.0;
            }
            
            try {
                return Double.parseDouble(value);
            } catch (NumberFormatException e) {
                return extractCountFromHistogramOptimized(value);
            }
        }

        private double extractCountFromHistogramOptimized(String value) {
            if (value.startsWith("Count: ")) {
                try {
                    int commaIndex = value.indexOf(',');
                    if (commaIndex > 7) {
                        String countPart = value.substring(7, commaIndex);
                        return Double.parseDouble(countPart);
                    }
                } catch (Exception ex) {
                    return 0.0;
                }
            }
            return 0.0;
        }
        
        private void parseAttributesOptimized(String attributesStr, Map<String, String> attributes) {
            if (!isValidString(attributesStr)) {
                return;
            }
            
            int start = 0;
            int length = attributesStr.length();
            
            while (start < length) {
                int equalsIndex = attributesStr.indexOf('=', start);
                if (equalsIndex == -1) break;
                
                int commaIndex = attributesStr.indexOf(',', equalsIndex);
                if (commaIndex == -1) commaIndex = length;
                
                String key = attributesStr.substring(start, equalsIndex).trim();
                String value = attributesStr.substring(equalsIndex + 1, commaIndex).trim();
                
                if (value.length() > 1 && value.charAt(0) == '"' && value.charAt(value.length() - 1) == '"') {
                    value = value.substring(1, value.length() - 1);
                }
                
                key = normalizeLabelNameCached(key);
                attributes.put(key, value);
                
                start = commaIndex + 1;
                while (start < length && attributesStr.charAt(start) == ' ') {
                    start++;
                }
            }
        }

        private boolean isValidString(String str) {
            return str != null && !str.isEmpty();
        }
    }

    public static class OTLPMetricData {
        public final String timestamp;
        public final String serviceName;
        public final String metricName;
        public final String metricType;
        public final String value;
        public final String attributes;
        public final String unit;
        public final String description;
        public final boolean isError;
        public final String errorMessage;
        public final String rawData;

        public OTLPMetricData(String timestamp, String serviceName, String metricName, 
                            String metricType, String value, String attributes, 
                            String unit, String description, boolean isError, String errorMessage) {
            this(timestamp, serviceName, metricName, metricType, value, attributes, 
                 unit, description, isError, errorMessage, null);
        }

        public OTLPMetricData(String timestamp, String serviceName, String metricName, 
                            String metricType, String value, String attributes, 
                            String unit, String description, boolean isError, 
                            String errorMessage, String rawData) {
            this.timestamp = timestamp;
            this.serviceName = serviceName;
            this.metricName = metricName;
            this.metricType = metricType;
            this.value = value;
            this.attributes = attributes;
            this.unit = unit;
            this.description = description;
            this.isError = isError;
            this.errorMessage = errorMessage;
            this.rawData = rawData;
        }
    }

    public static class OptimizedOTLPMetricsDeserializer implements DeserializationSchema<List<OTLPMetricData>> {
        
        private static final Logger LOGGER = LoggerFactory.getLogger(OptimizedOTLPMetricsDeserializer.class);
        private static final DateTimeFormatter FORMATTER = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss.SSS");
        
        private transient StringBuilder stringBuilder;
        private transient List<OTLPMetricData> metricsBuffer;

        @Override
        public List<OTLPMetricData> deserialize(byte[] message) throws IOException {
            if (stringBuilder == null) {
                stringBuilder = new StringBuilder(1024);
            }
            if (metricsBuffer == null) {
                metricsBuffer = new ArrayList<>(100);
            }
            
            long startTime = System.nanoTime();
            metricsBuffer.clear();
            
            try {
                if (message == null || message.length == 0) {
                    LOGGER.warn("Empty or null message received");
                    metricsBuffer.add(createErrorMetric("Empty or null message", null));
                    return new ArrayList<>(metricsBuffer);
                }

                LOGGER.debug("Processing message of {} bytes", message.length);
                
                ExportMetricsServiceRequest request = parseMessageOptimized(message);
                
                if (request == null) {
                    String hexDump = bytesToHexOptimized(message, 32);
                    metricsBuffer.add(createErrorMetric("Could not parse message as ExportMetricsServiceRequest", hexDump));
                    return new ArrayList<>(metricsBuffer);
                }
                
                LOGGER.debug("Request parsed successfully. ResourceMetrics count: {}", request.getResourceMetricsCount());
                
                processAllMetricsOptimized(request, metricsBuffer);
                
                if (metricsBuffer.isEmpty()) {
                    String hexDump = bytesToHexOptimized(message, 32);
                    metricsBuffer.add(createErrorMetric("No metrics found in message", hexDump));
                }
                
                long durationMicros = (System.nanoTime() - startTime) / 1_000;
                if (LOGGER.isDebugEnabled()) {
                    LOGGER.debug("Deserialization completed in {} μs. Metrics processed: {}", durationMicros, metricsBuffer.size());
                }
                
                return new ArrayList<>(metricsBuffer);
                
            } catch (Exception e) {
                LOGGER.error("Error deserializing OTLP message: {}", e.getMessage(), e);
                String hexDump = bytesToHexOptimized(message, 64);
                metricsBuffer.clear();
                metricsBuffer.add(createErrorMetric("Deserialization error: " + e.getMessage(), hexDump));
                return new ArrayList<>(metricsBuffer);
            }
        }

        private ExportMetricsServiceRequest parseMessageOptimized(byte[] message) {
            try {
                return ExportMetricsServiceRequest.parseFrom(message);
            } catch (Exception e1) {
                LOGGER.warn("Direct parsing failed: {}", e1.getMessage());
                
                try {
                    return ExportMetricsServiceRequest.newBuilder()
                        .mergeFrom(message)
                        .build();
                } catch (Exception e2) {
                    LOGGER.warn("Builder parsing failed: {}", e2.getMessage());
                    
                    String textContent = tryDecodeAsTextOptimized(message);
                    if (textContent != null) {
                        LOGGER.warn("Message appears to be text/JSON, not protobuf");
                    }
                    
                    return null;
                }
            }
        }

        private void processAllMetricsOptimized(ExportMetricsServiceRequest request, List<OTLPMetricData> outputMetrics) {
            for (ResourceMetrics resourceMetrics : request.getResourceMetricsList()) {
                String serviceName = extractServiceNameOptimized(resourceMetrics);
                
                for (var scopeMetrics : resourceMetrics.getScopeMetricsList()) {
                    for (Metric metric : scopeMetrics.getMetricsList()) {
                        OTLPMetricData processedMetric = processMetricOptimized(metric, serviceName);
                        outputMetrics.add(processedMetric);
                    }
                }
            }
        }

        private String extractServiceNameOptimized(ResourceMetrics resourceMetrics) {
            if (resourceMetrics.hasResource()) {
                for (KeyValue attr : resourceMetrics.getResource().getAttributesList()) {
                    if ("service.name".equals(attr.getKey())) {
                        return attr.getValue().getStringValue();
                    }
                }
            }
            return "unknown-service";
        }

        private OTLPMetricData processMetricOptimized(Metric metric, String serviceName) {
            String timestamp = formatCurrentTimeOptimized();
            
            try {
                if (metric.hasGauge()) {
                    return processGaugeMetricOptimized(metric, serviceName, timestamp);
                } else if (metric.hasSum()) {
                    return processSumMetricOptimized(metric, serviceName, timestamp);
                } else if (metric.hasHistogram()) {
                    return processHistogramMetricOptimized(metric, serviceName, timestamp);
                } else if (metric.hasExponentialHistogram()) {
                    return processExponentialHistogramMetricOptimized(metric, serviceName, timestamp);
                } else if (metric.hasSummary()) {
                    return processSummaryMetricOptimized(metric, serviceName, timestamp);
                } else {
                    return new OTLPMetricData(timestamp, serviceName, metric.getName(), "unknown", 
                                            "N/A", "", metric.getUnit(), metric.getDescription(), false, null);
                }
            } catch (Exception e) {
                LOGGER.warn("Error processing metric '{}': {}", metric.getName(), e.getMessage());
                return new OTLPMetricData(timestamp, serviceName, metric.getName(), "error", 
                                        "N/A", "", metric.getUnit(), metric.getDescription(), true, 
                                        "Processing error: " + e.getMessage());
            }
        }

        private OTLPMetricData processGaugeMetricOptimized(Metric metric, String serviceName, String timestamp) {
            if (!metric.getGauge().getDataPointsList().isEmpty()) {
                NumberDataPoint dataPoint = metric.getGauge().getDataPointsList().get(0);
                String value = extractValueOptimized(dataPoint);
                String attributes = extractAttributesOptimized(dataPoint.getAttributesList());
                
                return new OTLPMetricData(timestamp, serviceName, metric.getName(), "Gauge", 
                                        value, attributes, metric.getUnit(), metric.getDescription(), false, null);
            }
            return createErrorMetric("Gauge without data points", null);
        }

        private OTLPMetricData processSumMetricOptimized(Metric metric, String serviceName, String timestamp) {
            if (!metric.getSum().getDataPointsList().isEmpty()) {
                NumberDataPoint dataPoint = metric.getSum().getDataPointsList().get(0);
                String value = extractValueOptimized(dataPoint);
                String attributes = extractAttributesOptimized(dataPoint.getAttributesList());
                String metricType = "Sum" + (metric.getSum().getIsMonotonic() ? " (Monotonic)" : " (Non-Monotonic)");
                
                return new OTLPMetricData(timestamp, serviceName, metric.getName(), metricType, 
                                        value, attributes, metric.getUnit(), metric.getDescription(), false, null);
            }
            return createErrorMetric("Sum without data points", null);
        }

        private OTLPMetricData processHistogramMetricOptimized(Metric metric, String serviceName, String timestamp) {
            if (!metric.getHistogram().getDataPointsList().isEmpty()) {
                var histogramPoint = metric.getHistogram().getDataPointsList().get(0);
                if (stringBuilder == null) {
                    stringBuilder = new StringBuilder(1024);
                }
                stringBuilder.setLength(0);
                stringBuilder.append("Count: ").append(histogramPoint.getCount())
                  .append(", Sum: ").append(String.format("%.2f", histogramPoint.getSum()));
                String value = stringBuilder.toString();
                String attributes = extractAttributesOptimized(histogramPoint.getAttributesList());
                
                return new OTLPMetricData(timestamp, serviceName, metric.getName(), "Histogram", 
                                        value, attributes, metric.getUnit(), metric.getDescription(), false, null);
            }
            return createErrorMetric("Histogram without data points", null);
        }

        private OTLPMetricData processExponentialHistogramMetricOptimized(Metric metric, String serviceName, String timestamp) {
            if (!metric.getExponentialHistogram().getDataPointsList().isEmpty()) {
                var expHistPoint = metric.getExponentialHistogram().getDataPointsList().get(0);
                if (stringBuilder == null) {
                    stringBuilder = new StringBuilder(1024);
                }
                stringBuilder.setLength(0);
                stringBuilder.append("Count: ").append(expHistPoint.getCount())
                  .append(", Sum: ").append(String.format("%.2f", expHistPoint.getSum()))
                  .append(", Scale: ").append(expHistPoint.getScale());
                String value = stringBuilder.toString();
                String attributes = extractAttributesOptimized(expHistPoint.getAttributesList());
                
                return new OTLPMetricData(timestamp, serviceName, metric.getName(), "ExponentialHistogram", 
                                        value, attributes, metric.getUnit(), metric.getDescription(), false, null);
            }
            return createErrorMetric("ExponentialHistogram without data points", null);
        }

        private OTLPMetricData processSummaryMetricOptimized(Metric metric, String serviceName, String timestamp) {
            if (!metric.getSummary().getDataPointsList().isEmpty()) {
                var summaryPoint = metric.getSummary().getDataPointsList().get(0);
                if (stringBuilder == null) {
                    stringBuilder = new StringBuilder(1024);
                }
                stringBuilder.setLength(0);
                stringBuilder.append("Count: ").append(summaryPoint.getCount())
                  .append(", Sum: ").append(String.format("%.2f", summaryPoint.getSum()));
                String value = stringBuilder.toString();
                String attributes = extractAttributesOptimized(summaryPoint.getAttributesList());
                
                return new OTLPMetricData(timestamp, serviceName, metric.getName(), "Summary", 
                                        value, attributes, metric.getUnit(), metric.getDescription(), false, null);
            }
            return createErrorMetric("Summary without data points", null);
        }

        private String extractValueOptimized(NumberDataPoint dataPoint) {
            if (dataPoint.hasAsDouble()) {
                return String.format("%.6f", dataPoint.getAsDouble());
            } else if (dataPoint.hasAsInt()) {
                return String.valueOf(dataPoint.getAsInt());
            }
            return "N/A";
        }

        private String extractAttributesOptimized(List<KeyValue> attributesList) {
            if (attributesList.isEmpty()) {
                return "";
            }
            
            if (stringBuilder == null) {
                stringBuilder = new StringBuilder(1024);
            }
            stringBuilder.setLength(0);
            
            boolean first = true;
            for (KeyValue attr : attributesList) {
                if (!first) {
                    stringBuilder.append(", ");
                }
                stringBuilder.append(attr.getKey()).append("=").append(getAttributeValueOptimized(attr));
                first = false;
            }
            
            return stringBuilder.toString();
        }

        private String getAttributeValueOptimized(KeyValue attr) {
            if (attr.getValue().hasStringValue()) {
                return "\"" + attr.getValue().getStringValue() + "\"";
            } else if (attr.getValue().hasIntValue()) {
                return String.valueOf(attr.getValue().getIntValue());
            } else if (attr.getValue().hasBoolValue()) {
                return String.valueOf(attr.getValue().getBoolValue());
            } else if (attr.getValue().hasDoubleValue()) {
                return String.valueOf(attr.getValue().getDoubleValue());
            } else if (attr.getValue().hasArrayValue()) {
                return "[array]";
            } else if (attr.getValue().hasKvlistValue()) {
                return "{kvlist}";
            } else if (attr.getValue().hasBytesValue()) {
                return "[bytes]";
            }
            return "unknown";
        }

        private String tryDecodeAsTextOptimized(byte[] message) {
            try {
                String text = new String(message, 0, Math.min(message.length, 1000), StandardCharsets.UTF_8);
                boolean isText = true;
                for (int i = 0; i < text.length(); i++) {
                    char c = text.charAt(i);
                    if (!((c >= 32 && c <= 126) || Character.isWhitespace(c))) {
                        isText = false;
                        break;
                    }
                }
                return isText ? text : null;
            } catch (Exception e) {
                return null;
            }
        }

        private String bytesToHexOptimized(byte[] bytes, int maxBytes) {
            if (bytes == null) return "null";
            
            if (stringBuilder == null) {
                stringBuilder = new StringBuilder(1024);
            }
            stringBuilder.setLength(0);
            int limit = Math.min(bytes.length, maxBytes);
            
            for (int i = 0; i < limit; i++) {
                stringBuilder.append(String.format("%02x ", bytes[i]));
                if (i > 0 && (i + 1) % 16 == 0) {
                    stringBuilder.append("\n");
                }
            }
            if (bytes.length > maxBytes) {
                stringBuilder.append("... (truncated)");
            }
            return stringBuilder.toString();
        }

        private String formatCurrentTimeOptimized() {
            return LocalDateTime.now().format(FORMATTER);
        }

        private OTLPMetricData createErrorMetric(String errorMsg, String rawData) {
            String timestamp = formatCurrentTimeOptimized();
            return new OTLPMetricData(timestamp, "error", "error", "ERROR", 
                                    "N/A", "", "", "", true, errorMsg, rawData);
        }

        @Override
        public boolean isEndOfStream(List<OTLPMetricData> nextElement) {
            return false;
        }

        @Override
        public TypeInformation<List<OTLPMetricData>> getProducedType() {
            return TypeInformation.of((Class<List<OTLPMetricData>>) (Class<?>) List.class);
        }
    }

    public static class MetricsPrettyPrintSink implements SinkFunction<OTLPMetricData> {
        
        private static final Logger LOGGER = LoggerFactory.getLogger(MetricsPrettyPrintSink.class);
        private static final String SEPARATOR = "=".repeat(80);
        private static final String LINE = "-".repeat(80);
        
        private long processedCount = 0;
        private long lastLogTime = System.currentTimeMillis();
        private static final long LOG_INTERVAL_MS = 5000;

        @Override
        public void invoke(OTLPMetricData metric, Context context) throws Exception {
            processedCount++;
            
            long currentTime = System.currentTimeMillis();
            if (currentTime - lastLogTime >= LOG_INTERVAL_MS) {
                LOGGER.info("Processed {} metrics in the last {} seconds", 
                          processedCount, (currentTime - lastLogTime) / 1000.0);
                processedCount = 0;
                lastLogTime = currentTime;
            }
            printMetric(metric);            
            // if (ThreadLocalRandom.current().nextInt(1000) == 0) {
            //     if (metric.isError) {
            //         printError(metric);
            //     } else {
            //         printMetric(metric);
            //     }
            // }
        }

        private void printError(OTLPMetricData metric) {
            System.err.println(SEPARATOR);
            System.err.println("PROCESSING ERROR");
            System.err.println(LINE);
            System.err.println("Timestamp: " + metric.timestamp);
            System.err.println("Error: " + metric.errorMessage);
            
            if (metric.rawData != null) {
                System.err.println("Raw Data (hex):");
                System.err.println(metric.rawData);
            }
            
            System.err.println(SEPARATOR);
            System.err.println();
        }

        private void printMetric(OTLPMetricData metric) {
            System.out.println(SEPARATOR);
            System.out.println("NEW OTLP METRIC (SAMPLED)");
            System.out.println(LINE);
            System.out.println("Timestamp: " + metric.timestamp);
            System.out.println("Service: " + metric.serviceName);
            System.out.println("Name: " + metric.metricName);
            System.out.println("Type: " + metric.metricType);
            System.out.println("Unit: " + (metric.unit.isEmpty() ? "N/A" : metric.unit));
            System.out.println("Description: " + (metric.description.isEmpty() ? "N/A" : metric.description));
            System.out.println("Value: " + metric.value);
            
            if (!metric.attributes.isEmpty()) {
                System.out.println("Attributes: " + metric.attributes);
            }
            
            System.out.println("SENDING TO PROMETHEUS via Remote Write");
            System.out.println(SEPARATOR);
            System.out.println();
            
            LOGGER.info("Metric processed and sent to Prometheus: {} = {} [{}] from service '{}'", 
                    metric.metricName, metric.value, metric.metricType, metric.serviceName);
        }
    }
}