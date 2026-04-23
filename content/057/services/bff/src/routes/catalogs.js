import { makeClient } from '../clients/http.js';

const catalog = makeClient(process.env.CATALOG_URL || 'http://catalog:8003', 'catalog');

export function registerCatalogRoutes(app) {
  app.post('/api/catalogs', async (req) => catalog.request({ method: 'POST', url: '/catalogs', data: req.body }));
  app.get('/api/catalogs', async () => catalog.request({ url: '/catalogs' }));
  app.get('/api/catalogs/:id', async (req) => catalog.request({ url: `/catalogs/${req.params.id}` }));
}
