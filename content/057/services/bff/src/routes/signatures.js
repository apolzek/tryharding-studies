import { makeClient } from '../clients/http.js';

const signatures = makeClient(process.env.SIGNATURE_URL || 'http://signature:8005', 'signature');

export function registerSignatureRoutes(app) {
  app.get('/api/signatures/summary', async () => signatures.request({ url: '/signatures/summary' }));
  app.get('/api/signatures/by-customer/:id', async (req) =>
    signatures.request({ url: `/signatures/by-customer/${req.params.id}` }),
  );
  app.post('/api/signatures', async (req) => signatures.request({ method: 'POST', url: '/signatures', data: req.body }));
}
