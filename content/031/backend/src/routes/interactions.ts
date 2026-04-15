import type { FastifyInstance } from 'fastify';
import { z } from 'zod';
import { trace } from '@opentelemetry/api';
import { interactionsCounter } from '../metrics.js';

const bodySchema = z.object({
  post_id: z.string().min(1),
  type: z.enum(['like', 'comment', 'share']),
  user_id: z.string().min(1),
  session_id: z.string().min(1),
});

export async function interactionRoutes(app: FastifyInstance): Promise<void> {
  app.post('/api/interactions', async (request, reply) => {
    const parsed = bodySchema.safeParse(request.body);
    if (!parsed.success) {
      return reply.status(400).send({ error: 'invalid_body', issues: parsed.error.issues });
    }
    const payload = parsed.data;
    const span = trace.getActiveSpan();
    const traceId = span?.spanContext().traceId;

    span?.setAttributes({
      'interaction.type': payload.type,
      'interaction.post_id': payload.post_id,
      'session.id': payload.session_id,
      'user.id': payload.user_id,
    });

    interactionsCounter.add(1, { type: payload.type });

    request.log.info(
      {
        event: 'interaction',
        ...payload,
        trace_id: traceId,
      },
      'interaction received',
    );

    return { ok: true };
  });
}
