import { Router } from 'express';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const data = JSON.parse(readFileSync(resolve(here, '../data/rights.json'), 'utf8'));

export const rightsRouter = Router();

rightsRouter.get('/', (_req, res) => {
  res.json({
    categories: data.categories.map((c) => ({
      id: c.id,
      title: c.title,
      description: c.description,
      icon: c.icon,
      count: c.items.length,
    })),
  });
});

rightsRouter.get('/:id', (req, res) => {
  const cat = data.categories.find((c) => c.id === req.params.id);
  if (!cat) return res.status(404).json({ error: 'category not found' });
  res.json(cat);
});
