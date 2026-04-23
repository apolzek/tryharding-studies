/**
 * Cart repository — each cart lives in a Redis hash keyed by cart id,
 * with an absolute TTL that refreshes on every mutation. Mirrors the
 * way a real e-commerce cart behaves: survives across tabs, expires
 * if forgotten.
 */

const TTL_SECONDS = Number(process.env.CART_TTL_SECONDS || 60 * 60 * 24);

function cartKey(id) { return `cart:${id}`; }

export function makeCart(redis) {
  async function touch(id) { await redis.expire(cartKey(id), TTL_SECONDS); }

  async function addItem(id, { product_id, sku, qty, unit_price }) {
    const field = product_id;
    const existing = await redis.hget(cartKey(id), field);
    const current = existing ? JSON.parse(existing) : { product_id, sku, qty: 0, unit_price };
    current.qty = Number(current.qty) + Number(qty);
    current.unit_price = Number(unit_price);
    current.sku = sku;
    await redis.hset(cartKey(id), field, JSON.stringify(current));
    await touch(id);
    return current;
  }

  async function removeItem(id, product_id) {
    await redis.hdel(cartKey(id), product_id);
    await touch(id);
  }

  async function list(id) {
    const raw = await redis.hgetall(cartKey(id));
    const items = Object.values(raw).map((v) => JSON.parse(v));
    const total = items.reduce((acc, it) => acc + it.qty * it.unit_price, 0);
    const ttl = await redis.ttl(cartKey(id));
    return { id, items, total, ttl_seconds: ttl };
  }

  async function clear(id) {
    await redis.del(cartKey(id));
  }

  return { addItem, removeItem, list, clear };
}
