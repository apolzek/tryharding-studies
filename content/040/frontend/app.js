// userwatch dashboard client.
//
// Fetches /api/v1/users, /api/v1/report and /api/v1/sessions on a configurable
// refresh cycle and renders the dashboard. All charts reuse the same Chart.js
// instances and update in place.

const STATE_COLOURS = {
  active:  getCSS('--active')  || '#3fb950',
  light:   getCSS('--light')   || '#58a6ff',
  network: getCSS('--network') || '#d29922',
  idle:    getCSS('--idle')    || '#484f58',
  nodata:  getCSS('--nodata')  || '#21262d',
};
function getCSS(v) {
  return getComputedStyle(document.documentElement).getPropertyValue(v).trim();
}

const charts = {};
let autoTimer = null;

function humanBytes(n) {
  if (n == null) return '0';
  const u = ['B','KB','MB','GB','TB'];
  let i = 0;
  while (n >= 1024 && i < u.length - 1) { n /= 1024; i++; }
  return (i === 0 ? n.toFixed(0) : n.toFixed(1)) + ' ' + u[i];
}

function humanNum(n) {
  if (n == null) return '0';
  if (n >= 1e6) return (n/1e6).toFixed(1) + 'M';
  if (n >= 1e3) return (n/1e3).toFixed(1) + 'K';
  return n.toString();
}

function fmtTs(ts) {
  if (!ts) return '—';
  return new Date(ts * 1000).toLocaleString();
}

function fmtDuration(sec) {
  if (!sec || sec < 1) return '0s';
  const h = Math.floor(sec/3600);
  const m = Math.floor((sec%3600)/60);
  const s = Math.floor(sec%60);
  if (h) return `${h}h ${m}m`;
  if (m) return `${m}m ${s}s`;
  return `${s}s`;
}

async function api(path) {
  const r = await fetch(path);
  if (!r.ok) throw new Error(path + ': ' + r.status);
  return r.json();
}

async function loadUsers() {
  const { users } = await api('/api/v1/users');
  const sel = document.getElementById('userSelect');
  const current = sel.value;
  sel.innerHTML = '';
  if (!users || users.length === 0) {
    const opt = document.createElement('option');
    opt.value = ''; opt.textContent = '— no data yet —'; sel.appendChild(opt);
    return null;
  }
  users.forEach(u => {
    const opt = document.createElement('option');
    opt.value = u.user;
    opt.textContent = `${u.user}  (${u.session_count} sessions)`;
    sel.appendChild(opt);
  });
  if (current && users.some(u => u.user === current)) sel.value = current;
  return sel.value;
}

async function render() {
  const user = document.getElementById('userSelect').value;
  if (!user) return;
  const windowSec = parseInt(document.getElementById('windowSelect').value, 10);
  const now = Math.floor(Date.now() / 1000);
  const from = now - windowSec;

  const [report, sessionsResp, samplesResp] = await Promise.all([
    api(`/api/v1/report?user=${encodeURIComponent(user)}&from=${from}&to=${now}`),
    api(`/api/v1/sessions?user=${encodeURIComponent(user)}&since=${from}`),
    api(`/api/v1/samples?user=${encodeURIComponent(user)}&from=${from}&to=${now}`),
  ]);

  renderSummary(report);
  renderTimeline(report);
  renderInput(report);
  renderNet(report);
  renderHour(report);
  renderLoad(samplesResp.samples || []);
  renderSessions(sessionsResp.sessions || []);
  renderPerUserNet(samplesResp.samples || []);
}

function renderSummary(rep) {
  document.getElementById('scoreValue').textContent = rep.productivity_score.toFixed(1);
  const sub = rep.productivity_score >= 70 ? 'on track'
            : rep.productivity_score >= 40 ? 'partial'
            : 'low';
  document.getElementById('scoreSub').textContent = sub;
  document.getElementById('productiveHours').textContent = rep.productive_hours.toFixed(2);
  document.getElementById('targetHours').textContent = rep.target_hours.toFixed(1);
  const pct = Math.min(100, Math.max(0, rep.target_reached_pct));
  document.getElementById('targetPct').textContent = rep.target_reached_pct.toFixed(1);
  document.getElementById('targetBar').style.width = pct + '%';
  document.getElementById('peakHour').textContent = rep.total_buckets > 0 ? (rep.peak_hour + ':00') : '—';
  document.getElementById('totalKeys').textContent = humanNum(rep.total_keys);
  document.getElementById('totalClicks').textContent = humanNum(rep.total_clicks);
  document.getElementById('totalMoves').textContent = humanNum(rep.total_mouse_moves);
  document.getElementById('totalRx').textContent = humanBytes(rep.total_rx_bytes);
  document.getElementById('totalTx').textContent = humanBytes(rep.total_tx_bytes);
}

function renderTimeline(rep) {
  const ctx = document.getElementById('timeline');
  const from = rep.window_start;
  const to = rep.window_end;
  const totalMin = Math.ceil((to - from) / 60);
  // For each minute, find state (or 'nodata').
  const bucketByStart = new Map();
  (rep.buckets || []).forEach(b => bucketByStart.set(b.start, b));
  const labels = new Array(totalMin);
  const data = new Array(totalMin).fill(1);
  const colours = new Array(totalMin);
  for (let i = 0; i < totalMin; i++) {
    const mStart = Math.floor(from/60)*60 + i*60;
    labels[i] = new Date(mStart * 1000);
    const b = bucketByStart.get(mStart);
    const st = b ? b.state : 'nodata';
    colours[i] = STATE_COLOURS[st];
  }
  const cfg = {
    type: 'bar',
    data: {
      labels,
      datasets: [{
        data,
        backgroundColor: colours,
        barPercentage: 1.0, categoryPercentage: 1.0,
      }],
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      animation: false,
      scales: {
        x: {
          display: true,
          type: 'time',
          time: { unit: 'hour' },
          grid: { display: false },
          ticks: { color: '#8b949e', maxRotation: 0 },
        },
        y: { display: false, max: 1 },
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            title: ([item]) => new Date(labels[item.dataIndex]).toLocaleString(),
            label: (item) => {
              const mStart = Math.floor(from/60)*60 + item.dataIndex*60;
              const b = bucketByStart.get(mStart);
              if (!b) return 'no data';
              return `${b.state}  keys=${b.keys_pressed} clicks=${b.clicks} moves=${b.mouse_moves} rx=${humanBytes(b.rx_bytes)} tx=${humanBytes(b.tx_bytes)}`;
            },
          },
        },
      },
    },
  };
  upsert('timeline', ctx, cfg);
}

function renderInput(rep) {
  const ctx = document.getElementById('inputChart');
  const labels = (rep.buckets || []).map(b => new Date(b.start * 1000));
  const cfg = {
    type: 'line',
    data: {
      labels,
      datasets: [
        { label: 'keys', data: (rep.buckets||[]).map(b => b.keys_pressed), borderColor: '#58a6ff', backgroundColor: 'rgba(88,166,255,0.15)', fill: true, tension: 0.25, pointRadius: 0 },
        { label: 'clicks', data: (rep.buckets||[]).map(b => b.clicks), borderColor: '#3fb950', backgroundColor: 'rgba(63,185,80,0.15)', fill: true, tension: 0.25, pointRadius: 0 },
        { label: 'mouse moves', data: (rep.buckets||[]).map(b => b.mouse_moves), borderColor: '#d29922', backgroundColor: 'rgba(210,153,34,0.10)', fill: true, tension: 0.25, pointRadius: 0 },
      ],
    },
    options: {
      responsive: true, maintainAspectRatio: false, animation: false,
      scales: {
        x: { type: 'time', grid: { color: 'rgba(255,255,255,0.03)' }, ticks: { color: '#8b949e' } },
        y: { grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#8b949e' } },
      },
      plugins: { legend: { labels: { color: '#e6edf3' } } },
    },
  };
  upsert('inputChart', ctx, cfg);
}

function renderNet(rep) {
  const ctx = document.getElementById('netChart');
  const labels = (rep.buckets || []).map(b => new Date(b.start * 1000));
  const cfg = {
    type: 'line',
    data: {
      labels,
      datasets: [
        { label: 'rx', data: (rep.buckets||[]).map(b => b.rx_bytes), borderColor: '#58a6ff', backgroundColor: 'rgba(88,166,255,0.15)', fill: true, tension: 0.25, pointRadius: 0 },
        { label: 'tx', data: (rep.buckets||[]).map(b => b.tx_bytes), borderColor: '#f85149', backgroundColor: 'rgba(248,81,73,0.15)', fill: true, tension: 0.25, pointRadius: 0 },
      ],
    },
    options: {
      responsive: true, maintainAspectRatio: false, animation: false,
      scales: {
        x: { type: 'time', grid: { color: 'rgba(255,255,255,0.03)' }, ticks: { color: '#8b949e' } },
        y: { grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#8b949e',
             callback: v => humanBytes(v) } },
      },
      plugins: {
        legend: { labels: { color: '#e6edf3' } },
        tooltip: { callbacks: { label: (c) => `${c.dataset.label}: ${humanBytes(c.parsed.y)}` } },
      },
    },
  };
  upsert('netChart', ctx, cfg);
}

function renderHour(rep) {
  const ctx = document.getElementById('hourChart');
  const cfg = {
    type: 'bar',
    data: {
      labels: Array.from({length: 24}, (_, i) => i + ':00'),
      datasets: [{ label: 'productive hours', data: rep.hour_breakdown || [],
        backgroundColor: '#3fb950' }],
    },
    options: {
      responsive: true, maintainAspectRatio: false, animation: false,
      scales: {
        x: { grid: { display: false }, ticks: { color: '#8b949e' } },
        y: { beginAtZero: true, max: 1, grid: { color: 'rgba(255,255,255,0.05)' },
          ticks: { color: '#8b949e', callback: v => v + 'h' } },
      },
      plugins: { legend: { display: false } },
    },
  };
  upsert('hourChart', ctx, cfg);
}

function renderLoad(samples) {
  const ctx = document.getElementById('loadChart');
  const labels = samples.map(s => new Date(s.window_end * 1000));
  const cfg = {
    type: 'line',
    data: {
      labels,
      datasets: [
        { label: 'CPU %', data: samples.map(s => s.cpu_used_pct), borderColor: '#f85149', backgroundColor: 'rgba(248,81,73,0.15)', fill: true, tension: 0.25, pointRadius: 0 },
        { label: 'MEM %', data: samples.map(s => s.mem_used_pct), borderColor: '#d29922', backgroundColor: 'rgba(210,153,34,0.15)', fill: true, tension: 0.25, pointRadius: 0 },
        { label: 'load1', data: samples.map(s => s.load1 * 10), borderColor: '#58a6ff', borderDash: [4,4], pointRadius: 0, tension: 0.25, yAxisID: 'y' },
      ],
    },
    options: {
      responsive: true, maintainAspectRatio: false, animation: false,
      scales: {
        x: { type: 'time', grid: { color: 'rgba(255,255,255,0.03)' }, ticks: { color: '#8b949e' } },
        y: { beginAtZero: true, max: 100, grid: { color: 'rgba(255,255,255,0.05)' },
             ticks: { color: '#8b949e' } },
      },
      plugins: { legend: { labels: { color: '#e6edf3' } } },
    },
  };
  upsert('loadChart', ctx, cfg);
}

function renderSessions(sessions) {
  const tbody = document.querySelector('#sessionsTable tbody');
  tbody.innerHTML = '';
  sessions.forEach(s => {
    const dur = (s.closed_at || s.last_seen_at) - s.started_at;
    const status = s.closed_at
      ? `<span class="badge closed">closed</span>`
      : `<span class="badge live">live</span>`;
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${s.id}</td><td>${s.host}</td>
                    <td><code>${s.machine_id.slice(0,8)}</code></td>
                    <td>${s.os}</td><td>${s.kernel}</td>
                    <td>${fmtTs(s.started_at)}</td>
                    <td>${fmtTs(s.last_seen_at)}</td>
                    <td>${fmtDuration(dur)}</td>
                    <td>${status}</td>`;
    tbody.appendChild(tr);
  });
}

function renderPerUserNet(samples) {
  const agg = new Map();
  samples.forEach(s => {
    (s.per_user_net || []).forEach(u => {
      const a = agg.get(u.uid) || { uid: u.uid, user: u.user, rx_bytes: 0, tx_bytes: 0, rx_calls: 0, tx_calls: 0 };
      a.rx_bytes += u.rx_bytes; a.tx_bytes += u.tx_bytes;
      a.rx_calls += u.rx_calls; a.tx_calls += u.tx_calls;
      agg.set(u.uid, a);
    });
  });
  const rows = [...agg.values()].sort((a, b) => (b.rx_bytes + b.tx_bytes) - (a.rx_bytes + a.tx_bytes));
  const tbody = document.querySelector('#netTable tbody');
  tbody.innerHTML = '';
  rows.forEach(r => {
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${r.uid}</td><td>${r.user}</td>
                    <td>${humanBytes(r.rx_bytes)}</td>
                    <td>${humanBytes(r.tx_bytes)}</td>
                    <td>${humanNum(r.rx_calls)}</td>
                    <td>${humanNum(r.tx_calls)}</td>`;
    tbody.appendChild(tr);
  });
}

function upsert(key, canvas, cfg) {
  if (charts[key]) {
    charts[key].data = cfg.data;
    charts[key].options = cfg.options;
    charts[key].update('none');
  } else {
    charts[key] = new Chart(canvas, cfg);
  }
}

function setAutoRefresh(on) {
  if (autoTimer) { clearInterval(autoTimer); autoTimer = null; }
  if (on) autoTimer = setInterval(refreshAll, 10000);
}

async function refreshAll() {
  try {
    if (!document.getElementById('userSelect').value) {
      await loadUsers();
    }
    await render();
  } catch (e) {
    console.error(e);
  }
}

document.addEventListener('DOMContentLoaded', async () => {
  document.getElementById('refreshBtn').addEventListener('click', refreshAll);
  document.getElementById('userSelect').addEventListener('change', render);
  document.getElementById('windowSelect').addEventListener('change', render);
  document.getElementById('autoRefresh').addEventListener('change', (e) => setAutoRefresh(e.target.checked));
  await loadUsers();
  await render().catch(console.error);
  setAutoRefresh(true);
});
