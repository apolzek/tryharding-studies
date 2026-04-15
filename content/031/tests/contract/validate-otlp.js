#!/usr/bin/env node
// Validates a minimal OTLP/JSON traces payload.
// Usage: node validate-otlp.js [path/to/sample.json]
import { readFileSync } from 'node:fs';

const path = process.argv[2] ?? new URL('./sample-otlp.json', import.meta.url).pathname;

function fail(msg) {
  console.error(`FAIL: ${msg}`);
  process.exit(1);
}

let payload;
try {
  payload = JSON.parse(readFileSync(path, 'utf8'));
} catch (e) {
  fail(`cannot read/parse ${path}: ${e.message}`);
}

const resourceSpans = payload.resourceSpans;
if (!Array.isArray(resourceSpans) || resourceSpans.length === 0) {
  fail('resourceSpans missing or empty');
}

let sawServiceName = false;
let sawSessionId = false;
const traceIdRe = /^[0-9a-f]{32}$/i;

for (const rs of resourceSpans) {
  const attrs = rs.resource?.attributes ?? [];
  for (const a of attrs) {
    if (a.key === 'service.name' && a.value?.stringValue) sawServiceName = true;
  }
  for (const ss of rs.scopeSpans ?? []) {
    for (const span of ss.spans ?? []) {
      if (!span.traceId || !traceIdRe.test(span.traceId)) {
        fail(`invalid trace_id format: ${span.traceId}`);
      }
      for (const a of span.attributes ?? []) {
        if (a.key === 'session.id' && a.value?.stringValue) sawSessionId = true;
      }
    }
  }
}

if (!sawServiceName) fail('resource attribute service.name not found');
if (!sawSessionId) fail('span attribute session.id not found on any span');

console.log('OK: OTLP payload has service.name, session.id attribute, and valid trace_ids');
