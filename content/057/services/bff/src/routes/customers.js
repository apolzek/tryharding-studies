import { makeClient } from '../clients/http.js';

const customers = makeClient(process.env.CUSTOMER_URL || 'http://customer:8002', 'customer');

export function registerCustomerRoutes(app) {
  app.post('/api/customers', async (req) => customers.request({ method: 'POST', url: '/customers', data: req.body }));
  app.get('/api/customers', async () => customers.request({ url: '/customers' }));
  app.get('/api/customers/:id', async (req) => customers.request({ url: `/customers/${req.params.id}` }));
}
