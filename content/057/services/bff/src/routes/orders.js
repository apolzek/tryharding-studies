import { makeClient } from '../clients/http.js';

const orders = makeClient(process.env.ORDER_URL || 'http://order:8006', 'order');

export function registerOrderRoutes(app) {
  app.post('/api/orders', async (req) => orders.request({ method: 'POST', url: '/orders', data: req.body }));
  app.get('/api/orders/:id', async (req) => orders.request({ url: `/orders/${req.params.id}` }));
  app.get('/api/orders/:id/history', async (req) => orders.request({ url: `/orders/${req.params.id}/history` }));
  app.get('/api/orders/by-customer/:id', async (req) => orders.request({ url: `/orders/by-customer/${req.params.id}` }));
  app.post('/api/orders/:id/request-payment', async (req) =>
    orders.request({ method: 'POST', url: `/orders/${req.params.id}/request-payment` }),
  );
  app.post('/api/orders/:id/fulfill', async (req) =>
    orders.request({ method: 'POST', url: `/orders/${req.params.id}/fulfill` }),
  );
  app.post('/api/orders/:id/cancel', async (req) =>
    orders.request({ method: 'POST', url: `/orders/${req.params.id}/cancel` }),
  );
}
