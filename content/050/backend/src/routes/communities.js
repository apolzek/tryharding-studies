const express = require('express');
const { requireAuth } = require('../middleware/auth');

const router = express.Router();

router.post('/', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const { name, description, category, photo_url } = req.body || {};
  if (!name || !name.trim()) return res.status(400).json({ error: 'name required' });
  const now = Date.now();
  const info = db.prepare(`
    INSERT INTO communities (owner_id, name, description, category, photo_url, created_at)
    VALUES (?, ?, ?, ?, ?, ?)
  `).run(req.user.id, name.trim(), description || '', category || 'Geral', photo_url || '', now);
  db.prepare(`
    INSERT INTO community_members (community_id, user_id, joined_at) VALUES (?, ?, ?)
  `).run(info.lastInsertRowid, req.user.id, now);
  const c = db.prepare('SELECT * FROM communities WHERE id = ?').get(info.lastInsertRowid);
  res.status(201).json(c);
});

router.get('/', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const q = `%${(req.query.q || '').toString()}%`;
  const rows = db.prepare(`
    SELECT c.*, (SELECT COUNT(*) FROM community_members m WHERE m.community_id = c.id) AS member_count
    FROM communities c
    WHERE c.name LIKE ? OR c.description LIKE ? OR c.category LIKE ?
    ORDER BY c.created_at DESC
    LIMIT 100
  `).all(q, q, q);
  res.json(rows);
});

router.get('/mine', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rows = db.prepare(`
    SELECT c.*, (SELECT COUNT(*) FROM community_members m2 WHERE m2.community_id = c.id) AS member_count
    FROM communities c
    JOIN community_members m ON m.community_id = c.id
    WHERE m.user_id = ?
    ORDER BY m.joined_at DESC
  `).all(req.user.id);
  res.json(rows);
});

router.get('/user/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rows = db.prepare(`
    SELECT c.*, (SELECT COUNT(*) FROM community_members m2 WHERE m2.community_id = c.id) AS member_count
    FROM communities c
    JOIN community_members m ON m.community_id = c.id
    WHERE m.user_id = ?
    ORDER BY m.joined_at DESC
  `).all(req.params.userId);
  res.json(rows);
});

router.get('/:id', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const c = db.prepare('SELECT * FROM communities WHERE id = ?').get(req.params.id);
  if (!c) return res.status(404).json({ error: 'not found' });
  const members = db.prepare(`
    SELECT u.id, u.username, u.display_name, u.photo_url
    FROM community_members m
    JOIN users u ON u.id = m.user_id
    WHERE m.community_id = ?
    ORDER BY m.joined_at
  `).all(req.params.id);
  res.json({ ...c, members, member_count: members.length });
});

router.post('/:id/join', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const c = db.prepare('SELECT id FROM communities WHERE id = ?').get(req.params.id);
  if (!c) return res.status(404).json({ error: 'not found' });
  try {
    db.prepare(`
      INSERT INTO community_members (community_id, user_id, joined_at) VALUES (?, ?, ?)
    `).run(req.params.id, req.user.id, Date.now());
  } catch (e) {
    return res.status(409).json({ error: 'already a member' });
  }
  res.json({ ok: true });
});

router.post('/:id/leave', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const info = db.prepare(`
    DELETE FROM community_members WHERE community_id = ? AND user_id = ?
  `).run(req.params.id, req.user.id);
  res.json({ left: info.changes });
});

module.exports = router;
