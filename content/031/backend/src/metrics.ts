import { metrics, Counter, Histogram } from '@opentelemetry/api';

const meter = metrics.getMeter('backend-api', '0.1.0');

export const replayChunksReceived: Counter = meter.createCounter('replay_chunks_received_total', {
  description: 'Total number of rrweb replay chunks received',
});

export const replayBytesTotal: Counter = meter.createCounter('replay_bytes_total', {
  description: 'Total bytes of rrweb replay payloads received (compressed)',
  unit: 'By',
});

export const interactionsCounter: Counter = meter.createCounter('interactions_total', {
  description: 'Total feed interactions received',
});

export const feedRequestDuration: Histogram = meter.createHistogram('feed_request_duration_ms', {
  description: 'Feed request duration in milliseconds',
  unit: 'ms',
});
