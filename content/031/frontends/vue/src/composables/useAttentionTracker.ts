import { onMounted, onBeforeUnmount, type Ref } from 'vue';
import { trace } from '@opentelemetry/api';
import { getMeters, emitLog, STACK_NAME, getScrollDepthMax } from '../telemetry';

type TrackedPost = { id: string; author: string };

type Session = {
  span: any;
  startTs: number;
  activeMs: number;
  lastTickTs: number;
  lastInputTs: number;
  visibilityRatio: number;
};

export function useAttentionTracker(elementRef: Ref<HTMLElement | null>, post: TrackedPost) {
  let session: Session | null = null;
  let observer: IntersectionObserver | null = null;
  let interval: any;

  const tracer = trace.getTracer('attention-tracker');

  const tick = () => {
    if (!session) return;
    const now = performance.now();
    const idle = Date.now() - session.lastInputTs > 10000;
    const active = !document.hidden && !idle;
    if (active) session.activeMs += now - session.lastTickTs;
    session.lastTickTs = now;
  };

  const startSession = (ratio: number) => {
    if (session) return;
    const span = tracer.startSpan('post.view', {
      attributes: {
        'post.id': post.id,
        'post.author': post.author,
        'visibility.ratio': ratio,
        'frontend.stack': STACK_NAME,
      },
    });
    session = {
      span,
      startTs: performance.now(),
      activeMs: 0,
      lastTickTs: performance.now(),
      lastInputTs: Date.now(),
      visibilityRatio: ratio,
    };
  };

  const endSession = () => {
    if (!session) return;
    tick();
    const dwell = Math.round(session.activeMs);
    try {
      getMeters().postDwellHistogram?.record(dwell, {
        'post.id': post.id,
        'frontend.stack': STACK_NAME,
      });
    } catch {}
    const traceId = session.span.spanContext?.().traceId || '';
    emitLog({
      event: 'post.view_end',
      post_id: post.id,
      dwell_ms: dwell,
      scroll_depth_pct: getScrollDepthMax(),
      visibility_ratio: session.visibilityRatio,
      trace_id: traceId,
    });
    session.span.setAttribute('dwell.ms', dwell);
    session.span.end();
    session = null;
  };

  const onInput = () => {
    if (session) session.lastInputTs = Date.now();
  };

  onMounted(() => {
    const el = elementRef.value;
    if (!el) return;
    observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting && entry.intersectionRatio >= 0.5) {
            startSession(entry.intersectionRatio);
          } else {
            endSession();
          }
        }
      },
      {
        threshold: [0, 0.25, 0.5, 0.75, 1],
        // @ts-ignore IO v2
        trackVisibility: true,
        // @ts-ignore
        delay: 100,
      }
    );
    observer.observe(el);
    window.addEventListener('pointermove', onInput, { passive: true });
    window.addEventListener('keydown', onInput);
    window.addEventListener('scroll', onInput, { passive: true });
    document.addEventListener('visibilitychange', tick);
    window.addEventListener('beforeunload', endSession);
    interval = setInterval(tick, 1000);
  });

  onBeforeUnmount(() => {
    clearInterval(interval);
    observer?.disconnect();
    window.removeEventListener('pointermove', onInput);
    window.removeEventListener('keydown', onInput);
    window.removeEventListener('scroll', onInput);
    document.removeEventListener('visibilitychange', tick);
    window.removeEventListener('beforeunload', endSession);
    endSession();
  });
}
