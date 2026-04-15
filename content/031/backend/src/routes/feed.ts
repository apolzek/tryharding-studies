import type { FastifyInstance } from 'fastify';
import { z } from 'zod';
import { feedRequestDuration } from '../metrics.js';

const querySchema = z.object({
  offset: z.coerce.number().int().min(0).default(0),
  limit: z.coerce.number().int().min(1).max(100).default(10),
});

interface Post {
  id: string;
  author: string;
  avatarColor: string;
  gradient: string;
  title: string;
  caption: string;
  likes: number;
  comments: number;
  timestamp: string;
}

const AUTHORS = [
  'alice', 'bob', 'carol', 'dave', 'eve',
  'frank', 'grace', 'heidi', 'ivan', 'judy',
];
const COLORS = ['#ff6b6b', '#4ecdc4', '#45b7d1', '#96ceb4', '#ffeaa7', '#a29bfe', '#fd79a8'];
const GRADIENTS = [
  'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
  'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
  'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
  'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)',
  'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
];
const TITLES = [
  'Morning thoughts', 'Coffee break', 'Hot take', 'Shower idea',
  'Late night ramble', 'Quick update', 'Found this cool', 'Random thought',
];
const CAPTIONS = [
  'Just thinking about stuff.',
  'This changed my whole day.',
  'Not sure how I feel about this yet.',
  'Sharing for no reason in particular.',
  'Has anyone else noticed this?',
  'This is wild actually.',
];

function pick<T>(arr: readonly T[], seed: number): T {
  const idx = Math.abs(Math.floor(Math.sin(seed) * 10000)) % arr.length;
  return arr[idx] as T;
}

function buildPost(index: number): Post {
  return {
    id: `post_${index}`,
    author: pick(AUTHORS, index * 7 + 1),
    avatarColor: pick(COLORS, index * 11 + 3),
    gradient: pick(GRADIENTS, index * 13 + 5),
    title: pick(TITLES, index * 17 + 7),
    caption: pick(CAPTIONS, index * 19 + 9),
    likes: (index * 37) % 500,
    comments: (index * 13) % 80,
    timestamp: new Date(Date.UTC(2026, 3, 15) - index * 3600_000).toISOString(),
  };
}

export async function feedRoutes(app: FastifyInstance): Promise<void> {
  app.get('/api/feed', async (request, reply) => {
    const started = Date.now();
    const parsed = querySchema.safeParse(request.query);
    if (!parsed.success) {
      return reply.status(400).send({ error: 'invalid_query', issues: parsed.error.issues });
    }
    const { offset, limit } = parsed.data;
    const posts: Post[] = Array.from({ length: limit }, (_, i) => buildPost(offset + i));
    feedRequestDuration.record(Date.now() - started, { route: '/api/feed' });
    return { posts, offset, limit };
  });
}
