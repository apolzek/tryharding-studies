const express = require('express');
const { requireAuth } = require('../middleware/auth');

const router = express.Router();

router.get('/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rows = db.prepare(`
    SELECT t.*, u.username AS author_username, u.display_name AS author_display_name, u.photo_url AS author_photo_url
    FROM testimonials t
    JOIN users u ON u.id = t.author_user_id
    WHERE t.profile_user_id = ?
    ORDER BY t.created_at DESC
  `).all(req.params.userId);
  res.json(rows);
});

router.post('/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const { body } = req.body || {};
  if (!body || !body.trim()) return res.status(400).json({ error: 'body required' });
  if (Number(req.params.userId) === req.user.id) return res.status(400).json({ error: 'cannot testify for yourself' });
  const target = db.prepare('SELECT id FROM users WHERE id = ?').get(req.params.userId);
  if (!target) return res.status(404).json({ error: 'user not found' });
  const info = db.prepare(`
    INSERT INTO testimonials (profile_user_id, author_user_id, body, created_at)
    VALUES (?, ?, ?, ?)
  `).run(req.params.userId, req.user.id, body.trim(), Date.now());
  const row = db.prepare('SELECT * FROM testimonials WHERE id = ?').get(info.lastInsertRowid);
  res.status(201).json(row);
});

router.delete('/:id', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const t = db.prepare('SELECT * FROM testimonials WHERE id = ?').get(req.params.id);
  if (!t) return res.status(404).json({ error: 'not found' });
  if (t.author_user_id !== req.user.id && t.profile_user_id !== req.user.id) {
    return res.status(403).json({ error: 'forbidden' });
  }
  db.prepare('DELETE FROM testimonials WHERE id = ?').run(req.params.id);
  res.json({ ok: true });
});

module.exports = router;
