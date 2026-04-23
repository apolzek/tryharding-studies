import { makeClient } from '../clients/http.js';

const carts = makeClient(process.env.CART_URL || 'http://cart:8009', 'cart');

export function registerCartRoutes(app) {
  app.get('/api/carts/:id', async (req) => carts.request({ url: `/carts/${req.params.id}` }));
  app.post('/api/carts/:id/items', async (req) =>
    carts.request({ method: 'POST', url: `/carts/${req.params.id}/items`, data: req.body }),
  );
  app.delete('/api/carts/:id/items/:productId', async (req) =>
    carts.request({ method: 'DELETE', url: `/carts/${req.params.id}/items/${req.params.productId}` }),
  );
  app.delete('/api/carts/:id', async (req) =>
    carts.request({ method: 'DELETE', url: `/carts/${req.params.id}` }),
  );
}
