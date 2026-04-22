const express = require('express');
const { hashPassword, verifyPassword, signToken } = require('../auth');

const router = express.Router();

router.post('/register', (req, res) => {
  const db = req.app.locals.db;
  const { username, password, display_name } = req.body || {};
  if (!username || !password) return res.status(400).json({ error: 'username and password required' });
  if (username.length < 3) return res.status(400).json({ error: 'username too short' });
  if (password.length < 4) return res.status(400).json({ error: 'password too short' });

  const exists = db.prepare('SELECT id FROM users WHERE username = ?').get(username);
  if (exists) return res.status(409).json({ error: 'username taken' });

  const now = Date.now();
  const info = db.prepare(`
    INSERT INTO users (username, password_hash, display_name, created_at)
    VALUES (?, ?, ?, ?)
  `).run(username, hashPassword(password), display_name || username, now);

  const token = signToken({ sub: info.lastInsertRowid, username });
  res.status(201).json({
    token,
    user: { id: info.lastInsertRowid, username, display_name: display_name || username }
  });
});

router.post('/login', (req, res) => {
  const db = req.app.locals.db;
  const { username, password } = req.body || {};
  if (!username || !password) return res.status(400).json({ error: 'username and password required' });

  const user = db.prepare('SELECT * FROM users WHERE username = ?').get(username);
  if (!user || !verifyPassword(password, user.password_hash)) {
    return res.status(401).json({ error: 'invalid credentials' });
  }

  const token = signToken({ sub: user.id, username: user.username });
  res.json({
    token,
    user: {
      id: user.id,
      username: user.username,
      display_name: user.display_name,
      photo_url: user.photo_url
    }
  });
});

module.exports = router;
