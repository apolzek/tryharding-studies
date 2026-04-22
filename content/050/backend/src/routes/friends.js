const express = require('express');
const { requireAuth } = require('../middleware/auth');

const router = express.Router();

router.post('/request/:addresseeId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const addresseeId = Number(req.params.addresseeId);
  if (addresseeId === req.user.id) return res.status(400).json({ error: 'cannot friend yourself' });
  const target = db.prepare('SELECT id FROM users WHERE id = ?').get(addresseeId);
  if (!target) return res.status(404).json({ error: 'user not found' });

  const existing = db.prepare(`
    SELECT * FROM friendships
    WHERE (requester_id = ? AND addressee_id = ?)
       OR (requester_id = ? AND addressee_id = ?)
  `).get(req.user.id, addresseeId, addresseeId, req.user.id);
  if (existing) return res.status(409).json({ error: 'already exists', friendship: existing });

  const info = db.prepare(`
    INSERT INTO friendships (requester_id, addressee_id, status, created_at)
    VALUES (?, ?, 'pending', ?)
  `).run(req.user.id, addresseeId, Date.now());
  res.status(201).json({ id: info.lastInsertRowid, status: 'pending' });
});

router.post('/accept/:requesterId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const f = db.prepare(`
    SELECT * FROM friendships
    WHERE requester_id = ? AND addressee_id = ? AND status = 'pending'
  `).get(req.params.requesterId, req.user.id);
  if (!f) return res.status(404).json({ error: 'no pending request' });
  db.prepare(`UPDATE friendships SET status = 'accepted' WHERE id = ?`).run(f.id);
  res.json({ ok: true });
});

router.delete('/:otherUserId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const info = db.prepare(`
    DELETE FROM friendships
    WHERE (requester_id = ? AND addressee_id = ?)
       OR (requester_id = ? AND addressee_id = ?)
  `).run(req.user.id, req.params.otherUserId, req.params.otherUserId, req.user.id);
  res.json({ removed: info.changes });
});

router.get('/list/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rows = db.prepare(`
    SELECT u.id, u.username, u.display_name, u.photo_url
    FROM friendships f
    JOIN users u ON u.id = CASE WHEN f.requester_id = ? THEN f.addressee_id ELSE f.requester_id END
    WHERE (f.requester_id = ? OR f.addressee_id = ?)
      AND f.status = 'accepted'
    ORDER BY u.display_name
  `).all(req.params.userId, req.params.userId, req.params.userId);
  res.json(rows);
});

router.get('/pending', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rows = db.prepare(`
    SELECT f.id AS friendship_id, u.id, u.username, u.display_name, u.photo_url
    FROM friendships f
    JOIN users u ON u.id = f.requester_id
    WHERE f.addressee_id = ? AND f.status = 'pending'
  `).all(req.user.id);
  res.json(rows);
});

module.exports = router;
