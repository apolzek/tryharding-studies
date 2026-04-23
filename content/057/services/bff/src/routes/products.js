import { productGrpc } from '../clients/grpc.js';

export function registerProductRoutes(app) {
  app.get('/api/products', async (req) => {
    const limit = Number(req.query.limit || 50);
    const offset = Number(req.query.offset || 0);
    return productGrpc.list({ limit, offset });
  });

  app.get('/api/products/:id', async (req) => productGrpc.get({ id: req.params.id }));

  app.post('/api/products', async (req) => productGrpc.create(req.body));
}
