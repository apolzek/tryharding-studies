import { makeClient } from '../clients/http.js';

const authClient = makeClient(process.env.AUTH_URL || 'http://auth:8001', 'auth');

/**
 * Bearer-token guard. Exempts:
 * - /health
 * - /api/auth/*  (register/login/validate)
 * - /api/events  (SSE handshake — clients can subscribe anonymously)
 */
const EXEMPT = [/^\/health$/, /^\/api\/auth(\/|$)/, /^\/api\/events$/];

export function registerAuthMiddleware(app) {
  app.addHook('preHandler', async (req, reply) => {
    if (EXEMPT.some((re) => re.test(req.url))) return;
    if (process.env.AUTH_DISABLED === '1') return;

    const header = req.headers.authorization;
    if (!header || !header.startsWith('Bearer ')) {
      reply.code(401).send({ error: 'missing bearer token' });
      return reply;
    }
    try {
      const v = await authClient.request({
        method: 'GET',
        url: '/validate',
        headers: { Authorization: header },
      });
      if (v?.sub) req.user = v;
      else {
        reply.code(401).send({ error: 'invalid token' });
        return reply;
      }
    } catch {
      reply.code(401).send({ error: 'token validation failed' });
      return reply;
    }
  });
}
