import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { api, setToken, getToken } from '../api.js';

describe('api client', () => {
  beforeEach(() => {
    setToken(null);
    globalThis.fetch = vi.fn();
  });
  afterEach(() => { vi.restoreAllMocks(); });

  test('register sends JSON payload', async () => {
    globalThis.fetch.mockResolvedValue({
      ok: true, status: 201, text: async () => JSON.stringify({ token: 'tok', user: { id: 1 } })
    });
    const res = await api.register('alice', 'pw', 'Alice');
    expect(res.token).toBe('tok');
    const [url, opts] = globalThis.fetch.mock.calls[0];
    expect(url).toContain('/api/auth/register');
    expect(opts.method).toBe('POST');
    expect(JSON.parse(opts.body)).toEqual({ username: 'alice', password: 'pw', display_name: 'Alice' }); // pragma: allowlist secret
  });

  test('attaches Authorization header when token set', async () => {
    setToken('abc123');
    expect(getToken()).toBe('abc123');
    globalThis.fetch.mockResolvedValue({ ok: true, status: 200, text: async () => '{}' });
    await api.me();
    const [, opts] = globalThis.fetch.mock.calls[0];
    expect(opts.headers.Authorization).toBe('Bearer abc123');
  });

  test('throws with server error message', async () => {
    globalThis.fetch.mockResolvedValue({
      ok: false, status: 401, text: async () => JSON.stringify({ error: 'invalid credentials' })
    });
    await expect(api.login('a', 'b')).rejects.toThrow('invalid credentials');
  });
});
