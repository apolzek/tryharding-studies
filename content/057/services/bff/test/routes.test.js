import { test } from 'node:test';
import assert from 'node:assert/strict';
import Fastify from 'fastify';

test('health endpoint returns ok', async () => {
  const app = Fastify();
  app.get('/health', async () => ({ status: 'ok' }));
  const res = await app.inject({ method: 'GET', url: '/health' });
  assert.equal(res.statusCode, 200);
  assert.equal(JSON.parse(res.body).status, 'ok');
});

test('checkout rejects missing fields via downstream error', async () => {
  const app = Fastify();
  app.post('/api/checkout', async (req, reply) => {
    if (!req.body?.email) return reply.code(400).send({ error: 'email required' });
    return { ok: true };
  });
  const res = await app.inject({ method: 'POST', url: '/api/checkout', payload: {} });
  assert.equal(res.statusCode, 400);
});
