const express = require('express');
const { requireAuth } = require('../middleware/auth');

const router = express.Router();

function publicUser(u) {
  if (!u) return null;
  return {
    id: u.id,
    username: u.username,
    display_name: u.display_name,
    photo_url: u.photo_url,
    bio: u.bio,
    status: u.status,
    age: u.age,
    city: u.city,
    country: u.country,
    created_at: u.created_at
  };
}

router.get('/me', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const u = db.prepare('SELECT * FROM users WHERE id = ?').get(req.user.id);
  res.json(publicUser(u));
});

router.put('/me', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const { display_name, photo_url, bio, status, age, city, country } = req.body || {};
  db.prepare(`
    UPDATE users SET
      display_name = COALESCE(?, display_name),
      photo_url    = COALESCE(?, photo_url),
      bio          = COALESCE(?, bio),
      status       = COALESCE(?, status),
      age          = COALESCE(?, age),
      city         = COALESCE(?, city),
      country      = COALESCE(?, country)
    WHERE id = ?
  `).run(display_name, photo_url, bio, status, age, city, country, req.user.id);
  const u = db.prepare('SELECT * FROM users WHERE id = ?').get(req.user.id);
  res.json(publicUser(u));
});

router.get('/search', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const q = `%${(req.query.q || '').toString()}%`;
  const rows = db.prepare(`
    SELECT * FROM users
    WHERE username LIKE ? OR display_name LIKE ?
    ORDER BY display_name
    LIMIT 50
  `).all(q, q);
  res.json(rows.map(publicUser));
});

router.get('/:id', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const u = db.prepare('SELECT * FROM users WHERE id = ?').get(req.params.id);
  if (!u) return res.status(404).json({ error: 'not found' });
  res.json(publicUser(u));
});

module.exports = router;
