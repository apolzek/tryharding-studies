const express = require('express');
const { requireAuth } = require('../middleware/auth');

const router = express.Router();

function clamp(n, min, max) {
  n = Number(n);
  if (Number.isNaN(n)) return min;
  return Math.max(min, Math.min(max, Math.trunc(n)));
}

router.put('/:userId', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rateeId = Number(req.params.userId);
  if (rateeId === req.user.id) return res.status(400).json({ error: 'cannot rate yourself' });
  const target = db.prepare('SELECT id FROM users WHERE id = ?').get(rateeId);
  if (!target) return res.status(404).json({ error: 'user not found' });

  const { trust, cool, sexy, is_fan } = req.body || {};
  const now = Date.now();

  const existing = db.prepare('SELECT * FROM ratings WHERE rater_id = ? AND ratee_id = ?').get(req.user.id, rateeId);
  const nextTrust = trust !== undefined ? clamp(trust, 0, 3) : (existing ? existing.trust : 0);
  const nextCool = cool !== undefined ? clamp(cool, 0, 3) : (existing ? existing.cool : 0);
  const nextSexy = sexy !== undefined ? clamp(sexy, 0, 3) : (existing ? existing.sexy : 0);
  const nextFan = is_fan !== undefined ? (is_fan ? 1 : 0) : (existing ? existing.is_fan : 0);

  db.prepare(`
    INSERT INTO ratings (rater_id, ratee_id, trust, cool, sexy, is_fan, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(rater_id, ratee_id) DO UPDATE SET
      trust = excluded.trust,
      cool = excluded.cool,
      sexy = excluded.sexy,
      is_fan = excluded.is_fan,
      updated_at = excluded.updated_at
  `).run(req.user.id, rateeId, nextTrust, nextCool, nextSexy, nextFan, now);

  res.json({ trust: nextTrust, cool: nextCool, sexy: nextSexy, is_fan: nextFan });
});

router.get('/:userId/summary', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const agg = db.prepare(`
    SELECT
      COALESCE(AVG(trust), 0) AS trust,
      COALESCE(AVG(cool), 0) AS cool,
      COALESCE(AVG(sexy), 0) AS sexy,
      COALESCE(SUM(is_fan), 0) AS fans,
      COUNT(*) AS raters
    FROM ratings WHERE ratee_id = ?
  `).get(req.params.userId);

  let mine = null;
  if (req.user) {
    mine = db.prepare(`SELECT trust, cool, sexy, is_fan FROM ratings WHERE rater_id = ? AND ratee_id = ?`)
      .get(req.user.id, req.params.userId) || null;
  }

  res.json({
    summary: {
      trust: Number(agg.trust.toFixed(2)),
      cool: Number(agg.cool.toFixed(2)),
      sexy: Number(agg.sexy.toFixed(2)),
      fans: agg.fans,
      raters: agg.raters
    },
    mine
  });
});

router.get('/:userId/fans', requireAuth, (req, res) => {
  const db = req.app.locals.db;
  const rows = db.prepare(`
    SELECT u.id, u.username, u.display_name, u.photo_url
    FROM ratings r
    JOIN users u ON u.id = r.rater_id
    WHERE r.ratee_id = ? AND r.is_fan = 1
  `).all(req.params.userId);
  res.json(rows);
});

module.exports = router;
