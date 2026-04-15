import { v4 as uuidv4 } from 'uuid';
import { WebTracerProvider, BatchSpanProcessor } from '@opentelemetry/sdk-trace-web';
import { Resource } from '@opentelemetry/resources';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-http';
import { OTLPLogExporter } from '@opentelemetry/exporter-logs-otlp-http';
import { MeterProvider, PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics';
import { LoggerProvider, BatchLogRecordProcessor } from '@opentelemetry/sdk-logs';
import { logs } from '@opentelemetry/api-logs';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { UserInteractionInstrumentation } from '@opentelemetry/instrumentation-user-interaction';
import { DocumentLoadInstrumentation } from '@opentelemetry/instrumentation-document-load';
import { metrics } from '@opentelemetry/api';
import { initializeFaro, getWebInstrumentations } from '@grafana/faro-web-sdk';
import { TracingInstrumentation } from '@grafana/faro-web-tracing';
import * as rrweb from 'rrweb';
import { onLCP, onCLS, onINP, onFCP, onTTFB } from 'web-vitals';

const SERVICE_NAME = 'frontend-vanilla';
const SERVICE_VERSION = '0.1.0';
const STACK = 'vanilla';

const OTLP_ENDPOINT = import.meta.env.VITE_OTLP_ENDPOINT;
const FARO_URL = import.meta.env.VITE_FARO_URL;
const REPLAY_ENDPOINT = import.meta.env.VITE_REPLAY_ENDPOINT;

function getOrCreate(storage, key) {
  let v = storage.getItem(key);
  if (!v) {
    v = uuidv4();
    storage.setItem(key, v);
  }
  return v;
}

export const SESSION_ID = getOrCreate(sessionStorage, 'attentium.session_id');
export const USER_ID = getOrCreate(localStorage, 'attentium.user_id');
export const STACK_NAME = STACK;

let postInteractionsCounter;
let postDwellHistogram;
let scrollDepthGauge;
const webVitalGauges = {};

export function getMeters() {
  return { postInteractionsCounter, postDwellHistogram, scrollDepthGauge, webVitalGauges };
}

export function emitLog(body) {
  try {
    const logger = logs.getLogger(SERVICE_NAME);
    logger.emit({
      severityNumber: 9,
      severityText: 'INFO',
      body: JSON.stringify(body),
      attributes: { ...body, 'frontend.stack': STACK, 'session.id': SESSION_ID },
    });
  } catch (e) {
    console.warn('[telemetry] emitLog failed', e);
  }
}

export function initTelemetry() {
  try {
    const resource = new Resource({
      [SemanticResourceAttributes.SERVICE_NAME]: SERVICE_NAME,
      [SemanticResourceAttributes.SERVICE_VERSION]: SERVICE_VERSION,
      [SemanticResourceAttributes.DEPLOYMENT_ENVIRONMENT]: 'poc',
      'session.id': SESSION_ID,
      'user.id': USER_ID,
      'frontend.stack': STACK,
    });

    const tracerProvider = new WebTracerProvider({ resource });
    tracerProvider.addSpanProcessor(
      new BatchSpanProcessor(new OTLPTraceExporter({ url: `${OTLP_ENDPOINT}/v1/traces` }))
    );
    tracerProvider.register({ contextManager: new ZoneContextManager() });

    registerInstrumentations({
      instrumentations: [
        new FetchInstrumentation({ propagateTraceHeaderCorsUrls: [/.*/] }),
        new UserInteractionInstrumentation(),
        new DocumentLoadInstrumentation(),
      ],
    });

    const meterProvider = new MeterProvider({
      resource,
      readers: [
        new PeriodicExportingMetricReader({
          exporter: new OTLPMetricExporter({ url: `${OTLP_ENDPOINT}/v1/metrics` }),
          exportIntervalMillis: 10000,
        }),
      ],
    });
    metrics.setGlobalMeterProvider(meterProvider);

    const meter = metrics.getMeter(SERVICE_NAME);
    postInteractionsCounter = meter.createCounter('post_interactions_total');
    postDwellHistogram = meter.createHistogram('post_dwell_time_ms');
    scrollDepthGauge = meter.createHistogram('scroll_depth_max');
    ['lcp', 'cls', 'inp', 'fcp', 'ttfb'].forEach((name) => {
      webVitalGauges[name] = meter.createHistogram(`web_vitals_${name}`);
    });

    const loggerProvider = new LoggerProvider({ resource });
    loggerProvider.addLogRecordProcessor(
      new BatchLogRecordProcessor(new OTLPLogExporter({ url: `${OTLP_ENDPOINT}/v1/logs` }))
    );
    logs.setGlobalLoggerProvider(loggerProvider);

    try {
      initializeFaro({
        url: FARO_URL,
        app: { name: SERVICE_NAME, version: SERVICE_VERSION, environment: 'poc' },
        sessionTracking: { enabled: true, session: { id: SESSION_ID } },
        instrumentations: [...getWebInstrumentations(), new TracingInstrumentation()],
      });
    } catch (e) {
      console.warn('[telemetry] Faro init failed', e);
    }

    initWebVitals();
    initRrweb();
    initScrollTracking();
  } catch (e) {
    console.warn('[telemetry] init failed, app continues', e);
  }
}

function initWebVitals() {
  const record = (name) => (m) => {
    try {
      webVitalGauges[name]?.record(m.value, { 'frontend.stack': STACK });
    } catch (e) {
      console.warn('[telemetry] web-vital record', name, e);
    }
  };
  onLCP(record('lcp'));
  onCLS(record('cls'));
  onINP(record('inp'));
  onFCP(record('fcp'));
  onTTFB(record('ttfb'));
}

let scrollDepthMax = 0;
function initScrollTracking() {
  const compute = () => {
    const h = document.documentElement;
    const total = h.scrollHeight - h.clientHeight;
    if (total <= 0) return;
    const pct = Math.min(100, Math.round((h.scrollTop / total) * 100));
    if (pct > scrollDepthMax) scrollDepthMax = pct;
  };
  window.addEventListener('scroll', compute, { passive: true });
  window.addEventListener('beforeunload', () => {
    try {
      scrollDepthGauge?.record(scrollDepthMax, { 'frontend.stack': STACK });
    } catch {}
  });
}

export function getScrollDepthMax() {
  return scrollDepthMax;
}

const replayBuffer = [];
let chunkIndex = 0;
let flushTimer = null;

function flushReplay(useBeacon = false) {
  if (replayBuffer.length === 0) return;
  const payload = {
    session_id: SESSION_ID,
    user_id: USER_ID,
    trace_id: '',
    chunk_index: chunkIndex++,
    events: replayBuffer.splice(0, replayBuffer.length),
  };
  const body = JSON.stringify(payload);
  if (useBeacon && navigator.sendBeacon) {
    try {
      navigator.sendBeacon(REPLAY_ENDPOINT, new Blob([body], { type: 'application/json' }));
      return;
    } catch {}
  }
  sendWithRetry(body, 0);
}

function sendWithRetry(body, attempt) {
  fetch(REPLAY_ENDPOINT, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body,
    keepalive: true,
  }).catch(() => {
    if (attempt < 4) {
      const delay = Math.min(30000, 500 * Math.pow(2, attempt));
      setTimeout(() => sendWithRetry(body, attempt + 1), delay);
    }
  });
}

function initRrweb() {
  try {
    rrweb.record({
      emit(event) {
        replayBuffer.push(event);
        if (replayBuffer.length >= 100) flushReplay();
      },
    });
    flushTimer = setInterval(() => flushReplay(), 10000);
    window.addEventListener('beforeunload', () => {
      clearInterval(flushTimer);
      flushReplay(true);
    });
  } catch (e) {
    console.warn('[telemetry] rrweb init failed', e);
  }
}
