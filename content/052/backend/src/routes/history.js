import { Router } from 'express';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const data = JSON.parse(readFileSync(resolve(here, '../data/history.json'), 'utf8'));

export const historyRouter = Router();

historyRouter.get('/', (_req, res) => res.json(data));
historyRouter.get('/:id', (req, res) => {
  const p = data.periods.find((x) => x.id === req.params.id);
  if (!p) return res.status(404).json({ error: 'period not found' });
  res.json(p);
});
