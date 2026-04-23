import { makeClient } from '../clients/http.js';

const auth = makeClient(process.env.AUTH_URL || 'http://auth:8001', 'auth');

export function registerAuthRoutes(app) {
  app.post('/api/auth/register', async (req) => auth.request({ method: 'POST', url: '/register', data: req.body }));
  app.post('/api/auth/login', async (req) => auth.request({ method: 'POST', url: '/login', data: req.body }));
  app.get('/api/auth/validate', async (req) =>
    auth.request({ method: 'GET', url: '/validate', headers: { Authorization: req.headers.authorization || '' } }),
  );
}
