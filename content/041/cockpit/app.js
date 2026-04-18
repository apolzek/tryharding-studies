// Cockpit · Chart.js v4 · minimalista.
// 5 charts, sem animação, sem 200 queries em paralelo.
// Desenha tudo num único rAF após carregar Chart.js.

const PROM = "/api/v1";
const RANGE_SECONDS = { "5m": 300, "15m": 900, "30m": 1800, "1h": 3600, "3h": 10800 };

// Paleta (alinhada ao CSS).
const C = {
  blue: "#60a5fa", cyan: "#22d3ee", amber: "#fbbf24", red: "#f87171",
  green: "#34d399", mag: "#c084fc", yellow: "#facc15", pink: "#f472b6",
  fg2: "#8a92a0", fg3: "#5c6472", border: "#2a2f3a", grid: "rgba(255,255,255,0.05)",
};
const INST_COLORS = [C.blue, C.green, C.amber, C.mag, C.cyan, C.pink, C.yellow, C.red];
const SEV_COLORS = { info: C.green, warn: C.amber, error: C.red };

const state = {
  instances: [],
  selectedInstances: new Set(),
  range: "15m",
  refreshSeconds: 30,
  timer: null,
  charts: {},
  booted: false,
};

// ============ Formatters (Grafana-style) ============

function fmtShort(v) {
  if (v === null || v === undefined || !isFinite(v)) return "—";
  const a = Math.abs(v);
  if (a === 0) return "0";
  if (a >= 1e12) return (v / 1e12).toFixed(2) + " T";
  if (a >= 1e9)  return (v / 1e9).toFixed(2) + " G";
  if (a >= 1e6)  return (v / 1e6).toFixed(2) + " M";
  if (a >= 1e3)  return (v / 1e3).toFixed(2) + " K";
  if (a >= 100)  return v.toFixed(0);
  if (a >= 10)   return v.toFixed(1);
  if (a >= 1)    return v.toFixed(2);
  if (a >= 0.01) return v.toFixed(3);
  return v.toExponential(1);
}
function fmtAutoRate(v) {
  if (v === null || !isFinite(v)) return "—";
  const a = Math.abs(v);
  if (a === 0) return "0/s";
  if (a >= 1) return fmtShort(v) + "/s";
  if (a * 60 >= 1) return fmtShort(v * 60) + "/min";
  return fmtShort(v * 3600) + "/h";
}
function fmtMs(v) {
  if (v === null || !isFinite(v)) return "—";
  if (v >= 60000) return (v / 60000).toFixed(1) + " min";
  if (v >= 1000)  return (v / 1000).toFixed(2) + " s";
  if (v >= 10)    return v.toFixed(0) + " ms";
  if (v >= 1)     return v.toFixed(1) + " ms";
  return v.toFixed(2) + " ms";
}
function fmtPct(v) {
  if (v === null || !isFinite(v)) return "—";
  if (v >= 10) return v.toFixed(1) + "%";
  if (v >= 1)  return v.toFixed(2) + "%";
  return v.toFixed(3) + "%";
}
function fmtBytes(v) {
  if (v === null || !isFinite(v)) return "—";
  const a = Math.abs(v), K = 1024;
  if (a >= K**4) return (v/K**4).toFixed(2) + " TB";
  if (a >= K**3) return (v/K**3).toFixed(2) + " GB";
  if (a >= K**2) return (v/K**2).toFixed(2) + " MB";
  if (a >= K)    return (v/K).toFixed(1) + " KB";
  return v.toFixed(0) + " B";
}
function fmtDuration(s) {
  if (s === null || !isFinite(s)) return "—";
  if (s >= 86400) return (s / 86400).toFixed(1) + "d";
  if (s >= 3600)  return (s / 3600).toFixed(1) + "h";
  if (s >= 60)    return (s / 60).toFixed(0) + "m";
  return s.toFixed(0) + "s";
}

// ============ Prom HTTP API ============

async function promGet(path, params) {
  const url = new URL(PROM + path, window.location.origin);
  if (params) for (const [k, v] of Object.entries(params))
    Array.isArray(v) ? v.forEach((vv) => url.searchParams.append(k, vv)) : url.searchParams.set(k, v);
  const res = await fetch(url);
  if (!res.ok) throw new Error(`prom ${path} HTTP ${res.status}`);
  const j = await res.json();
  if (j.status !== "success") throw new Error("prom error");
  return j.data;
}
const promInstant = (q) => promGet("/query", { query: q });
const promLabelValues = (lb, match) => promGet(`/label/${lb}/values`, match ? { "match[]": match } : {});
async function promRange(q, rangeKey) {
  const secs = RANGE_SECONDS[rangeKey] || 900;
  const end = Math.floor(Date.now() / 1000);
  const start = end - secs;
  const step = Math.max(15, Math.floor(secs / 80));
  return promGet("/query_range", { query: q, start, end, step });
}
async function scalar(q) {
  const d = await promInstant(q).catch(() => ({ result: [] }));
  const v = d.result?.[0]?.value?.[1];
  return v !== undefined && isFinite(+v) ? +v : null;
}

// ============ Filtros ============

function instanceMatcher() {
  if (state.selectedInstances.size === 0) return "";
  const rx = [...state.selectedInstances].map(i => i.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("|");
  return `instance=~"${rx}"`;
}
const sel = (...ms) => {
  const inner = ms.filter(Boolean).join(",");
  return inner ? `{${inner}}` : "";
};
const rateWin = () => {
  const s = RANGE_SECONDS[state.range] || 900;
  return s <= 300 ? "1m" : s <= 1800 ? "2m" : "5m";
};

// ============ UI ============

function renderInstanceChips() {
  const root = document.getElementById("instance-chips");
  root.innerHTML = "";
  const mk = (label, key, active) => {
    const b = document.createElement("button");
    b.className = "chip" + (active ? " active" : "");
    b.textContent = label;
    b.onclick = () => {
      if (key === "__all__") {
        if (state.selectedInstances.size === state.instances.length) state.selectedInstances.clear();
        else state.instances.forEach(i => state.selectedInstances.add(i));
      } else if (state.selectedInstances.has(key)) state.selectedInstances.delete(key);
      else state.selectedInstances.add(key);
      renderInstanceChips();
      refreshAll();
    };
    return b;
  };
  root.appendChild(mk("all", "__all__", state.selectedInstances.size === state.instances.length));
  state.instances.forEach(i => {
    const short = i.replace(/^otel-collector-/, "c");
    root.appendChild(mk(short, i, state.selectedInstances.has(i)));
  });
}

function bindFilters() {
  document.getElementById("range-select").onchange = e => { state.range = e.target.value; refreshAll(); };
  document.getElementById("refresh-select").onchange = e => { state.refreshSeconds = +e.target.value; scheduleRefresh(); };
  document.getElementById("refresh-now").onclick = () => refreshAll();
}
function scheduleRefresh() {
  if (state.timer) clearInterval(state.timer);
  if (state.refreshSeconds > 0) state.timer = setInterval(refreshAll, state.refreshSeconds * 1000);
}

// ============ Chart.js helpers ============

function commonOptions(extra = {}) {
  return {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,                  // nada de animação
    normalized: true,
    parsing: false,                    // esperamos {x,y} já prontos
    spanGaps: true,
    interaction: { mode: "nearest", intersect: false },
    plugins: {
      legend: {
        position: "bottom",
        labels: { color: C.fg2, boxWidth: 10, boxHeight: 10, font: { size: 10, family: "ui-monospace, Menlo, monospace" } },
      },
      tooltip: {
        backgroundColor: "#0b0d10",
        borderColor: C.border,
        borderWidth: 1,
        titleColor: "#e8eaed",
        bodyColor: "#c0c4cc",
        titleFont: { family: "ui-monospace, Menlo, monospace", size: 11 },
        bodyFont:  { family: "ui-monospace, Menlo, monospace", size: 11 },
      },
    },
    scales: {
      x: {
        type: "time",
        time: { displayFormats: { second: "HH:mm:ss", minute: "HH:mm", hour: "HH:mm" } },
        ticks: { color: C.fg2, maxRotation: 0, autoSkip: true, font: { size: 10, family: "ui-monospace, monospace" } },
        grid:  { color: C.grid, drawTicks: false },
        border: { color: C.border },
      },
      y: {
        ticks: { color: C.fg2, font: { size: 10, family: "ui-monospace, monospace" }, callback: v => fmtShort(+v) },
        grid:  { color: C.grid, drawTicks: false },
        border: { color: C.border },
      },
    },
    ...extra,
  };
}

function mkOrUpdate(id, type, data, opts) {
  // Chart.js não requer date adapter nativo para x do tipo "time";
  // como precisamos, usamos eixo "linear" com labels pré-formatadas
  // quando não há adapter disponível. Fallback: converter x para index.
  const ctx = document.getElementById(id);
  if (!ctx) return;
  if (state.charts[id]) {
    state.charts[id].data = data;
    state.charts[id].options = opts;
    state.charts[id].update("none");
    return;
  }
  state.charts[id] = new Chart(ctx, { type, data, options: opts });
}

// ============ Queries ============

const qThroughputSpans   = () => `sum(rate(otelcol_receiver_accepted_spans_total${sel(instanceMatcher())}[${rateWin()}]))`;
const qThroughputLogs    = () => `sum(rate(otelcol_receiver_accepted_log_records_total${sel(instanceMatcher())}[${rateWin()}]))`;
const qThroughputMetrics = () => `sum(rate(otelcol_receiver_accepted_metric_points_total${sel(instanceMatcher())}[${rateWin()}]))`;
const qLogsBySev         = () => `sum by (severity_bucket) (rate(otelpoc_logs_by_severity_total${sel(instanceMatcher())}[${rateWin()}]))`;
const qErrorsByInst      = () => `sum by (instance) (rate(otelpoc_spans_errors_total${sel(instanceMatcher())}[${rateWin()}])) * 60`;
const qP95Inst           = () => `histogram_quantile(0.95, sum by (le, instance) (rate(traces_spanmetrics_duration_milliseconds_bucket${sel(instanceMatcher())}[${rateWin()}])))`;
const qMemInst           = () => `max by (instance) (otelcol_process_memory_rss_bytes${sel(instanceMatcher())})`;

// Matrix → Chart.js points {x: ms, y: num}
function toPoints(matrixResult, keyFn) {
  return matrixResult.map(r => ({
    label: keyFn(r.metric),
    data: r.values.map(([t, v]) => ({ x: t * 1000, y: +v || 0 })),
  }));
}
function toSinglePoints(matrixResult, label) {
  const row = matrixResult[0];
  if (!row) return { label, data: [] };
  return { label, data: row.values.map(([t, v]) => ({ x: t * 1000, y: +v || 0 })) };
}
const instLabel = m => (m.instance || "?").replace(/^otel-collector-/, "c");
const sevOrder = { info: 0, warn: 1, error: 2 };

// ============ Discovery ============

async function discoverInstances() {
  const values = await promLabelValues("instance", 'up{job="collector-app"}');
  state.instances = (values || []).sort();
  if (state.selectedInstances.size === 0) state.instances.forEach(i => state.selectedInstances.add(i));
  renderInstanceChips();
}

// ============ Panels ============

async function panelKPIs() {
  const im = instanceMatcher(); const s = sel(im);
  const Q = {
    spansT:   `sum(otelcol_receiver_accepted_spans_total${s})`,
    logsT:    `sum(otelcol_receiver_accepted_log_records_total${s})`,
    metricsT: `sum(otelcol_receiver_accepted_metric_points_total${s})`,
    errorsT:  `sum(otelpoc_spans_errors_total${s})`,
    spansR:   `sum(rate(otelcol_receiver_accepted_spans_total${s}[2m]))`,
    logsR:    `sum(rate(otelcol_receiver_accepted_log_records_total${s}[2m]))`,
    metricsR: `sum(rate(otelcol_receiver_accepted_metric_points_total${s}[2m]))`,
    errorsR:  `sum(rate(otelpoc_spans_errors_total${s}[2m]))`,
    totalSpR: `sum(rate(otelpoc_spans_total${s}[2m]))`,
    p95:      `histogram_quantile(0.95, sum by (le) (rate(traces_spanmetrics_duration_milliseconds_bucket${s}[5m])))`,
    up:       `sum(up{job="collector-app"${im ? "," + im : ""}})`,
    total:    `count(up{job="collector-app"})`,
  };
  const pairs = await Promise.all(Object.entries(Q).map(async ([k, q]) => [k, await scalar(q)]));
  const v = Object.fromEntries(pairs);

  const set = (id, t) => { const e = document.getElementById(id); if (e) e.textContent = t; };
  set("kpi-spans-total",   fmtShort(v.spansT));
  set("kpi-spans-sub",     fmtAutoRate(v.spansR || 0));
  set("kpi-logs-total",    fmtShort(v.logsT));
  set("kpi-logs-sub",      fmtAutoRate(v.logsR || 0));
  set("kpi-metrics-total", fmtShort(v.metricsT));
  set("kpi-metrics-sub",   fmtAutoRate(v.metricsR || 0));
  set("kpi-errors-total",  fmtShort(v.errorsT));
  const errPct = v.totalSpR && v.totalSpR > 0 ? (v.errorsR / v.totalSpR) * 100 : 0;
  set("kpi-errors-sub", `error rate ${fmtPct(errPct)}`);
  set("kpi-p95",    fmtMs(v.p95));
  set("kpi-up",     `${v.up ?? 0}/${v.total ?? 0}`);
  set("kpi-up-sub", v.up === v.total ? "all active" : `${v.up || 0} of ${v.total || 0}`);

  // Ribbon
  const total = (v.spansT || 0) + (v.logsT || 0) + (v.metricsT || 0);
  const pct = x => (total > 0 ? (x / total) * 100 : 0);
  const ps = pct(v.spansT || 0), pl = pct(v.logsT || 0), pm = pct(v.metricsT || 0);
  document.getElementById("ribbon-spans").style.flexGrow   = Math.max(ps, 0.5);
  document.getElementById("ribbon-logs").style.flexGrow    = Math.max(pl, 0.5);
  document.getElementById("ribbon-metrics").style.flexGrow = Math.max(pm, 0.5);
  document.getElementById("ribbon-spans").textContent   = ps > 10 ? "spans " + fmtPct(ps) : "";
  document.getElementById("ribbon-logs").textContent    = pl > 10 ? "logs " + fmtPct(pl) : "";
  document.getElementById("ribbon-metrics").textContent = pm > 10 ? "metrics " + fmtPct(pm) : "";
  set("legend-spans",   `spans ${fmtShort(v.spansT)} · ${fmtPct(ps)}`);
  set("legend-logs",    `logs ${fmtShort(v.logsT)} · ${fmtPct(pl)}`);
  set("legend-metrics", `metrics ${fmtShort(v.metricsT)} · ${fmtPct(pm)}`);
  set("ribbon-totals",  `total ${fmtShort(total)} accepted`);
}

async function panelThroughput() {
  const [ds, dl, dm] = await Promise.all([
    promRange(qThroughputSpans(), state.range),
    promRange(qThroughputLogs(), state.range),
    promRange(qThroughputMetrics(), state.range),
  ]);
  const mk = (d, label, color) => {
    const row = d.result?.[0];
    return {
      label,
      data: row ? row.values.map(([t, v]) => ({ x: t * 1000, y: +v || 0 })) : [],
      borderColor: color,
      backgroundColor: color + "33",
      borderWidth: 1.5,
      pointRadius: 0,
      pointHoverRadius: 3,
      tension: 0.25,
      fill: true,
    };
  };
  mkOrUpdate("chart-throughput", "line", {
    datasets: [mk(ds, "spans", C.blue), mk(dl, "logs", C.cyan), mk(dm, "metrics", C.amber)],
  }, commonOptions({
    plugins: {
      ...commonOptions().plugins,
      tooltip: {
        ...commonOptions().plugins.tooltip,
        callbacks: { label: ctx => `${ctx.dataset.label}: ${fmtAutoRate(ctx.parsed.y)}` },
      },
    },
    scales: {
      ...commonOptions().scales,
      y: { ...commonOptions().scales.y, ticks: { ...commonOptions().scales.y.ticks, callback: v => fmtShort(+v) + "/s" } },
    },
  }));
}

async function panelSeverity() {
  const d = await promRange(qLogsBySev(), state.range);
  const rows = (d.result || []).sort((a, b) => (sevOrder[a.metric.severity_bucket] ?? 9) - (sevOrder[b.metric.severity_bucket] ?? 9));
  const datasets = rows.map(r => {
    const sev = r.metric.severity_bucket || "?";
    return {
      label: sev,
      data: r.values.map(([t, v]) => ({ x: t * 1000, y: +v || 0 })),
      borderColor: SEV_COLORS[sev] || C.fg2,
      backgroundColor: (SEV_COLORS[sev] || C.fg2) + "40",
      borderWidth: 1,
      pointRadius: 0,
      stack: "sev",
    };
  });
  mkOrUpdate("chart-severity", "bar", { datasets }, commonOptions({
    scales: {
      ...commonOptions().scales,
      x: { ...commonOptions().scales.x, stacked: true },
      y: { ...commonOptions().scales.y, stacked: true, ticks: { ...commonOptions().scales.y.ticks, callback: v => fmtShort(+v) + "/s" } },
    },
  }));
}

async function panelErrors() {
  const d = await promRange(qErrorsByInst(), state.range);
  const series = toPoints(d.result || [], instLabel);
  const datasets = series.map((s, i) => ({
    label: s.label,
    data: s.data,
    borderColor: INST_COLORS[i % INST_COLORS.length],
    backgroundColor: INST_COLORS[i % INST_COLORS.length],
    borderWidth: 0,
    pointRadius: 0,
  }));
  mkOrUpdate("chart-errors", "bar", { datasets }, commonOptions({
    scales: {
      ...commonOptions().scales,
      y: { ...commonOptions().scales.y, ticks: { ...commonOptions().scales.y.ticks, callback: v => fmtShort(+v) + "/min" } },
    },
    plugins: {
      ...commonOptions().plugins,
      tooltip: {
        ...commonOptions().plugins.tooltip,
        callbacks: { label: ctx => `${ctx.dataset.label}: ${fmtShort(ctx.parsed.y)}/min` },
      },
    },
  }));
}

async function panelLatency() {
  const d = await promRange(qP95Inst(), state.range);
  const series = toPoints(d.result || [], instLabel);
  const datasets = series.map((s, i) => ({
    label: s.label,
    data: s.data,
    borderColor: INST_COLORS[i % INST_COLORS.length],
    backgroundColor: "transparent",
    borderWidth: 1.5,
    pointRadius: 0,
    tension: 0.25,
  }));
  mkOrUpdate("chart-latency", "line", { datasets }, commonOptions({
    plugins: {
      ...commonOptions().plugins,
      tooltip: {
        ...commonOptions().plugins.tooltip,
        callbacks: { label: ctx => `${ctx.dataset.label}: ${fmtMs(ctx.parsed.y)}` },
      },
    },
    scales: {
      ...commonOptions().scales,
      y: { ...commonOptions().scales.y, ticks: { ...commonOptions().scales.y.ticks, callback: v => fmtMs(+v) } },
    },
  }));
}

async function panelMem() {
  const d = await promRange(qMemInst(), state.range);
  const series = toPoints(d.result || [], instLabel);
  const datasets = series.map((s, i) => ({
    label: s.label,
    data: s.data,
    borderColor: INST_COLORS[i % INST_COLORS.length],
    backgroundColor: INST_COLORS[i % INST_COLORS.length] + "22",
    borderWidth: 1.5,
    pointRadius: 0,
    tension: 0.2,
    fill: true,
  }));
  mkOrUpdate("chart-mem", "line", { datasets }, commonOptions({
    plugins: {
      ...commonOptions().plugins,
      tooltip: {
        ...commonOptions().plugins.tooltip,
        callbacks: { label: ctx => `${ctx.dataset.label}: ${fmtBytes(ctx.parsed.y)}` },
      },
    },
    scales: {
      ...commonOptions().scales,
      y: { ...commonOptions().scales.y, ticks: { ...commonOptions().scales.y.ticks, callback: v => fmtBytes(+v) } },
    },
  }));
}

async function panelTable() {
  const im = instanceMatcher(); const s = sel(im); const w = rateWin();
  const Q = {
    up:      `up{job="collector-app"${im ? "," + im : ""}}`,
    spans:   `sum by (instance) (rate(otelcol_receiver_accepted_spans_total${s}[${w}]))`,
    logs:    `sum by (instance) (rate(otelcol_receiver_accepted_log_records_total${s}[${w}]))`,
    metrics: `sum by (instance) (rate(otelcol_receiver_accepted_metric_points_total${s}[${w}]))`,
    errs:    `sum by (instance) (rate(otelpoc_spans_errors_total${s}[${w}])) * 60`,
    errPct:  `100 * (sum by (instance) (rate(otelpoc_spans_errors_total${s}[${w}])) / clamp_min(sum by (instance) (rate(otelpoc_spans_total${s}[${w}])), 0.0001))`,
    p95:     `histogram_quantile(0.95, sum by (le, instance) (rate(traces_spanmetrics_duration_milliseconds_bucket${s}[5m])))`,
    cpu:     `avg by (instance) (rate(system_cpu_time_seconds_total${sel(im, 'state!="idle"')}[${w}])) * 100`,
    mem:     `max by (instance) (otelcol_process_memory_rss_bytes${s})`,
    refused: `sum by (instance) (
      rate(otelcol_receiver_refused_spans_total${s}[${w}])
      + rate(otelcol_receiver_refused_log_records_total${s}[${w}])
      + rate(otelcol_receiver_refused_metric_points_total${s}[${w}])
    ) * 60`,
    uptime:  `max by (instance) (otelcol_process_uptime_seconds_total${s})`,
  };
  const entries = await Promise.all(Object.entries(Q).map(async ([k, q]) => {
    const d = await promInstant(q).catch(() => ({ result: [] }));
    const m = {};
    d.result.forEach(r => { m[r.metric.instance] = +r.value[1]; });
    return [k, m];
  }));
  const M = Object.fromEntries(entries);

  const rows = state.instances
    .filter(i => state.selectedInstances.size === 0 || state.selectedInstances.has(i))
    .sort()
    .map(inst => ({
      instance: inst,
      up:      M.up[inst] ?? 0,
      spans:   M.spans[inst] ?? 0,
      logs:    M.logs[inst] ?? 0,
      metrics: M.metrics[inst] ?? 0,
      errs:    M.errs[inst] ?? 0,
      errPct:  M.errPct[inst] ?? 0,
      p95:     M.p95[inst],
      cpu:     M.cpu[inst] ?? 0,
      mem:     M.mem[inst] ?? 0,
      refused: M.refused[inst] ?? 0,
      uptime:  M.uptime[inst] ?? 0,
    }));

  const maxSpans = Math.max(1, ...rows.map(r => r.spans));
  const tbody = document.querySelector("#instance-table tbody");
  tbody.innerHTML = "";
  rows.forEach(r => {
    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td class="inst">${r.instance}</td>
      <td class="${r.up ? "ok" : "danger"}">${r.up ? "UP" : "DOWN"}</td>
      <td>${fmtShort(r.spans)}<div class="bar"><span style="width:${(r.spans / maxSpans) * 100}%"></span></div></td>
      <td>${fmtShort(r.logs)}</td>
      <td>${fmtShort(r.metrics)}</td>
      <td class="${r.errs > 0 ? "danger" : "muted"}">${fmtShort(r.errs)}</td>
      <td class="${r.errPct > 1 ? "danger" : "muted"}">${fmtPct(r.errPct)}</td>
      <td>${fmtMs(r.p95)}</td>
      <td>${fmtShort(r.cpu)}</td>
      <td>${fmtBytes(r.mem)}</td>
      <td class="${r.refused > 0 ? "danger" : "muted"}">${fmtShort(r.refused)}</td>
      <td class="muted">${fmtDuration(r.uptime)}</td>
    `;
    tbody.appendChild(tr);
  });
}

async function updateHealth() {
  const pill = document.getElementById("health-pill");
  const text = document.getElementById("health-text");
  try {
    const up = await scalar(`count(up{job="collector-app"}==1)`);
    const total = state.instances.length || 5;
    pill.classList.remove("ok", "error");
    if (up === total) { pill.classList.add("ok"); text.textContent = `${up}/${total} online`; }
    else if (up === 0) { pill.classList.add("error"); text.textContent = "nenhum collector acessível"; }
    else text.textContent = `${up}/${total} online`;
  } catch { pill.classList.add("error"); text.textContent = "prom inacessível"; }
}

// ============ Orchestration ============

async function safe(name, fn) {
  try { await fn(); } catch (e) { console.warn("fail " + name, e); }
}

async function refreshAll() {
  // KPIs + table primeiro (leve, preenchem o topo logo)
  await Promise.allSettled([safe("kpis", panelKPIs), safe("table", panelTable), safe("health", updateHealth)]);
  // Charts depois, em lote único mas sem animação → rápido
  await Promise.allSettled([
    safe("throughput", panelThroughput),
    safe("severity",   panelSeverity),
    safe("errors",     panelErrors),
    safe("latency",    panelLatency),
    safe("mem",        panelMem),
  ]);
  document.getElementById("last-updated").textContent = new Date().toLocaleTimeString("pt-BR");
}

async function init() {
  // Espera o Chart.js (carregado com defer, pode já estar pronto).
  if (typeof Chart === "undefined") {
    await new Promise(r => {
      const i = setInterval(() => { if (typeof Chart !== "undefined") { clearInterval(i); r(); } }, 25);
    });
  }
  // Date-adapter built-in do Chart.js v4 só existe se adapter de date for registrado.
  // Como usamos "time" scale, precisamos fallback: se faltar adapter, cai pra linear.
  if (!Chart._adapters || !Chart._adapters._date || !Chart._adapters._date.prototype?.format) {
    // registra adapter mínimo usando Intl
    Chart._adapters = Chart._adapters || {};
    class MiniAdapter {
      constructor() {}
      init() {}
      formats() { return {}; }
      parse(v) { return typeof v === "number" ? v : +new Date(v); }
      format(ts) {
        const d = new Date(ts);
        return d.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
      }
      add(ts, amount, unit) {
        const d = new Date(ts);
        if (unit === "second") d.setSeconds(d.getSeconds() + amount);
        else if (unit === "minute") d.setMinutes(d.getMinutes() + amount);
        else if (unit === "hour") d.setHours(d.getHours() + amount);
        else if (unit === "day") d.setDate(d.getDate() + amount);
        return +d;
      }
      diff(a, b) { return a - b; }
      startOf(ts) { return ts; }
      endOf(ts) { return ts; }
    }
    Chart._adapters._date = MiniAdapter;
  }
  bindFilters();
  try { await discoverInstances(); } catch (e) { console.error(e); }
  await refreshAll();
  scheduleRefresh();
  state.booted = true;
}

if (document.readyState === "loading") window.addEventListener("DOMContentLoaded", init);
else init();
