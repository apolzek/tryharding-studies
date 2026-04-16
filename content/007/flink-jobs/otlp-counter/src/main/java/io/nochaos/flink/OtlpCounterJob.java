package io.nochaos.flink;

import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeHint;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.runtime.state.storage.JobManagerCheckpointStorage;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.connector.prometheus.sink.PrometheusSink;
import org.apache.flink.connector.prometheus.sink.PrometheusTimeSeries;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;

import io.opentelemetry.proto.collector.trace.v1.ExportTraceServiceRequest;
import java.io.IOException;

/**
 * Flink job that counts OTLP telemetry from Kafka, tagged by signal type
 * (traces / logs / metrics).
 *
 * <p>The OTel Collector L1 writes to three Kafka topics:
 * <pre>
 *   otlp-traces
 *   otlp-logs
 *   otlp-metrics
 * </pre>
 *
 * <p>Metrics exported (monotonically increasing counters via Prometheus Remote Write):
 * <pre>
 *   flink_otlp_messages_total{telemetry_type}
 *   flink_otlp_bytes_total{telemetry_type}
 *   flink_otlp_spans_total{telemetry_type}
 * </pre>
 *
 * <p>With the telemetry_type label, PromQL can derive every requested aggregate:
 * <ul>
 *   <li>Total all:   {@code sum(flink_otlp_messages_total)}</li>
 *   <li>Per type:    {@code sum by (telemetry_type)(flink_otlp_messages_total)}</li>
 *   <li>Traces only: {@code flink_otlp_messages_total{telemetry_type="traces"}}</li>
 * </ul>
 *
 * <p>State is checkpointed so counters survive Flink restarts.
 * Values are emitted every {@value #EMIT_INTERVAL_MS} ms via processing-time timers.
 */
public class OtlpCounterJob {

    private static final String KAFKA_BOOTSTRAP_SERVERS = "kafka:29092";
    private static final String PROMETHEUS_REMOTE_WRITE_URL = "http://prometheus:9090/api/v1/write";
    private static final long EMIT_INTERVAL_MS = 10_000L;

    private static final String[] SIGNAL_TYPES = {"traces", "logs", "metrics"};

    public static void main(String[] args) throws Exception {
        final StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(2);
        env.enableCheckpointing(10_000L);
        env.getCheckpointConfig().setCheckpointStorage(new JobManagerCheckpointStorage());

        // Build one Kafka source per signal type and union all streams
        DataStream<TelemetryCount> allStreams = null;

        for (String type : SIGNAL_TYPES) {
            String topic   = "otlp-" + type;
            String groupId = "flink-otlp-counter-" + type;

            DataStream<TelemetryCount> stream = env
                .fromSource(
                    createKafkaSource(topic, groupId),
                    WatermarkStrategy.noWatermarks(),
                    "Kafka [" + type + "]")
                .map(new TagTelemetry(type));

            allStreams = (allStreams == null) ? stream : allStreams.union(stream);
        }

        var prometheusSink = PrometheusSink.builder()
            .setPrometheusRemoteWriteUrl(PROMETHEUS_REMOTE_WRITE_URL)
            .build();

        // Key by signal type → 3 independent state cells
        allStreams
            .keyBy(tc -> tc.type)
            .process(new CumulativeCounterFunction())
            .sinkTo(prometheusSink);

        env.execute("OTLP Telemetry Counter");
    }

    // -------------------------------------------------------------------------
    // Kafka source factory
    // -------------------------------------------------------------------------

    private static KafkaSource<byte[]> createKafkaSource(String topic, String groupId) {
        return KafkaSource.<byte[]>builder()
            .setBootstrapServers(KAFKA_BOOTSTRAP_SERVERS)
            .setTopics(topic)
            .setGroupId(groupId)
            .setStartingOffsets(OffsetsInitializer.latest())
            .setValueOnlyDeserializer(new ByteArrayDeserializationSchema())
            .build();
    }

    // -------------------------------------------------------------------------
    // Data model
    // -------------------------------------------------------------------------

    public static class TelemetryCount {
        public String type;   // traces | logs | metrics
        public long   count;  // number of Kafka messages (= OTLP batches)
        public long   bytes;  // raw payload size in bytes
        public long   spans;  // individual span count (traces only)

        public TelemetryCount() {}

        public TelemetryCount(String type, long count, long bytes, long spans) {
            this.type  = type;
            this.count = count;
            this.bytes = bytes;
            this.spans = spans;
        }
    }

    // -------------------------------------------------------------------------
    // Map: raw bytes → tagged TelemetryCount
    // -------------------------------------------------------------------------

    public static class TagTelemetry implements MapFunction<byte[], TelemetryCount> {
        private final String type;

        public TagTelemetry(String type) {
            this.type = type;
        }

        @Override
        public TelemetryCount map(byte[] value) {
            long spans = 0L;
            if ("traces".equals(type) && value != null) {
                try {
                    spans = ExportTraceServiceRequest.parseFrom(value)
                        .getResourceSpansList().stream()
                        .flatMap(rs -> rs.getScopeSpansList().stream())
                        .mapToLong(ss -> ss.getSpansList().size())
                        .sum();
                } catch (Exception ignored) {}
            }
            return new TelemetryCount(type, 1L, value != null ? value.length : 0L, spans);
        }
    }

    // -------------------------------------------------------------------------
    // Stateful process: accumulate totals and emit on a fixed timer
    // -------------------------------------------------------------------------

    /**
     * Maintains monotonically increasing cumulative totals per signal type key.
     * Emits three Prometheus time series every {@value OtlpCounterJob#EMIT_INTERVAL_MS} ms:
     * <ul>
     *   <li>{@code flink_otlp_messages_total{telemetry_type}}</li>
     *   <li>{@code flink_otlp_bytes_total{telemetry_type}}</li>
     *   <li>{@code flink_otlp_spans_total{telemetry_type}}</li>
     * </ul>
     */
    public static class CumulativeCounterFunction
            extends KeyedProcessFunction<String, TelemetryCount, PrometheusTimeSeries> {

        private transient ValueState<Long> totalMessagesState;
        private transient ValueState<Long> totalBytesState;
        private transient ValueState<Long> totalSpansState;
        private transient ValueState<Long> timerState;

        @Override
        public void open(Configuration parameters) {
            totalMessagesState = getRuntimeContext().getState(
                new ValueStateDescriptor<>("totalMessages", Long.class));
            totalBytesState = getRuntimeContext().getState(
                new ValueStateDescriptor<>("totalBytes", Long.class));
            totalSpansState = getRuntimeContext().getState(
                new ValueStateDescriptor<>("totalSpans", Long.class));
            timerState = getRuntimeContext().getState(
                new ValueStateDescriptor<>("timerTs", Long.class));
        }

        @Override
        public void processElement(TelemetryCount value, Context ctx, Collector<PrometheusTimeSeries> out)
                throws Exception {
            long messages = totalMessagesState.value() == null ? 0L : totalMessagesState.value();
            long bytes    = totalBytesState.value()    == null ? 0L : totalBytesState.value();
            long spans    = totalSpansState.value()    == null ? 0L : totalSpansState.value();

            totalMessagesState.update(messages + value.count);
            totalBytesState.update(bytes + value.bytes);
            totalSpansState.update(spans + value.spans);

            // Register the emit timer only once per key; onTimer re-registers it
            if (timerState.value() == null) {
                long nextFire = ctx.timerService().currentProcessingTime() + EMIT_INTERVAL_MS;
                ctx.timerService().registerProcessingTimeTimer(nextFire);
                timerState.update(nextFire);
            }
        }

        @Override
        public void onTimer(long timestamp, OnTimerContext ctx, Collector<PrometheusTimeSeries> out)
                throws Exception {
            long totalMessages = totalMessagesState.value() == null ? 0L : totalMessagesState.value();
            long totalBytes    = totalBytesState.value()    == null ? 0L : totalBytesState.value();
            long totalSpans    = totalSpansState.value()    == null ? 0L : totalSpansState.value();

            String sigType = ctx.getCurrentKey(); // traces | logs | metrics
            long   now     = System.currentTimeMillis();

            out.collect(PrometheusTimeSeries.builder()
                .withMetricName("flink_otlp_messages_total")
                .addLabel("telemetry_type", sigType)
                .addSample(totalMessages, now)
                .build());

            out.collect(PrometheusTimeSeries.builder()
                .withMetricName("flink_otlp_bytes_total")
                .addLabel("telemetry_type", sigType)
                .addSample(totalBytes, now)
                .build());

            out.collect(PrometheusTimeSeries.builder()
                .withMetricName("flink_otlp_spans_total")
                .addLabel("telemetry_type", sigType)
                .addSample(totalSpans, now)
                .build());

            // Heartbeat: value 1 emitted every cycle — absence signals job is down
            out.collect(PrometheusTimeSeries.builder()
                .withMetricName("flink_job_alive")
                .addLabel("job", "otlp-counter")
                .addSample(1L, now)
                .build());

            // Schedule next emit
            long nextFire = timestamp + EMIT_INTERVAL_MS;
            ctx.timerService().registerProcessingTimeTimer(nextFire);
            timerState.update(nextFire);
        }
    }

    // -------------------------------------------------------------------------
    // Kafka deserializer: pass raw bytes through unchanged
    // -------------------------------------------------------------------------

    public static class ByteArrayDeserializationSchema implements DeserializationSchema<byte[]> {
        @Override
        public byte[] deserialize(byte[] message) throws IOException {
            return message;
        }

        @Override
        public boolean isEndOfStream(byte[] nextElement) {
            return false;
        }

        @Override
        public TypeInformation<byte[]> getProducedType() {
            return TypeInformation.of(new TypeHint<byte[]>() {});
        }
    }
}
