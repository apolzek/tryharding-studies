import { makeClient } from '../clients/http.js';

const payment = makeClient(process.env.PAYMENT_URL || 'http://payment:8007', 'payment');

export function registerPaymentRoutes(app) {
  app.post('/api/payments/charges', async (req) => {
    const headers = {};
    if (req.headers['idempotency-key']) headers['Idempotency-Key'] = req.headers['idempotency-key'];
    return payment.request({ method: 'POST', url: '/charges', data: req.body, headers });
  });
}
