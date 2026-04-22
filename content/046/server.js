require('dotenv').config();
const express = require('express');
const multer = require('multer');
const path = require('path');
const fs = require('fs');

const { cfg, allMasked, update } = require('./config');
const tokens = require('./services/tokens');

const youtube = require('./services/youtube');
const tiktok = require('./services/tiktok');
const twitch = require('./services/twitch');
const instagram = require('./services/instagram');
const facebook = require('./services/facebook');
const twitter = require('./services/twitter');
const linkedin = require('./services/linkedin');

const platforms = { youtube, tiktok, twitch, instagram, facebook, twitter, linkedin };

const app = express();
const PORT = process.env.PORT || 3000;
const UPLOAD_DIR = path.join(__dirname, 'uploads');
const PUBLIC_MEDIA_DIR = path.join(__dirname, 'public-media');

for (const d of [UPLOAD_DIR, PUBLIC_MEDIA_DIR, path.join(__dirname, 'tokens')]) {
  if (!fs.existsSync(d)) fs.mkdirSync(d, { recursive: true });
}

const upload = multer({
  dest: UPLOAD_DIR,
  limits: { fileSize: 2 * 1024 * 1024 * 1024 },
});

app.use(express.json({ limit: '1mb' }));
app.use(express.static(path.join(__dirname, 'public')));
app.use('/public-media', express.static(PUBLIC_MEDIA_DIR));

app.get('/api/auth/status', (_req, res) => {
  const status = {};
  for (const [name, svc] of Object.entries(platforms)) {
    try {
      status[name] = {
        authenticated: !!svc.isAuthenticated(),
        supportsUpload: svc.supportsUpload !== false,
        flow: svc.authFlow || 'oauth',
      };
    } catch {
      status[name] = { authenticated: false, supportsUpload: false, flow: 'oauth' };
    }
  }
  res.json(status);
});

app.get('/api/auth/:platform', (req, res) => {
  const svc = platforms[req.params.platform];
  if (!svc || !svc.getAuthUrl) {
    return res.status(404).json({ error: 'Unknown platform' });
  }
  try {
    res.redirect(svc.getAuthUrl());
  } catch (e) {
    res.status(500).send(`Cannot start auth for ${req.params.platform}: ${e.message}`);
  }
});

for (const [name, svc] of Object.entries(platforms)) {
  if (!svc.handleCallback) continue;
  app.get(`/api/auth/${name}/callback`, async (req, res) => {
    try {
      await svc.handleCallback(req);
      res.redirect(`/?auth=${name}`);
    } catch (e) {
      console.error(`[${name}] auth error:`, e.response?.data || e.message);
      res.status(500).send(`Auth failed for ${name}: ${JSON.stringify(e.response?.data || e.message)}`);
    }
  });
}

app.post('/api/auth/:platform/disconnect', (req, res) => {
  const name = req.params.platform;
  if (!platforms[name]) return res.status(404).json({ error: 'Unknown platform' });
  tokens.clear(name);
  res.json({ ok: true });
});

app.get('/api/settings', (_req, res) => res.json(allMasked()));

app.post('/api/settings', (req, res) => {
  try {
    update(req.body || {});
    res.json({ ok: true, settings: allMasked() });
  } catch (e) {
    res.status(500).json({ error: e.message });
  }
});

app.post('/api/upload', upload.single('video'), async (req, res) => {
  if (!req.file) return res.status(400).json({ error: 'No video file uploaded' });
  const targets = (req.body.targets || '').split(',').map(s => s.trim()).filter(Boolean);
  if (targets.length === 0) {
    fs.unlink(req.file.path, () => {});
    return res.status(400).json({ error: 'No target platforms selected' });
  }

  const meta = {
    title: (req.body.title || 'Untitled').slice(0, 100),
    description: req.body.description || '',
    tags: (req.body.tags || '').split(',').map(t => t.trim()).filter(Boolean),
    filePath: req.file.path,
    mimeType: req.file.mimetype,
    originalName: req.file.originalname,
  };

  const results = {};
  await Promise.all(targets.map(async (name) => {
    const svc = platforms[name];
    if (!svc) { results[name] = { ok: false, error: 'Unknown platform' }; return; }
    try {
      const r = await svc.postVideo(meta);
      results[name] = { ok: true, ...r };
    } catch (e) {
      const apiErr = e.response?.data;
      results[name] = { ok: false, error: apiErr ? JSON.stringify(apiErr) : e.message };
    }
  }));

  fs.unlink(req.file.path, () => {});
  res.json(results);
});

app.listen(PORT, () => {
  console.log(`Multi-post video running at http://localhost:${PORT}`);
  if (cfg('APP_URL')) console.log(`APP_URL:    ${cfg('APP_URL')}`);
  if (cfg('PUBLIC_URL')) console.log(`PUBLIC_URL: ${cfg('PUBLIC_URL')}`);
});
