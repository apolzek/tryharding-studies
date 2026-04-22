import { Router } from 'express';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const data = JSON.parse(readFileSync(resolve(here, '../data/taxes.json'), 'utf8'));

export const taxesRouter = Router();

taxesRouter.get('/', (_req, res) => res.json(data));
