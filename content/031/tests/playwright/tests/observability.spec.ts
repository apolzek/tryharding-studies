import { test, expect, request } from '@playwright/test';

const PROM = process.env.PROM_URL ?? 'http://localhost:9090';
const LOKI = process.env.LOKI_URL ?? 'http://localhost:3100';
const TEMPO = process.env.TEMPO_URL ?? 'http://localhost:3200';
const GRAFANA = process.env.GRAFANA_URL ?? 'http://localhost:3000';

test.describe('observability stack health', () => {
  test('prometheus reports otel-collector, prometheus, backend-api as up', async () => {
    const ctx = await request.newContext();
    const res = await ctx.get(`${PROM}/api/v1/query?query=up`);
    expect(res.ok()).toBeTruthy();
    const body = (await res.json()) as {
      data: { result: Array<{ metric: Record<string, string>; value: [number, string] }> };
    };
    const ups = body.data.result
      .filter((r) => r.value[1] === '1')
      .map((r) => r.metric.job ?? r.metric.service ?? '');
    const expected = ['otel-collector', 'prometheus', 'backend-api'];
    for (const name of expected) {
      expect(
        ups.some((j) => j.includes(name)),
        `expected ${name} to be up, got: ${ups.join(',')}`,
      ).toBeTruthy();
    }
  });

  test('loki /ready returns 200', async () => {
    const ctx = await request.newContext();
    const res = await ctx.get(`${LOKI}/ready`);
    expect(res.status()).toBe(200);
  });

  test('tempo /ready returns 200', async () => {
    const ctx = await request.newContext();
    const res = await ctx.get(`${TEMPO}/ready`);
    expect(res.status()).toBe(200);
  });

  test('grafana /api/health reports database ok', async () => {
    const ctx = await request.newContext();
    const res = await ctx.get(`${GRAFANA}/api/health`);
    expect(res.ok()).toBeTruthy();
    const body = (await res.json()) as { database: string };
    expect(body.database).toBe('ok');
  });
});
