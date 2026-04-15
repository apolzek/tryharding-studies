import Fastify, { FastifyInstance } from 'fastify';
import cors from '@fastify/cors';
import helmet from '@fastify/helmet';
import rateLimit from '@fastify/rate-limit';
import { trace } from '@opentelemetry/api';
import { healthRoutes } from './routes/health.js';
import { feedRoutes } from './routes/feed.js';
import { interactionRoutes } from './routes/interactions.js';
import { replayRoutes } from './routes/replay.js';

function parseCorsOrigin(): string[] | true {
  const raw = process.env.CORS_ORIGIN;
  if (!raw) {
    return [
      'http://localhost:5173',
      'http://localhost:5174',
      'http://localhost:5175',
    ];
  }
  if (raw.trim() === '*') return true;
  return raw.split(',').map((s) => s.trim()).filter(Boolean);
}

export async function buildServer(): Promise<FastifyInstance> {
  const app = Fastify({
    logger: {
      level: process.env.LOG_LEVEL ?? 'info',
      mixin() {
        const span = trace.getActiveSpan();
        if (!span) return {};
        const ctx = span.spanContext();
        return { trace_id: ctx.traceId, span_id: ctx.spanId };
      },
    },
    trustProxy: true,
    disableRequestLogging: false,
  });

  await app.register(helmet, { global: true, contentSecurityPolicy: false });
  await app.register(cors, {
    origin: parseCorsOrigin(),
    credentials: false,
    methods: ['GET', 'POST', 'OPTIONS'],
  });
  await app.register(rateLimit, {
    global: true,
    max: 200,
    timeWindow: '1 minute',
  });

  await app.register(healthRoutes);
  await app.register(feedRoutes);
  await app.register(interactionRoutes);
  await app.register(replayRoutes);

  app.setErrorHandler((err, _req, reply) => {
    app.log.error({ err }, 'unhandled error');
    const statusCode = err.statusCode ?? 500;
    reply.status(statusCode).send({
      error: err.name ?? 'internal_error',
      message: err.message,
    });
  });

  return app;
}
