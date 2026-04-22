const store = new Map();

export function cacheGet(key) {
  const entry = store.get(key);
  if (!entry) return null;
  if (Date.now() > entry.expiresAt) {
    store.delete(key);
    return null;
  }
  return entry.value;
}

export function cacheSet(key, value, ttlMs) {
  store.set(key, { value, expiresAt: Date.now() + ttlMs });
}

export async function withCache(key, ttlMs, producer) {
  const hit = cacheGet(key);
  if (hit) return hit;
  const value = await producer();
  cacheSet(key, value, ttlMs);
  return value;
}
