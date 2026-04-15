import { trace } from '@opentelemetry/api';
import { getMeters, emitLog, STACK_NAME, getScrollDepthMax } from './telemetry.js';

const sessions = new WeakMap();
let observer = null;
let globalListenersInstalled = false;
let lastInputTs = Date.now();

function installGlobals() {
  if (globalListenersInstalled) return;
  globalListenersInstalled = true;
  const onInput = () => {
    lastInputTs = Date.now();
  };
  window.addEventListener('pointermove', onInput, { passive: true });
  window.addEventListener('keydown', onInput);
  window.addEventListener('scroll', onInput, { passive: true });
  document.addEventListener('visibilitychange', tickAll);
  window.addEventListener('beforeunload', endAll);
  setInterval(tickAll, 1000);
}

const liveSessions = new Set();

function tick(session) {
  const now = performance.now();
  const idle = Date.now() - lastInputTs > 10000;
  const active = !document.hidden && !idle;
  if (active) session.activeMs += now - session.lastTickTs;
  session.lastTickTs = now;
}

function tickAll() {
  liveSessions.forEach(tick);
}

function endSession(el) {
  const session = sessions.get(el);
  if (!session) return;
  tick(session);
  const dwell = Math.round(session.activeMs);
  try {
    getMeters().postDwellHistogram?.record(dwell, {
      'post.id': session.post.id,
      'frontend.stack': STACK_NAME,
    });
  } catch {}
  const traceId = session.span.spanContext?.().traceId || '';
  emitLog({
    event: 'post.view_end',
    post_id: session.post.id,
    dwell_ms: dwell,
    scroll_depth_pct: getScrollDepthMax(),
    visibility_ratio: session.visibilityRatio,
    trace_id: traceId,
  });
  session.span.setAttribute('dwell.ms', dwell);
  session.span.end();
  sessions.delete(el);
  liveSessions.delete(session);
}

function endAll() {
  const arr = [];
  liveSessions.forEach((s) => arr.push(s));
  arr.forEach((s) => endSession(s.el));
}

function startSession(el, post, ratio) {
  if (sessions.has(el)) return;
  const tracer = trace.getTracer('attention-tracker');
  const span = tracer.startSpan('post.view', {
    attributes: {
      'post.id': post.id,
      'post.author': post.author,
      'visibility.ratio': ratio,
      'frontend.stack': STACK_NAME,
    },
  });
  const session = {
    el,
    post,
    span,
    startTs: performance.now(),
    activeMs: 0,
    lastTickTs: performance.now(),
    visibilityRatio: ratio,
  };
  sessions.set(el, session);
  liveSessions.add(session);
}

function ensureObserver() {
  if (observer) return observer;
  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        const post = entry.target.__post;
        if (!post) continue;
        if (entry.isIntersecting && entry.intersectionRatio >= 0.5) {
          startSession(entry.target, post, entry.intersectionRatio);
        } else {
          endSession(entry.target);
        }
      }
    },
    {
      threshold: [0, 0.25, 0.5, 0.75, 1],
      trackVisibility: true,
      delay: 100,
    }
  );
  return observer;
}

export function trackPostElement(el, post) {
  installGlobals();
  el.__post = post;
  ensureObserver().observe(el);
}

export function untrackPostElement(el) {
  if (observer) observer.unobserve(el);
  endSession(el);
}
