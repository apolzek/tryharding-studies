const express = require('express');
const { requireAuth } = require('../middleware/auth');

const router = express.Router();

router.post('/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const profileId = Number(req.params.userId);
  if (profileId === req.user.id) return res.json({ ok: true, skipped: true });
  db.prepare(`
    INSERT INTO visits (profile_user_id, visitor_user_id, visited_at) VALUES (?, ?, ?)
  `).run(profileId, req.user.id, Date.now());
  res.json({ ok: true });
});

router.get('/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rows = db.prepare(`
    SELECT u.id, u.username, u.display_name, u.photo_url, MAX(v.visited_at) AS last_visit
    FROM visits v
    JOIN users u ON u.id = v.visitor_user_id
    WHERE v.profile_user_id = ?
    GROUP BY u.id
    ORDER BY last_visit DESC
    LIMIT 10
  `).all(req.params.userId);
  res.json(rows);
});

module.exports = router;
