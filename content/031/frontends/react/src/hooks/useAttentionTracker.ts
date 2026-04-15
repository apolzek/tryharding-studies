import { useEffect, useRef } from 'react';
import { trace, context } from '@opentelemetry/api';
import { getMeters, emitLog, STACK_NAME, getScrollDepthMax } from '../telemetry';

type TrackedPost = { id: string; author: string };

type Session = {
  span: any;
  startTs: number;
  activeMs: number;
  lastTickTs: number;
  lastInputTs: number;
  visibilityRatio: number;
  paused: boolean;
};

export function useAttentionTracker(
  elementRef: React.RefObject<HTMLElement>,
  post: TrackedPost
) {
  const sessionRef = useRef<Session | null>(null);

  useEffect(() => {
    const el = elementRef.current;
    if (!el) return;

    const tracer = trace.getTracer('attention-tracker');

    const startSession = (ratio: number) => {
      if (sessionRef.current) return;
      const span = tracer.startSpan('post.view', {
        attributes: {
          'post.id': post.id,
          'post.author': post.author,
          'visibility.ratio': ratio,
          'frontend.stack': STACK_NAME,
        },
      });
      sessionRef.current = {
        span,
        startTs: performance.now(),
        activeMs: 0,
        lastTickTs: performance.now(),
        lastInputTs: Date.now(),
        visibilityRatio: ratio,
        paused: document.hidden,
      };
    };

    const tick = () => {
      const s = sessionRef.current;
      if (!s) return;
      const now = performance.now();
      const idle = Date.now() - s.lastInputTs > 10000;
      const active = !document.hidden && !idle;
      if (active) s.activeMs += now - s.lastTickTs;
      s.lastTickTs = now;
    };

    const endSession = () => {
      const s = sessionRef.current;
      if (!s) return;
      tick();
      const dwell = Math.round(s.activeMs);
      const { postDwellHistogram } = getMeters();
      try {
        postDwellHistogram?.record(dwell, {
          'post.id': post.id,
          'frontend.stack': STACK_NAME,
        });
      } catch {}
      const ctx = trace.setSpan(context.active(), s.span);
      const traceId = s.span.spanContext?.().traceId || '';
      emitLog({
        event: 'post.view_end',
        post_id: post.id,
        dwell_ms: dwell,
        scroll_depth_pct: getScrollDepthMax(),
        visibility_ratio: s.visibilityRatio,
        trace_id: traceId,
      });
      s.span.setAttribute('dwell.ms', dwell);
      s.span.end();
      sessionRef.current = null;
      void ctx;
    };

    const observer = new IntersectionObserver(
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
        // @ts-ignore - IO v2
        trackVisibility: true,
        // @ts-ignore
        delay: 100,
      }
    );
    observer.observe(el);

    const onInput = () => {
      if (sessionRef.current) sessionRef.current.lastInputTs = Date.now();
    };
    const onVisibility = () => tick();
    window.addEventListener('pointermove', onInput, { passive: true });
    window.addEventListener('keydown', onInput);
    window.addEventListener('scroll', onInput, { passive: true });
    document.addEventListener('visibilitychange', onVisibility);
    window.addEventListener('beforeunload', endSession);

    const interval = setInterval(tick, 1000);

    return () => {
      clearInterval(interval);
      observer.disconnect();
      window.removeEventListener('pointermove', onInput);
      window.removeEventListener('keydown', onInput);
      window.removeEventListener('scroll', onInput);
      document.removeEventListener('visibilitychange', onVisibility);
      window.removeEventListener('beforeunload', endSession);
      endSession();
    };
  }, [elementRef, post.id, post.author]);
}
