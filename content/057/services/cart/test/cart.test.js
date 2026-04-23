import { test } from 'node:test';
import assert from 'node:assert/strict';
import { makeCart } from '../src/cart.js';

/** In-memory shim mimicking the subset of ioredis we use. */
function fakeRedis() {
  const data = new Map();
  const ttls = new Map();
  return {
    async hget(k, f) { const h = data.get(k); return h ? h.get(f) || null : null; },
    async hset(k, f, v) {
      if (!data.has(k)) data.set(k, new Map());
      data.get(k).set(f, v);
    },
    async hdel(k, f) { data.get(k)?.delete(f); },
    async hgetall(k) { return Object.fromEntries(data.get(k) || new Map()); },
    async expire(k, ttl) { ttls.set(k, ttl); },
    async ttl(k) { return ttls.get(k) ?? -1; },
    async del(k) { data.delete(k); ttls.delete(k); },
  };
}

test('add item, list, and see running total', async () => {
  const r = fakeRedis();
  const c = makeCart(r);

  await c.addItem('abc', { product_id: 'p1', sku: 'SKU1', qty: 2, unit_price: 10 });
  await c.addItem('abc', { product_id: 'p2', sku: 'SKU2', qty: 1, unit_price: 5 });

  const v = await c.list('abc');
  assert.equal(v.items.length, 2);
  assert.equal(v.total, 25);
});

test('adding same product accumulates qty', async () => {
  const r = fakeRedis();
  const c = makeCart(r);
  await c.addItem('x', { product_id: 'p1', sku: 'SKU1', qty: 2, unit_price: 10 });
  await c.addItem('x', { product_id: 'p1', sku: 'SKU1', qty: 3, unit_price: 10 });
  const v = await c.list('x');
  assert.equal(v.items[0].qty, 5);
  assert.equal(v.total, 50);
});

test('remove item drops it from the cart', async () => {
  const r = fakeRedis();
  const c = makeCart(r);
  await c.addItem('y', { product_id: 'p1', sku: 'SKU1', qty: 1, unit_price: 7 });
  await c.removeItem('y', 'p1');
  const v = await c.list('y');
  assert.equal(v.items.length, 0);
});
