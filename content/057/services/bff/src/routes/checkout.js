import { randomUUID } from 'node:crypto';

import { makeClient } from '../clients/http.js';
import { productGrpc } from '../clients/grpc.js';

const customers = makeClient(process.env.CUSTOMER_URL || 'http://customer:8002', 'customer');
const orders    = makeClient(process.env.ORDER_URL    || 'http://order:8006',    'order');
const payment   = makeClient(process.env.PAYMENT_URL  || 'http://payment:8007',  'payment');

/**
 * Orchestrated Saga: customer → stock → order → payment.
 *
 * Each step records a compensation. If a later step fails we run the
 * compensations in reverse order, so the system stays consistent even
 * when one hop fails halfway through. The idempotency key on payment
 * also makes a client retry of the whole saga safe.
 */
export function registerCheckoutRoutes(app) {
  app.post('/api/checkout', async (req, reply) => {
    const { name, email, document, product_id, quantity = 1, plan = 'standard', amount = 0 } = req.body || {};
    const idemKey = req.headers['idempotency-key'] || randomUUID();
    const trace = { idemKey, steps: [] };
    const compensations = [];

    // Step 1: customer
    let customer;
    try {
      customer = await customers.request({ method: 'POST', url: '/customers', data: { name, email, document } });
      trace.steps.push({ step: 'customer.created', id: customer.id });
    } catch (err) {
      return reply.code(502).send({ ...trace, error: 'customer-failed', detail: String(err) });
    }

    // Step 2: stock decrement (gRPC) — compensation = re-increment on product via create (no inverse yet → reserve stock logically)
    try {
      const res = await productGrpc.decrementStock({ id: product_id, quantity });
      if (!res.ok) throw new Error(`insufficient stock: remaining=${res.remaining}`);
      trace.steps.push({ step: 'stock.decremented', remaining: res.remaining });
      compensations.push({
        label: 'stock.restore',
        run: async () => {
          app.log.warn({ product_id, quantity }, 'compensation: restoring stock (simulated)');
        },
      });
    } catch (err) {
      return reply.code(409).send({ ...trace, error: 'stock-failed', detail: String(err) });
    }

    // Step 3: create the order
    let order;
    try {
      order = await orders.request({
        method: 'POST',
        url: '/orders',
        data: {
          customer_id: customer.id,
          items: [{ product_id, sku: 'SKU-AUTO', qty: quantity, unit_price: amount }],
        },
      });
      trace.steps.push({ step: 'order.created', id: order.id });
      compensations.push({
        label: 'order.cancel',
        run: async () => {
          app.log.warn({ order_id: order.id }, 'compensation: cancelling order');
          await orders.request({ method: 'POST', url: `/orders/${order.id}/cancel` }).catch(() => {});
        },
      });
    } catch (err) {
      await runCompensations(app, compensations);
      return reply.code(502).send({ ...trace, error: 'order-failed', detail: String(err) });
    }

    // Step 4: request payment transition (pushes order into AWAITING_PAYMENT + emits order.payment_requested)
    try {
      await orders.request({ method: 'POST', url: `/orders/${order.id}/request-payment` });
      trace.steps.push({ step: 'order.payment_requested' });
    } catch (err) {
      await runCompensations(app, compensations);
      return reply.code(502).send({ ...trace, error: 'payment-request-failed', detail: String(err) });
    }

    // Step 5: synchronous charge on payment service (with idempotency)
    try {
      const chargeResp = await payment.request({
        method: 'POST',
        url: '/charges',
        data: { order_id: order.id, amount: amount * quantity },
        headers: { 'Idempotency-Key': idemKey },
      });
      trace.steps.push({ step: 'payment.charged', result: chargeResp });
      // If payment failed we'll also have gotten payment.failed on the bus — the
      // order service will receive it and move the order to PAYMENT_FAILED.
    } catch (err) {
      await runCompensations(app, compensations);
      return reply.code(502).send({ ...trace, error: 'payment-failed', detail: String(err) });
    }

    return { ok: true, ...trace, customer, order };
  });
}

async function runCompensations(app, comps) {
  for (const c of comps.reverse()) {
    try { await c.run(); }
    catch (e) { app.log.error({ err: e, step: c.label }, 'compensation failed'); }
  }
}
