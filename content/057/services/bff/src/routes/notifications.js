import { makeClient } from '../clients/http.js';

const notif = makeClient(process.env.NOTIFICATION_URL || 'http://notification:8008', 'notification');

export function registerNotificationRoutes(app) {
  app.get('/api/notifications/recent', async (req) =>
    notif.request({ url: `/notifications/recent?limit=${Number(req.query.limit || 50)}` }),
  );
}
