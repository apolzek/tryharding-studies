/**
 * Auto-instrumentation bootstrap.
 * Loaded via `node -r ./otel.cjs`, before any app module is imported,
 * so every supported library (http, fastify, grpc, etc.) gets patched.
 */
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-grpc');
const { Resource } = require('@opentelemetry/resources');
const { SemanticResourceAttributes } = require('@opentelemetry/semantic-conventions');

const sdk = new NodeSDK({
  resource: new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: process.env.SERVICE_NAME || 'bff',
  }),
  traceExporter: new OTLPTraceExporter({
    url: process.env.OTEL_EXPORTER_OTLP_ENDPOINT || 'http://otel-collector:4317',
  }),
  instrumentations: [getNodeAutoInstrumentations()],
});

try {
  sdk.start();
} catch (e) {
  console.error('otel bootstrap failed', e);
}

process.on('SIGTERM', () => sdk.shutdown().catch(() => {}));
