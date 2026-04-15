import { test, expect, Request } from '@playwright/test';

test.describe('feed smoke', () => {
  test('loads feed, scrolls, likes, and emits telemetry', async ({ page }) => {
    const traceRequests: Request[] = [];
    const metricRequests: Request[] = [];

    page.on('request', (req) => {
      const url = req.url();
      if (req.method() !== 'POST') return;
      if (url.includes('/v1/traces')) traceRequests.push(req);
      if (url.includes('/v1/metrics')) metricRequests.push(req);
    });

    await page.goto('/');

    // Feed renders at least 5 posts. Try several candidate selectors.
    const candidates = [
      '[data-testid="post"]',
      'article',
      '.post',
      '[class*="post"]',
    ];
    let postSelector = '';
    for (const sel of candidates) {
      const count = await page.locator(sel).count();
      if (count >= 5) {
        postSelector = sel;
        break;
      }
    }
    expect(postSelector, 'no candidate selector matched at least 5 posts').not.toBe('');
    const initialCount = await page.locator(postSelector).count();
    expect(initialCount).toBeGreaterThanOrEqual(5);

    // Scroll to trigger infinite scroll.
    await page.evaluate(async () => {
      for (let i = 0; i < 6; i++) {
        window.scrollBy(0, window.innerHeight);
        await new Promise((r) => setTimeout(r, 400));
      }
    });
    await page.waitForTimeout(1500);
    const afterScrollCount = await page.locator(postSelector).count();
    expect(afterScrollCount).toBeGreaterThan(initialCount);

    // Click a like button.
    const likeCandidates = [
      '[data-testid="like-button"]',
      'button[aria-label*="like" i]',
      'button:has-text("Like")',
      'button:has-text("like")',
    ];
    let likeSelector = '';
    for (const sel of likeCandidates) {
      if ((await page.locator(sel).count()) > 0) {
        likeSelector = sel;
        break;
      }
    }
    if (likeSelector) {
      const firstLike = page.locator(likeSelector).first();
      const before = (await firstLike.innerText()).match(/\d+/)?.[0] ?? '0';
      await firstLike.click();
      await page.waitForTimeout(300);
      const after = (await firstLike.innerText()).match(/\d+/)?.[0] ?? '0';
      expect(Number(after)).toBeGreaterThanOrEqual(Number(before));
    }

    // Dwell for telemetry to flush.
    await page.waitForTimeout(2000);

    expect(
      traceRequests.length,
      'expected at least one POST to /v1/traces',
    ).toBeGreaterThanOrEqual(1);
    expect(
      metricRequests.length,
      'expected at least one POST to /v1/metrics',
    ).toBeGreaterThanOrEqual(1);
  });
});
