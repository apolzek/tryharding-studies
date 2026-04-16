# OtlpCounterJob — Code Review & Explanation

## What this job does

`OtlpCounterJob` is the Flink job that feeds the `otlp-counter` Grafana dashboard.

It reads raw OTLP protobuf batches from 6 Kafka topics (`otlp-{traces,logs,metrics}-{grpc,http}`),
counts messages and bytes per `(telemetry_type, protocol)` pair, and pushes two monotonically
increasing counters to Prometheus via Remote Write every 10 seconds:

| Metric | Labels | Description |
|---|---|---|
| `otlp_telemetry_messages_total` | `telemetry_type`, `protocol` | Cumulative OTLP batch count |
| `otlp_telemetry_bytes_total` | `telemetry_type`, `protocol` | Cumulative raw byte count |
| `flink_job_alive` | `job` | Heartbeat — absence signals job is down |

---

## Architecture

```
Kafka [traces/grpc]  ─┐
Kafka [traces/http]  ─┤
Kafka [logs/grpc]    ─┤─→ union → keyBy(type_protocol) → CumulativeCounterFunction → PrometheusSink
Kafka [logs/http]    ─┤
Kafka [metrics/grpc] ─┤
Kafka [metrics/http] ─┘
```

- **6 independent sources** unioned into a single stream.
- **Stateful keyed aggregation** — Flink `ValueState` accumulates totals per key (6 keys total).
- **Processing-time timer** fires every 10 s per key and emits the current cumulative value.
- **Prometheus Remote Write** receives counter samples; Grafana queries them with `rate()`.

---

## Code Review

### Issues found

#### 1. Key decoding in `onTimer` is fragile

```java
// onTimer — OtlpCounterJob.java:205-208
String key    = ctx.getCurrentKey();   // e.g. "traces_grpc"
int    sep    = key.lastIndexOf('_');
String sigType  = key.substring(0, sep);
String protocol = key.substring(sep + 1);
```

The composite key `type + "_" + protocol` is decoded by splitting on the last `_`.
This works today because the only values are `traces`, `logs`, `metrics` and `grpc`, `http`.
But if a future signal type contained an underscore this would silently produce wrong labels.

**Recommendation:** store `type` and `protocol` in dedicated `ValueState<String>` fields
(as `OtlpAnalyticsInsightsJob` correctly does with `metricNameState` / `labelsState`).

---

#### 2. No restart strategy configured

```java
final StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
// missing: env.setRestartStrategy(...)
```

Without an explicit restart strategy Flink defaults to the cluster-level setting, which in
this docker-compose setup is "no restart" (the job dies on the first unhandled exception,
e.g. a transient Prometheus Remote Write failure).

**Recommendation:**
```java
env.setRestartStrategy(
    RestartStrategies.fixedDelayRestart(3, Time.seconds(10))
);
```

---

#### 3. `JobManagerCheckpointStorage` loses state on JobManager restart

```java
env.getCheckpointConfig().setCheckpointStorage(new JobManagerCheckpointStorage());
```

Checkpoints are stored in the JobManager's heap. If the JM process restarts, all
accumulated counter state is lost and the counters reset to 0. This shows up in Grafana
as a counter reset (sudden drop followed by monotonic growth from zero).

**Recommendation for production:** use `FileSystemCheckpointStorage` pointing to a
persistent volume or object store.

```java
env.getCheckpointConfig().setCheckpointStorage("file:///opt/flink/checkpoints");
```

---

#### 4. Configuration is hardcoded

```java
private static final String KAFKA_BOOTSTRAP_SERVERS    = "kafka:29092";
private static final String PROMETHEUS_REMOTE_WRITE_URL = "http://prometheus:9090/api/v1/write";
```

These tie the JAR to a specific environment. The JAR cannot be reused across
dev / staging / prod without recompilation.

**Recommendation:** read from `ParameterTool`:
```java
ParameterTool params = ParameterTool.fromArgs(args);
String kafkaBootstrap = params.get("kafka.bootstrap", "kafka:29092");
String prometheusUrl  = params.get("prometheus.url",  "http://prometheus:9090/api/v1/write");
```

---

#### 5. Parallelism is hardcoded

```java
env.setParallelism(2);
```

Same problem as above — parallelism should be driven by the deployment environment.
For a 3-partition topic with 6 sources, parallelism 2 means some source subtasks consume
more than one partition, which is acceptable but not optimal.

**Recommendation:** remove `setParallelism` from the job and set it at submission time:
```bash
curl -X POST http://flink-jobmanager:8081/jars/<id>/run \
  -d '{"parallelism": 3}'
```

---

#### 6. Heartbeat emits one series per key (6 redundant series)

```java
// Added in onTimer for all 6 keys
out.collect(PrometheusTimeSeries.builder()
    .withMetricName("flink_job_alive")
    .addLabel("job", "otlp-counter")
    .addSample(1L, now)
    .build());
```

The heartbeat has no label distinguishing the keys, so Prometheus receives 6 identical
series and deduplicates them to 1 — wasteful but not harmful.

**Recommendation:** emit the heartbeat only from a single designated key (e.g. `traces_grpc`)
or from a dedicated non-keyed `ProcessFunction` on a side stream.

---

#### 7. `ByteArrayDeserializationSchema` reimplements existing Flink API

The custom deserializer just returns the raw `byte[]` unchanged.
Flink provides `ByteArrayDeserializationSchema` in `flink-connector-kafka` already.
Using the built-in avoids maintaining dead code.

---

### What is done well

- **`union()` over 6 sources** — idiomatic Flink pattern; sources stay independent until the
  keyed aggregation, preserving parallelism.
- **Processing-time timer registered once, re-registered in `onTimer`** — avoids timer leaks
  and is the standard Flink pattern for periodic emission.
- **Monotonically increasing counters** — correct semantics for Prometheus counters;
  `rate()` and `increase()` work correctly.
- **Null-safe state reads** — `value() == null ? 0L : value()` guards the first-run case
  before any state is written.
- **`WatermarkStrategy.noWatermarks()`** — correct for a processing-time-only job.
- **`OffsetsInitializer.latest()`** — only applies on the very first run (no checkpoint).
  On restart from checkpoint, the Flink Kafka connector uses the checkpointed offsets,
  not this initializer.

---

## Removed: `otlp-analytics-insights`

`OtlpAnalyticsInsightsJob` was present in the repository but not wired to any dashboard
or downstream consumer. Its metrics (`otlp_messages_total`, `otlp_service_messages_total`,
`otlp_span_error_total`, etc.) are not queried anywhere.

The job was removed to reduce maintenance surface. If deep per-service or attribute
cardinality analytics are needed in the future, the code can be recovered from git history.
