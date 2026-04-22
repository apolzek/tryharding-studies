const fs = require('fs');
const path = require('path');

const FILE = path.join(__dirname, 'config.json');

const KEYS = [
  'APP_URL', 'PUBLIC_URL',
  'YOUTUBE_CLIENT_ID', 'YOUTUBE_CLIENT_SECRET', 'YOUTUBE_PRIVACY',
  'TIKTOK_CLIENT_KEY', 'TIKTOK_CLIENT_SECRET', 'TIKTOK_PRIVACY', 'TIKTOK_MODE',
  'TWITCH_CLIENT_ID', 'TWITCH_CLIENT_SECRET',
  'META_APP_ID', 'META_APP_SECRET', 'META_GRAPH_VERSION',
  'TWITTER_API_KEY', 'TWITTER_API_SECRET',
  'TWITTER_ACCESS_TOKEN', 'TWITTER_ACCESS_SECRET',
  'LINKEDIN_CLIENT_ID', 'LINKEDIN_CLIENT_SECRET',
];

const SECRET_KEYS = new Set([
  'YOUTUBE_CLIENT_SECRET', 'TIKTOK_CLIENT_SECRET', 'TWITCH_CLIENT_SECRET',
  'META_APP_SECRET', 'TWITTER_API_SECRET', 'TWITTER_ACCESS_SECRET',
  'LINKEDIN_CLIENT_SECRET',
]);

let disk = {};
function load() {
  try {
    disk = fs.existsSync(FILE) ? JSON.parse(fs.readFileSync(FILE, 'utf8')) : {};
  } catch {
    disk = {};
  }
}
load();

function cfg(k) {
  const envVal = process.env[k];
  if (envVal != null && envVal !== '') return envVal;
  const v = disk[k];
  if (v) return v;
  if (k === 'APP_URL') return `http://localhost:${process.env.PORT || 3000}`;
  return '';
}

function all() {
  const out = {};
  for (const k of KEYS) out[k] = cfg(k);
  return out;
}

function allMasked() {
  const out = {};
  for (const k of KEYS) {
    const v = cfg(k);
    if (!v) { out[k] = { set: false, value: '' }; continue; }
    if (SECRET_KEYS.has(k)) out[k] = { set: true, value: '', hint: `${v.slice(0, 3)}…${v.slice(-3)}` };
    else out[k] = { set: true, value: v };
  }
  return out;
}

function update(patch) {
  const next = { ...disk };
  for (const [k, v] of Object.entries(patch || {})) {
    if (!KEYS.includes(k)) continue;
    if (v == null || v === '') delete next[k];
    else next[k] = String(v).trim();
  }
  fs.writeFileSync(FILE, JSON.stringify(next, null, 2));
  disk = next;
}

module.exports = { cfg, all, allMasked, update, KEYS, SECRET_KEYS };
