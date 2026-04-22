import express from 'express';
import cors from 'cors';
import { tipsRouter } from './routes/tips.js';
import { rightsRouter } from './routes/rights.js';
import { taxesRouter } from './routes/taxes.js';
import { politiciansRouter } from './routes/politicians.js';
import { historyRouter } from './routes/history.js';

export const app = express();

app.use(cors());
app.use(express.json());

app.get('/health', (_req, res) => res.json({ status: 'ok', service: 'brasilzao' }));

app.use('/api/tips', tipsRouter);
app.use('/api/rights', rightsRouter);
app.use('/api/taxes', taxesRouter);
app.use('/api/politicians', politiciansRouter);
app.use('/api/history', historyRouter);

app.use((err, _req, res, _next) => {
  console.error('[brasilzao] error:', err);
  res.status(err.status || 500).json({ error: err.message || 'internal error' });
});
