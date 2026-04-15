import { useEffect, useRef } from 'react';
import { trace } from '@opentelemetry/api';
import { getMeters, emitLog, STACK_NAME } from '../telemetry';
import type { Story } from '../data/stories';

type Session = {
  story: Story;
  span: any;
  startTs: number;
  activeMs: number;
  lastTickTs: number;
  skipped: 'forward' | 'back' | null;
};

function completionBucket(ratio: number): string {
  if (ratio >= 0.9) return '1.0';
  if (ratio >= 0.75) return '0.75';
  if (ratio >= 0.5) return '0.5';
  if (ratio >= 0.25) return '0.25';
  return '0.0';
}

export function useStoryAttention(
  activeStory: Story | null,
  skipDirectionRef: React.MutableRefObject<'forward' | 'back' | null>
) {
  const sessionRef = useRef<Session | null>(null);

  useEffect(() => {
    const tracer = trace.getTracer('story-attention');

    const tick = () => {
      const s = sessionRef.current;
      if (!s) return;
      const now = performance.now();
      if (!document.hidden) s.activeMs += now - s.lastTickTs;
      s.lastTickTs = now;
    };

    const endSession = (reasonSkipped: 'forward' | 'back' | null = null) => {
      const s = sessionRef.current;
      if (!s) return;
      tick();
      const activeMs = Math.round(s.activeMs);
      const durationMs = s.story.durationMs;
      const ratio = durationMs > 0 ? activeMs / durationMs : 0;
      const completed = activeMs >= 0.9 * durationMs;
      const bucket = completionBucket(ratio);
      const skipped = reasonSkipped ?? s.skipped;
      const {
        storyDwellHistogram,
        storyViewsCounter,
        storySkipsCounter,
      } = getMeters();
      try {
        storyDwellHistogram?.record(activeMs, {
          'story.id': s.story.id,
          'frontend.stack': STACK_NAME,
          'completion.ratio': bucket,
        });
      } catch {}
      try {
        storyViewsCounter?.add(1, {
          story_id: s.story.id,
          completed: completed ? 'true' : 'false',
          'frontend.stack': STACK_NAME,
        });
      } catch {}
      if (skipped) {
        try {
          storySkipsCounter?.add(1, {
            direction: skipped,
            'frontend.stack': STACK_NAME,
          });
        } catch {}
      }
      const traceId = s.span.spanContext?.().traceId || '';
      emitLog({
        event: 'story.view_end',
        story_id: s.story.id,
        active_ms: activeMs,
        completion_ratio: Number(ratio.toFixed(3)),
        skipped: skipped || '',
        trace_id: traceId,
      });
      s.span.setAttribute('active.ms', activeMs);
      s.span.setAttribute('completion.ratio', bucket);
      s.span.end();
      sessionRef.current = null;
    };

    const startSession = (story: Story) => {
      const span = tracer.startSpan('story.view', {
        attributes: {
          'story.id': story.id,
          'story.author': story.author,
          'story.duration_ms': story.durationMs,
          'frontend.stack': STACK_NAME,
        },
      });
      sessionRef.current = {
        story,
        span,
        startTs: performance.now(),
        activeMs: 0,
        lastTickTs: performance.now(),
        skipped: null,
      };
    };

    // End any previous session with the skip reason captured at switch time.
    const pendingSkip = skipDirectionRef.current;
    skipDirectionRef.current = null;
    endSession(pendingSkip);

    if (activeStory) startSession(activeStory);

    const onVisibility = () => tick();
    const onUnload = () => endSession();
    const interval = setInterval(tick, 500);
    document.addEventListener('visibilitychange', onVisibility);
    window.addEventListener('beforeunload', onUnload);

    return () => {
      clearInterval(interval);
      document.removeEventListener('visibilitychange', onVisibility);
      window.removeEventListener('beforeunload', onUnload);
      // Do NOT end session here — end happens at next effect run or unmount fallback below.
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeStory?.id]);

  // On unmount, flush any in-flight session.
  useEffect(() => {
    return () => {
      const s = sessionRef.current;
      if (!s) return;
      const activeMs = Math.round(s.activeMs);
      const durationMs = s.story.durationMs;
      const ratio = durationMs > 0 ? activeMs / durationMs : 0;
      const completed = activeMs >= 0.9 * durationMs;
      const bucket = completionBucket(ratio);
      const {
        storyDwellHistogram,
        storyViewsCounter,
      } = getMeters();
      try {
        storyDwellHistogram?.record(activeMs, {
          'story.id': s.story.id,
          'frontend.stack': STACK_NAME,
          'completion.ratio': bucket,
        });
      } catch {}
      try {
        storyViewsCounter?.add(1, {
          story_id: s.story.id,
          completed: completed ? 'true' : 'false',
          'frontend.stack': STACK_NAME,
        });
      } catch {}
      const traceId = s.span.spanContext?.().traceId || '';
      emitLog({
        event: 'story.view_end',
        story_id: s.story.id,
        active_ms: activeMs,
        completion_ratio: Number(ratio.toFixed(3)),
        skipped: '',
        trace_id: traceId,
      });
      s.span.setAttribute('active.ms', activeMs);
      s.span.setAttribute('completion.ratio', bucket);
      s.span.end();
      sessionRef.current = null;
    };
  }, []);
}
