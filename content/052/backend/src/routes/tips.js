import { Router } from 'express';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const data = JSON.parse(readFileSync(resolve(here, '../data/tips.json'), 'utf8'));

function dayOfYear(date) {
  const start = new Date(date.getFullYear(), 0, 0);
  const diff = date - start;
  return Math.floor(diff / 86400000);
}

export const tipsRouter = Router();

tipsRouter.get('/today', (_req, res) => {
  const tips = data.tips;
  const idx = dayOfYear(new Date()) % tips.length;
  res.json({ date: new Date().toISOString().slice(0, 10), ...tips[idx] });
});

tipsRouter.get('/', (_req, res) => {
  res.json({ total: data.tips.length, tips: data.tips });
});
