import Fastify from 'fastify';
import cors from '@fastify/cors';

import { registerAuthMiddleware } from './middleware/auth.js';
import { registerAuthRoutes } from './routes/auth.js';
import { registerCustomerRoutes } from './routes/customers.js';
import { registerCatalogRoutes } from './routes/catalogs.js';
import { registerProductRoutes } from './routes/products.js';
import { registerSignatureRoutes } from './routes/signatures.js';
import { registerCheckoutRoutes } from './routes/checkout.js';
import { registerOrderRoutes } from './routes/orders.js';
import { registerPaymentRoutes } from './routes/payments.js';
import { registerCartRoutes } from './routes/carts.js';
import { registerNotificationRoutes } from './routes/notifications.js';
import { registerEventRoutes } from './routes/events.js';

const fastify = Fastify({ logger: { level: process.env.LOG_LEVEL || 'info' } });

await fastify.register(cors, { origin: true });

fastify.get('/health', async () => ({
  status: 'ok',
  instance: process.env.HOSTNAME || 'bff',
}));

registerAuthMiddleware(fastify);

registerAuthRoutes(fastify);
registerCustomerRoutes(fastify);
registerCatalogRoutes(fastify);
registerProductRoutes(fastify);
registerSignatureRoutes(fastify);
registerCheckoutRoutes(fastify);
registerOrderRoutes(fastify);
registerPaymentRoutes(fastify);
registerCartRoutes(fastify);
registerNotificationRoutes(fastify);
registerEventRoutes(fastify);

const port = Number(process.env.PORT || 3000);
fastify
  .listen({ port, host: '0.0.0.0' })
  .then(() => fastify.log.info(`bff listening on ${port}`))
  .catch((err) => {
    fastify.log.error(err);
    process.exit(1);
  });
