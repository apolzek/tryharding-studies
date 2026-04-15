import type { FastifyInstance } from 'fastify';
import { z } from 'zod';
import { gzipSync } from 'node:zlib';
import { trace } from '@opentelemetry/api';
import { minioClient, bucket, putObjectWithRetry } from '../minio-client.js';
import { replayBytesTotal, replayChunksReceived } from '../metrics.js';

const bodySchema = z.object({
  session_id: z.string().min(1),
  user_id: z.string().min(1),
  trace_id: z.string().min(1),
  chunk_index: z.number().int().min(0),
  events: z.array(z.unknown()),
});

function datePrefix(): string {
  const now = new Date();
  const y = now.getUTCFullYear();
  const m = String(now.getUTCMonth() + 1).padStart(2, '0');
  const d = String(now.getUTCDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

interface SidecarMetadata {
  session_id: string;
  user_id: string;
  trace_id: string;
  first_seen: string;
  last_seen: string;
  chunk_count: number;
  total_bytes: number;
}

async function upsertSidecar(key: string, update: SidecarMetadata): Promise<void> {
  let existing: SidecarMetadata | undefined;
  try {
    const stream = await minioClient.getObject(bucket, key);
    const chunks: Buffer[] = [];
    for await (const chunk of stream) {
      chunks.push(chunk as Buffer);
    }
    existing = JSON.parse(Buffer.concat(chunks).toString('utf8')) as SidecarMetadata;
  } catch {
    existing = undefined;
  }
  const merged: SidecarMetadata = existing
    ? {
        ...existing,
        last_seen: update.last_seen,
        chunk_count: existing.chunk_count + 1,
        total_bytes: existing.total_bytes + update.total_bytes,
      }
    : update;
  const body = Buffer.from(JSON.stringify(merged, null, 2), 'utf8');
  await putObjectWithRetry(key, body, 'application/json');
}

export async function replayRoutes(app: FastifyInstance): Promise<void> {
  app.post(
    '/replay/ingest',
    {
      config: {
        rateLimit: { max: 50, timeWindow: '1 minute' },
      },
    },
    async (request, reply) => {
      const parsed = bodySchema.safeParse(request.body);
      if (!parsed.success) {
        return reply.status(400).send({ error: 'invalid_body', issues: parsed.error.issues });
      }
      const { session_id, user_id, trace_id, chunk_index, events } = parsed.data;
      const span = trace.getActiveSpan();
      span?.setAttributes({
        'session.id': session_id,
        'user.id': user_id,
        'replay.chunk_index': chunk_index,
        'replay.events_count': events.length,
      });

      const json = JSON.stringify({ session_id, user_id, trace_id, chunk_index, events });
      const compressed = gzipSync(Buffer.from(json, 'utf8'));
      const prefix = datePrefix();
      const key = `${prefix}/${session_id}/${chunk_index}.json.gz`;
      const sidecarKey = `${prefix}/${session_id}/metadata.json`;

      try {
        await putObjectWithRetry(key, compressed, 'application/gzip');
        await upsertSidecar(sidecarKey, {
          session_id,
          user_id,
          trace_id,
          first_seen: new Date().toISOString(),
          last_seen: new Date().toISOString(),
          chunk_count: 1,
          total_bytes: compressed.length,
        });
      } catch (err) {
        request.log.error({ err, key }, 'failed to persist replay chunk');
        return reply.status(502).send({ error: 'storage_failure' });
      }

      replayChunksReceived.add(1, { session_id });
      replayBytesTotal.add(compressed.length, { session_id });

      request.log.info(
        {
          event: 'replay_chunk',
          session_id,
          user_id,
          chunk_index,
          bytes: compressed.length,
          trace_id: span?.spanContext().traceId ?? trace_id,
        },
        'replay chunk stored',
      );

      return { stored: true, key, bytes: compressed.length };
    },
  );
}
