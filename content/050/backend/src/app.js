const express = require('express');
const cors = require('cors');

const authRoutes = require('./routes/auth');
const userRoutes = require('./routes/users');
const scrapRoutes = require('./routes/scraps');
const friendRoutes = require('./routes/friends');
const communityRoutes = require('./routes/communities');
const testimonialRoutes = require('./routes/testimonials');
const ratingRoutes = require('./routes/ratings');
const photoRoutes = require('./routes/photos');
const visitRoutes = require('./routes/visits');

function createApp(db) {
  const app = express();
  app.use(cors());
  app.use(express.json({ limit: '1mb' }));

  app.locals.db = db;

  app.get('/health', (_req, res) => res.json({ ok: true }));

  app.use('/api/auth', authRoutes);
  app.use('/api/users', userRoutes);
  app.use('/api/scraps', scrapRoutes);
  app.use('/api/friends', friendRoutes);
  app.use('/api/communities', communityRoutes);
  app.use('/api/testimonials', testimonialRoutes);
  app.use('/api/ratings', ratingRoutes);
  app.use('/api/photos', photoRoutes);
  app.use('/api/visits', visitRoutes);

  app.use((err, _req, res, _next) => {
    console.error(err);
    res.status(500).json({ error: 'internal error' });
  });

  return app;
}

module.exports = { createApp };
