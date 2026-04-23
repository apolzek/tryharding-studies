/**
 * k6 load test for the checkout saga.
 *
 * Simulates a ramp of virtual users hammering /api/checkout through the
 * Nginx load balancer. Records a custom trend so you can see p95/p99
 * over the BFF circuit breaker and the payment retry loop.
 *
 * Run:  docker compose --profile load run --rm k6 run /scripts/checkout.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';

export const options = {
  stages: [
    { duration: '10s', target: 5 },
    { duration: '20s', target: 25 },
    { duration: '20s', target: 50 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<2000'],
  },
};

const BASE = __ENV.BASE || 'http://nginx';
const checkoutTrend = new Trend('checkout_duration_ms', true);
const sagaFailed = new Counter('saga_failed');

function login() {
  const email = `load-${Math.floor(Math.random() * 1_000_000)}@057.test`;
  const password = 'load12345';
  http.post(`${BASE}/api/auth/register`, JSON.stringify({ email, password }), {
    headers: { 'content-type': 'application/json' },
  });
  const res = http.post(`${BASE}/api/auth/login`, JSON.stringify({ email, password }), {
    headers: { 'content-type': 'application/json' },
  });
  return res.json('access_token');
}

function pickProduct(token) {
  const res = http.get(`${BASE}/api/products?limit=10`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = res.json();
  const items = body?.items || [];
  return items[Math.floor(Math.random() * items.length)]?.id || null;
}

export default function () {
  const token = login();
  const pid = pickProduct(token);
  if (!pid) { sleep(1); return; }

  const payload = {
    name: `VU-${__VU}`,
    email: `vu${__VU}-${__ITER}@057.test`,
    document: `${__VU}${__ITER}`,
    product_id: pid,
    quantity: 1,
    plan: 'load',
    amount: 9.9,
  };

  const t0 = Date.now();
  const res = http.post(`${BASE}/api/checkout`, JSON.stringify(payload), {
    headers: {
      'content-type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    timeout: '10s',
  });
  checkoutTrend.add(Date.now() - t0);

  const ok = check(res, { '2xx': (r) => r.status >= 200 && r.status < 300 });
  if (!ok) sagaFailed.add(1);

  sleep(0.5);
}
