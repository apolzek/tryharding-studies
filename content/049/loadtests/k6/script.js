import http from 'k6/http';
import { check } from 'k6';

export const options = {
  scenarios: {
    simple: {
      executor: 'constant-vus',
      vus: 50,
      duration: '15s',
      exec: 'simple',
      tags: { endpoint: 'simple' },
    },
    medium: {
      executor: 'constant-vus',
      vus: 50,
      duration: '15s',
      exec: 'medium',
      startTime: '15s',
      tags: { endpoint: 'medium' },
    },
    heavy: {
      executor: 'constant-vus',
      vus: 50,
      duration: '15s',
      exec: 'heavy',
      startTime: '30s',
      tags: { endpoint: 'heavy' },
    },
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)'],
};

const BASE = __ENV.BASE || 'http://127.0.0.1:8765';

export function simple() {
  const r = http.get(`${BASE}/simple`);
  check(r, { 'status 200': (x) => x.status === 200 });
}
export function medium() {
  const r = http.get(`${BASE}/medium?n=2000`);
  check(r, { 'status 200': (x) => x.status === 200 });
}
export function heavy() {
  const r = http.get(`${BASE}/heavy?n=25`);
  check(r, { 'status 200': (x) => x.status === 200 });
}
