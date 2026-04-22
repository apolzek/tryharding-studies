const express = require('express');
const { requireAuth } = require('../middleware/auth');

const router = express.Router();

router.get('/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rows = db.prepare(`
    SELECT s.*, u.username AS author_username, u.display_name AS author_display_name, u.photo_url AS author_photo_url
    FROM scraps s
    JOIN users u ON u.id = s.author_user_id
    WHERE s.profile_user_id = ?
    ORDER BY s.created_at DESC
    LIMIT 200
  `).all(req.params.userId);
  res.json(rows);
});

router.post('/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const { body } = req.body || {};
  if (!body || !body.trim()) return res.status(400).json({ error: 'body required' });
  const target = db.prepare('SELECT id FROM users WHERE id = ?').get(req.params.userId);
  if (!target) return res.status(404).json({ error: 'user not found' });
  const info = db.prepare(`
    INSERT INTO scraps (profile_user_id, author_user_id, body, created_at)
    VALUES (?, ?, ?, ?)
  `).run(req.params.userId, req.user.id, body.trim(), Date.now());
  const row = db.prepare('SELECT * FROM scraps WHERE id = ?').get(info.lastInsertRowid);
  res.status(201).json(row);
});

router.delete('/:id', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const s = db.prepare('SELECT * FROM scraps WHERE id = ?').get(req.params.id);
  if (!s) return res.status(404).json({ error: 'not found' });
  if (s.author_user_id !== req.user.id && s.profile_user_id !== req.user.id) {
    return res.status(403).json({ error: 'forbidden' });
  }
  db.prepare('DELETE FROM scraps WHERE id = ?').run(req.params.id);
  res.json({ ok: true });
});

module.exports = router;
