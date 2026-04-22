const express = require('express');
const { requireAuth } = require('../middleware/auth');

const router = express.Router();

router.get('/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rows = db.prepare(`SELECT * FROM photos WHERE user_id = ? ORDER BY created_at DESC`).all(req.params.userId);
  res.json(rows);
});

router.post('/', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const { url, caption } = req.body || {};
  if (!url || !url.trim()) return res.status(400).json({ error: 'url required' });
  const info = db.prepare(`
    INSERT INTO photos (user_id, url, caption, created_at) VALUES (?, ?, ?, ?)
  `).run(req.user.id, url.trim(), caption || '', Date.now());
  const row = db.prepare('SELECT * FROM photos WHERE id = ?').get(info.lastInsertRowid);
  res.status(201).json(row);
});

router.delete('/:id', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const p = db.prepare('SELECT * FROM photos WHERE id = ?').get(req.params.id);
  if (!p) return res.status(404).json({ error: 'not found' });
  if (p.user_id !== req.user.id) return res.status(403).json({ error: 'forbidden' });
  db.prepare('DELETE FROM photos WHERE id = ?').run(req.params.id);
  res.json({ ok: true });
});

module.exports = router;
