import Fastify from 'fastify';
import Redis from 'ioredis';

import { makeCart } from './cart.js';

const fastify = Fastify({ logger: { level: process.env.LOG_LEVEL || 'info' } });

const redis = new Redis(process.env.REDIS_URL || 'redis://redis:6379/0');
const cart = makeCart(redis);

fastify.get('/health', async () => ({ status: 'ok' }));

fastify.post('/carts/:id/items', async (req) => cart.addItem(req.params.id, req.body));
fastify.delete('/carts/:id/items/:productId', async (req) => {
  await cart.removeItem(req.params.id, req.params.productId);
  return { ok: true };
});
fastify.get('/carts/:id', async (req) => cart.list(req.params.id));
fastify.delete('/carts/:id', async (req) => {
  await cart.clear(req.params.id);
  return { ok: true };
});

const port = Number(process.env.PORT || 8009);
fastify
  .listen({ port, host: '0.0.0.0' })
  .then(() => fastify.log.info(`cart listening on ${port}`))
  .catch((err) => { fastify.log.error(err); process.exit(1); });
