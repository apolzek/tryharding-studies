import amqp from 'amqplib';

/**
 * Server-Sent Events live feed — fans out RabbitMQ messages to every
 * connected browser. The BFF binds anonymous exclusive queues to the
 * three main topic exchanges and pushes every message down the wire as
 * an SSE event.
 */
const EXCHANGES = ['customer.events', 'order.events', 'payment.events'];

async function subscribe(onMessage) {
  const url = process.env.RABBIT_URL || 'amqp://guest:guest@rabbitmq:5672/';
  const conn = await amqp.connect(url);
  const ch = await conn.createChannel();
  for (const ex of EXCHANGES) {
    await ch.assertExchange(ex, 'topic', { durable: true });
    const q = await ch.assertQueue('', { exclusive: true, autoDelete: true });
    await ch.bindQueue(q.queue, ex, '#');
    ch.consume(q.queue, (msg) => {
      if (!msg) return;
      try {
        onMessage({
          exchange: ex,
          routingKey: msg.fields.routingKey,
          body: JSON.parse(msg.content.toString()),
        });
      } catch {
        /* ignore malformed */
      }
      ch.ack(msg);
    });
  }
  return conn;
}

export function registerEventRoutes(app) {
  const clients = new Set();

  (async () => {
    try {
      await subscribe((evt) => {
        const line = `event: ${evt.exchange}\ndata: ${JSON.stringify(evt)}\n\n`;
        for (const reply of clients) reply.raw.write(line);
      });
      app.log.info('SSE bridge subscribed to RabbitMQ');
    } catch (e) {
      app.log.error({ err: e }, 'SSE bridge failed to subscribe');
    }
  })();

  app.get('/api/events', (req, reply) => {
    reply.raw.writeHead(200, {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive',
      'Access-Control-Allow-Origin': '*',
    });
    reply.raw.write('retry: 3000\n\n');
    clients.add(reply);
    req.raw.on('close', () => clients.delete(reply));
  });
}
