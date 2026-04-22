/* ============================================================
 * Platform metadata
 * ========================================================== */
const PLATFORMS = {
  youtube: {
    title: 'YouTube',
    tagline: 'Upload to your channel via the Data API v3 (resumable upload).',
    callback: '/api/auth/youtube/callback',
    portal: 'https://console.cloud.google.com/apis/credentials',
    consent: 'https://console.cloud.google.com/apis/credentials/consent',
    steps: [
      'Open the Google Cloud Console and enable <b>YouTube Data API v3</b>.',
      'Create credentials → <b>OAuth client ID</b> → type <b>Web application</b>.',
      'Add the redirect URI below to <i>Authorized redirect URIs</i>.',
      'On the <a href="https://console.cloud.google.com/apis/credentials/consent" target="_blank">OAuth consent screen</a> add your Google account as a <b>Test user</b> (else refresh tokens expire in 7 days).',
    ],
    fields: [
      { key: 'YOUTUBE_CLIENT_ID', label: 'Client ID' },
      { key: 'YOUTUBE_CLIENT_SECRET', label: 'Client Secret', secret: true },
      { key: 'YOUTUBE_PRIVACY', label: 'Privacy', placeholder: 'private | unlisted | public' },
    ],
    tags: ['Video', 'OAuth 2.0', 'Google'],
  },
  tiktok: {
    title: 'TikTok',
    tagline: 'Content Posting API — inbox (draft) or direct publish.',
    callback: '/api/auth/tiktok/callback',
    portal: 'https://developers.tiktok.com/apps',
    steps: [
      'Register an app at <a href="https://developers.tiktok.com/apps" target="_blank">developers.tiktok.com</a> and add the <b>Login Kit</b> + <b>Content Posting API</b> products.',
      'Add the redirect URI below to the app Login settings.',
      'Choose a mode: <b>inbox</b> (video lands in the creator\'s TikTok drafts) or <b>direct</b> (publishes immediately — needs Content Posting API audit).',
      'For unaudited apps, <code>privacy_level</code> is forced to <code>SELF_ONLY</code>.',
    ],
    fields: [
      { key: 'TIKTOK_CLIENT_KEY', label: 'Client Key' },
      { key: 'TIKTOK_CLIENT_SECRET', label: 'Client Secret', secret: true },
      { key: 'TIKTOK_MODE', label: 'Mode', placeholder: 'inbox | direct' },
      { key: 'TIKTOK_PRIVACY', label: 'Privacy (direct mode)', placeholder: 'SELF_ONLY' },
    ],
    tags: ['Video', 'OAuth 2.0'],
  },
  twitch: {
    title: 'Twitch',
    tagline: 'Auth only. Twitch has no public VOD upload API.',
    callback: '/api/auth/twitch/callback',
    portal: 'https://dev.twitch.tv/console/apps',
    steps: [
      'Register an app at <a href="https://dev.twitch.tv/console/apps" target="_blank">dev.twitch.tv/console/apps</a>.',
      'Add the redirect URL below to <i>OAuth Redirect URLs</i>.',
      'Twitch removed the public VOD upload endpoint — this integration can authenticate only. To publish a VOD use <b>Twitch Studio</b> or <b>OBS</b>.',
    ],
    fields: [
      { key: 'TWITCH_CLIENT_ID', label: 'Client ID' },
      { key: 'TWITCH_CLIENT_SECRET', label: 'Client Secret', secret: true },
    ],
    tags: ['Auth only', 'OAuth 2.0'],
  },
  instagram: {
    title: 'Instagram',
    tagline: 'Meta Graph API — Reels container + publish.',
    callback: '/api/auth/instagram/callback',
    portal: 'https://developers.facebook.com/apps',
    steps: [
      'Create a Meta app at <a href="https://developers.facebook.com/apps" target="_blank">developers.facebook.com</a> (type <b>Business</b>).',
      'Add <b>Facebook Login for Business</b> and <b>Instagram Graph API</b> products.',
      'Add the redirect URI below to <i>Valid OAuth Redirect URIs</i>.',
      'Link a <b>Business/Creator</b> Instagram account to a Facebook Page you own.',
      'Set <code>PUBLIC_URL</code> in App Settings to a public HTTPS (ngrok) — Meta fetches the video from it.',
      'Credentials are shared with Facebook (same Meta app). Configure once there.',
    ],
    fields: [
      { key: 'META_APP_ID', label: 'Meta App ID (shared with Facebook)' },
      { key: 'META_APP_SECRET', label: 'Meta App Secret (shared)', secret: true },
    ],
    tags: ['Video', 'Reels', 'Meta'],
  },
  facebook: {
    title: 'Facebook',
    tagline: 'Meta Graph API — multipart upload to a Page.',
    callback: '/api/auth/facebook/callback',
    portal: 'https://developers.facebook.com/apps',
    steps: [
      'Same Meta app as Instagram (configure once).',
      'Add <i>Facebook Login for Business</i> product; request <code>pages_manage_posts</code>, <code>publish_video</code>, <code>pages_show_list</code>.',
      'Add the redirect URI below to <i>Valid OAuth Redirect URIs</i>.',
      'You must own (or administer) at least one Facebook Page.',
    ],
    fields: [
      { key: 'META_APP_ID', label: 'Meta App ID (shared with Instagram)' },
      { key: 'META_APP_SECRET', label: 'Meta App Secret (shared)', secret: true },
    ],
    tags: ['Video', 'Pages', 'Meta'],
  },
  twitter: {
    title: 'X (Twitter)',
    tagline: 'api.x.com/2/media/upload + /2/tweets. OAuth 1.0a user context.',
    callback: null,
    portal: 'https://developer.x.com/',
    steps: [
      'Create a Project + App at <a href="https://developer.x.com/" target="_blank">developer.x.com</a> with <b>Read and write</b> permissions.',
      'Under <i>Keys and tokens</i>, generate <b>API Key / Secret</b> (consumer) and <b>Access Token / Secret</b> for YOUR account.',
      'Generate the access tokens <b>after</b> setting Read & Write, otherwise regenerate them.',
      'No OAuth flow is run in the UI — these four keys authenticate you directly.',
      'The legacy v1.1 media endpoint was sunset on 2025-06-09; this app uses the new v2 endpoint.',
    ],
    fields: [
      { key: 'TWITTER_API_KEY', label: 'API Key' },
      { key: 'TWITTER_API_SECRET', label: 'API Secret', secret: true },
      { key: 'TWITTER_ACCESS_TOKEN', label: 'Access Token' },
      { key: 'TWITTER_ACCESS_SECRET', label: 'Access Token Secret', secret: true },
    ],
    tags: ['Video', 'Static keys', 'OAuth 1.0a'],
  },
  linkedin: {
    title: 'LinkedIn',
    tagline: '/rest/videos + /rest/posts (Videos API replaces legacy /v2/assets).',
    callback: '/api/auth/linkedin/callback',
    portal: 'https://www.linkedin.com/developers/apps',
    steps: [
      'Create an app at <a href="https://www.linkedin.com/developers/apps" target="_blank">LinkedIn Developer</a>.',
      'Request these products: <b>Sign In with LinkedIn using OpenID Connect</b> + <b>Share on LinkedIn</b>.',
      'Under <i>Auth</i> → add the redirect URL below to <i>Authorized redirect URLs</i>.',
      'Posts go out on behalf of the authenticated person (<code>w_member_social</code>). Company Pages need a separate Marketing Developer Platform app.',
    ],
    fields: [
      { key: 'LINKEDIN_CLIENT_ID', label: 'Client ID' },
      { key: 'LINKEDIN_CLIENT_SECRET', label: 'Client Secret', secret: true },
    ],
    tags: ['Video', 'OAuth 2.0 + OIDC'],
  },
};

const GENERAL_FIELDS = [
  { key: 'APP_URL', label: 'APP_URL (local base)', placeholder: 'http://localhost:3000' },
  { key: 'PUBLIC_URL', label: 'PUBLIC_URL (ngrok, required by Instagram)', placeholder: 'https://<id>.ngrok.io' },
  { key: 'META_GRAPH_VERSION', label: 'Meta Graph version', placeholder: 'v22.0' },
];

/* ============================================================
 * State
 * ========================================================== */
let statusData = {};
let settingsData = {};
const $ = (sel) => document.querySelector(sel);

/* ============================================================
 * Data fetch
 * ========================================================== */
async function fetchStatus() {
  const r = await fetch('/api/auth/status');
  statusData = await r.json();
}
async function fetchSettings() {
  const r = await fetch('/api/settings');
  settingsData = await r.json();
}

/* ============================================================
 * Router
 * ========================================================== */
const views = {
  overview: $('#view-overview'),
  detail: $('#view-detail'),
  settings: $('#view-settings'),
};

function setView(name) {
  for (const v of Object.values(views)) v.hidden = true;
  views[name].hidden = false;
  window.scrollTo({ top: 0, behavior: 'instant' });
}

async function route() {
  const hash = location.hash || '#/';
  await Promise.all([fetchStatus(), fetchSettings()]);

  document.querySelectorAll('.route-link').forEach(a => {
    a.classList.toggle('active', a.dataset.route === hash || (hash.startsWith('#/platform/') && a.dataset.route === '#/platforms'));
  });

  if (hash === '#/' || hash === '' || hash === '#/publish') {
    renderOverview();
    setView('overview');
  } else if (hash === '#/platforms') {
    renderOverview();
    setView('overview');
    document.getElementById('platforms').scrollIntoView({ behavior: 'smooth' });
  } else if (hash.startsWith('#/platform/')) {
    const slug = hash.split('/')[2];
    if (PLATFORMS[slug]) {
      renderDetail(slug);
      setView('detail');
    } else {
      location.hash = '#/platforms';
    }
  } else if (hash === '#/settings') {
    renderGeneralSettings();
    setView('settings');
  } else {
    location.hash = '#/';
  }
}

window.addEventListener('hashchange', route);
document.addEventListener('click', (e) => {
  const t = e.target.closest('[data-route]');
  if (t) {
    e.preventDefault();
    location.hash = t.dataset.route;
  }
});

/* ============================================================
 * OVERVIEW — Publish form + platforms grid
 * ========================================================== */
function renderOverview() {
  renderTargets();
  renderPlatformsGrid();
}

function renderTargets() {
  const host = $('#targets');
  host.innerHTML = '';
  for (const [slug, meta] of Object.entries(PLATFORMS)) {
    const s = statusData[slug] || {};
    const lbl = document.createElement('label');
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.name = 'targets';
    cb.value = slug;
    cb.disabled = !s.authenticated || !s.supportsUpload;
    lbl.appendChild(cb);
    lbl.append(` ${meta.title}`);
    host.appendChild(lbl);
  }
}

function renderPlatformsGrid() {
  const host = $('#platforms-grid');
  host.innerHTML = '';
  for (const [slug, meta] of Object.entries(PLATFORMS)) {
    const s = statusData[slug] || {};
    const card = document.createElement('div');
    card.className = 'brut platform-card clickable';
    card.dataset.route = `#/platform/${slug}`;

    const badge = !s.supportsUpload
      ? `<span class="badge outline">AUTH ONLY</span>`
      : s.authenticated
        ? `<span class="badge">CONNECTED</span>`
        : `<span class="badge dashed">OFFLINE</span>`;

    card.innerHTML = `
      <div class="platform-head">
        <h3>${meta.title.toLowerCase()}</h3>
        ${badge}
      </div>
      <p>${meta.tagline}</p>
      <span class="platform-link">configure</span>
    `;
    host.appendChild(card);
  }
}

/* ============================================================
 * DETAIL — per-platform page
 * ========================================================== */
function renderDetail(slug) {
  const meta = PLATFORMS[slug];
  const s = statusData[slug] || {};
  const host = $('#detail-host');

  const statusBadge = !s.supportsUpload
    ? `<span class="badge outline">AUTH ONLY</span>`
    : s.authenticated
      ? `<span class="badge">CONNECTED</span>`
      : `<span class="badge dashed">OFFLINE</span>`;

  const stepsList = meta.steps.map(s => `<li>${s}</li>`).join('');
  const tagHtml = meta.tags.map(t => `<span class="badge outline">${t}</span>`).join('');

  const fieldsHtml = meta.fields.map(f => {
    const info = settingsData[f.key] || { set: false, value: '' };
    const hint = f.secret && info.set ? `<div class="hint-line">currently set — hint ${info.hint} — leave blank to keep</div>` : '';
    const val = !f.secret ? (info.value || '').replace(/"/g, '&quot;') : '';
    return `
      <div class="field">
        <label class="lbl">${f.label}</label>
        <input type="${f.secret ? 'password' : 'text'}"
               name="${f.key}"
               placeholder="${f.placeholder || (f.secret && info.set ? '••••••••' : '')}"
               value="${val}">
        ${hint}
      </div>
    `;
  }).join('');

  const callbackUrl = meta.callback ? `${(settingsData.APP_URL?.value || location.origin)}${meta.callback}` : null;
  const callbackBlock = callbackUrl ? `
    <div class="step">
      <h3><span class="num">02</span> Redirect URI</h3>
      <p style="font-size:0.9rem;line-height:1.8;color:var(--mid);margin-bottom:0.8rem">
        Paste this <b>exactly</b> into your app's authorized redirect URIs list at the provider.
      </p>
      <div class="callback-box">
        <span class="url">${callbackUrl}</span>
        <button class="copy-btn" id="copy-callback">Copy</button>
      </div>
    </div>
  ` : `
    <div class="step">
      <h3><span class="num">02</span> No redirect URI</h3>
      <p style="font-size:0.9rem;line-height:1.8;color:var(--mid)">
        This integration uses static keys — no OAuth redirect to configure.
      </p>
    </div>
  `;

  const authBtn = meta.callback
    ? (s.authenticated
        ? `<a href="/api/auth/${slug}" class="btn ghost">Reconnect</a>`
        : `<a href="/api/auth/${slug}" class="btn">Connect →</a>`)
    : '';
  const disconnectBtn = s.authenticated
    ? `<button class="btn ghost danger" id="disconnect-btn">Disconnect</button>`
    : '';

  host.innerHTML = `
    <div class="detail-head">
      <div>
        <h2>${meta.title}</h2>
        <div class="tags">${tagHtml}</div>
      </div>
      <div class="status-line ${s.authenticated ? '' : 'off'}">
        <span class="dot"></span>
        ${statusBadge}
      </div>
    </div>
    <p style="margin-top:0.8rem;color:var(--mid);font-family:'Space Mono',monospace;font-size:0.85rem">
      ${meta.tagline}
    </p>

    <div class="detail-body">
      <div class="step">
        <h3><span class="num">01</span> How to obtain credentials</h3>
        <ol>${stepsList}</ol>
      </div>

      ${callbackBlock}

      <div class="step">
        <h3><span class="num">03</span> Paste credentials</h3>
        <form id="creds-form" class="creds-form">${fieldsHtml}</form>
        <div class="detail-actions" style="margin-top:1.1rem">
          <button class="btn" id="creds-save">Save</button>
          <span class="inline-msg" id="creds-status"></span>
        </div>
      </div>

      <div class="step">
        <h3><span class="num">04</span> Authorize</h3>
        <div class="detail-actions">
          ${authBtn}
          ${disconnectBtn}
        </div>
      </div>
    </div>
  `;

  // Copy callback
  const copyBtn = document.getElementById('copy-callback');
  if (copyBtn) {
    copyBtn.onclick = async () => {
      try {
        await navigator.clipboard.writeText(callbackUrl);
        copyBtn.textContent = 'Copied';
        setTimeout(() => { copyBtn.textContent = 'Copy'; }, 1500);
      } catch { copyBtn.textContent = 'Failed'; }
    };
  }

  // Save credentials (only this platform's fields)
  document.getElementById('creds-save').onclick = async () => {
    const patch = {};
    document.querySelectorAll('#creds-form input').forEach(i => {
      if (i.type === 'password' && i.value === '') return;
      patch[i.name] = i.value;
    });
    const statusEl = document.getElementById('creds-status');
    statusEl.textContent = 'Saving…';
    try {
      const r = await fetch('/api/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(patch),
      });
      if (!r.ok) throw new Error('save failed');
      statusEl.textContent = 'Saved ✓';
      await fetchSettings();
      setTimeout(() => { statusEl.textContent = ''; }, 2500);
    } catch (e) {
      statusEl.textContent = 'Error: ' + e.message;
    }
  };

  // Disconnect
  const dc = document.getElementById('disconnect-btn');
  if (dc) {
    dc.onclick = async () => {
      await fetch(`/api/auth/${slug}/disconnect`, { method: 'POST' });
      await fetchStatus();
      renderDetail(slug);
    };
  }
}

/* ============================================================
 * GENERAL SETTINGS
 * ========================================================== */
function renderGeneralSettings() {
  const host = $('#general-form');
  host.innerHTML = GENERAL_FIELDS.map(f => {
    const info = settingsData[f.key] || { set: false, value: '' };
    const val = (info.value || '').replace(/"/g, '&quot;');
    return `
      <div class="field">
        <label class="lbl">${f.label}</label>
        <input type="text" name="${f.key}" placeholder="${f.placeholder || ''}" value="${val}">
      </div>
    `;
  }).join('');
}

document.getElementById('general-save').onclick = async () => {
  const patch = {};
  document.querySelectorAll('#general-form input').forEach(i => { patch[i.name] = i.value; });
  const st = document.getElementById('general-status');
  st.textContent = 'Saving…';
  try {
    const r = await fetch('/api/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(patch),
    });
    if (!r.ok) throw new Error('save failed');
    st.textContent = 'Saved ✓';
    await fetchSettings();
    renderGeneralSettings();
    setTimeout(() => { st.textContent = ''; }, 2500);
  } catch (e) {
    st.textContent = 'Error: ' + e.message;
  }
};

/* ============================================================
 * Publish (upload) form
 * ========================================================== */
const videoInput = document.getElementById('video-input');
const fileName = document.getElementById('file-name');
videoInput.addEventListener('change', () => {
  const f = videoInput.files[0];
  if (f) {
    fileName.textContent = f.name;
    fileName.classList.add('has-file');
  } else {
    fileName.textContent = 'No file chosen';
    fileName.classList.remove('has-file');
  }
});

const form = document.getElementById('upload-form');
const resultsCard = document.getElementById('results-card');
const resultsEl = document.getElementById('results');
const submitBtn = document.getElementById('submit-btn');

form.addEventListener('submit', async (e) => {
  e.preventDefault();
  const targets = Array.from(form.querySelectorAll('input[name=targets]:checked')).map(i => i.value);
  if (targets.length === 0) { alert('Select at least one platform.'); return; }

  const fd = new FormData(form);
  fd.delete('targets');
  fd.append('targets', targets.join(','));

  submitBtn.disabled = true;
  submitBtn.textContent = 'Working…';
  resultsCard.hidden = false;
  resultsEl.textContent = 'Uploading…';

  try {
    const res = await fetch('/api/upload', { method: 'POST', body: fd });
    const data = await res.json();
    resultsEl.textContent = JSON.stringify(data, null, 2);
  } catch (err) {
    resultsEl.textContent = 'Error: ' + err.message;
  } finally {
    submitBtn.disabled = false;
    submitBtn.textContent = 'Post ▶';
  }
});

/* ============================================================
 * Typewriter hero sub
 * ========================================================== */
(function() {
  const el = document.getElementById('typewriter-text');
  if (!el) return;

  function currentLine() {
    const total = Object.keys(PLATFORMS).length;
    const connected = Object.entries(statusData)
      .filter(([, s]) => s.authenticated && s.supportsUpload)
      .map(([k]) => k);
    if (connected.length === 0) {
      return '0 / ' + total + ' connected — configure a platform to start posting.';
    }
    return connected.length + ' / ' + total + ' connected: ' + connected.join(', ') + '.';
  }

  const statics = [
    'Upload once. Publish everywhere.',
    'No browser tabs. No copy-paste. Just post.',
  ];
  let idx = 0;

  function typeText(text, done) {
    let i = 0;
    el.textContent = '';
    const t = setInterval(() => {
      el.textContent += text.charAt(i++);
      if (i >= text.length) { clearInterval(t); done && done(); }
    }, 18);
  }
  function cycle() {
    setTimeout(() => {
      el.classList.add('selected');
      setTimeout(() => {
        el.classList.remove('selected');
        const lines = [currentLine(), ...statics];
        idx = (idx + 1) % lines.length;
        typeText(lines[idx], cycle);
      }, 650);
    }, 4200);
  }
  setTimeout(() => {
    const lines = [currentLine(), ...statics];
    typeText(lines[0], cycle);
  }, 400);
})();

/* ============================================================
 * 3D title parallax
 * ========================================================== */
(function() {
  const title = document.querySelector('.title-3d');
  if (!title) return;
  const hero = document.querySelector('.hero');
  if (!hero) return;
  hero.addEventListener('mousemove', (e) => {
    const r = hero.getBoundingClientRect();
    const x = ((e.clientX - r.left) / r.width - 0.5) * 2;
    const y = ((e.clientY - r.top) / r.height - 0.5) * 2;
    title.style.animation = 'none';
    title.style.transform = `rotateX(${10 - y * 8}deg) rotateY(${x * 12}deg)`;
  });
  hero.addEventListener('mouseleave', () => {
    title.style.animation = '';
    title.style.transform = '';
  });
})();

/* ============================================================
 * Scroll reveal
 * ========================================================== */
const revealObserver = new IntersectionObserver((entries) => {
  entries.forEach(e => {
    if (e.isIntersecting) { e.target.classList.add('reveal-visible'); revealObserver.unobserve(e.target); }
  });
}, { threshold: 0.08, rootMargin: '0px 0px -40px 0px' });
document.querySelectorAll('section').forEach(s => {
  s.classList.add('reveal-hidden');
  revealObserver.observe(s);
});

/* ============================================================
 * Handle OAuth callback redirects (?auth=xxx)
 * ========================================================== */
const params = new URLSearchParams(location.search);
if (params.get('auth')) {
  const slug = params.get('auth');
  history.replaceState({}, '', `/#/platform/${slug}`);
}

/* ============================================================
 * Boot
 * ========================================================== */
route();
