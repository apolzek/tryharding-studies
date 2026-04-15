import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const BASE = __ENV.BASE_URL || 'http://backend-api:8080';

export const options = {
  scenarios: {
    browse: {
      executor: 'constant-vus',
      vus: 20,
      duration: '1m',
      exec: 'browse',
    },
    interact: {
      executor: 'constant-vus',
      vus: 10,
      duration: '1m',
      exec: 'interact',
    },
    replay: {
      executor: 'constant-vus',
      vus: 5,
      duration: '1m',
      exec: 'replay',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

export function browse() {
  const offset = randomIntBetween(0, 100);
  const res = http.get(`${BASE}/api/feed?offset=${offset}&limit=10`);
  check(res, {
    'feed 200': (r) => r.status === 200,
    'has posts': (r) => {
      try {
        return JSON.parse(r.body).posts.length === 10;
      } catch (_e) {
        return false;
      }
    },
  });
  sleep(1);
}

const TYPES = ['like', 'comment', 'share'];

export function interact() {
  const payload = JSON.stringify({
    post_id: `post_${randomIntBetween(0, 200)}`,
    type: TYPES[randomIntBetween(0, TYPES.length - 1)],
    user_id: `user_${randomIntBetween(1, 500)}`,
    session_id: `sess_${randomIntBetween(1, 50)}`,
  });
  const res = http.post(`${BASE}/api/interactions`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, { 'interaction 200': (r) => r.status === 200 });
  sleep(1);
}

function fakeEvents(n) {
  const out = [];
  for (let i = 0; i < n; i++) {
    out.push({
      type: 3,
      timestamp: Date.now() + i,
      data: { source: 1, x: randomIntBetween(0, 1000), y: randomIntBetween(0, 1000) },
    });
  }
  return out;
}

export function replay() {
  const payload = JSON.stringify({
    session_id: `sess_${randomIntBetween(1, 50)}`,
    user_id: `user_${randomIntBetween(1, 500)}`,
    trace_id: `${Date.now().toString(16)}${randomIntBetween(1000, 9999)}`.padEnd(32, '0'),
    chunk_index: randomIntBetween(0, 100),
    events: fakeEvents(10),
  });
  const res = http.post(`${BASE}/replay/ingest`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, { 'replay 200': (r) => r.status === 200 });
  sleep(2);
}
