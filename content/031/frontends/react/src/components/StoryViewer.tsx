import { useEffect, useRef, useState } from 'react';
import type { Story } from '../data/stories';
import { useStoryAttention } from '../hooks/useStoryAttention';

type Props = {
  stories: Story[];
  startIndex: number;
  onClose: () => void;
};

export default function StoryViewer({ stories, startIndex, onClose }: Props) {
  const [index, setIndex] = useState(startIndex);
  const [progressKey, setProgressKey] = useState(0);
  const skipDirectionRef = useRef<'forward' | 'back' | null>(null);
  const activeStory = stories[index] ?? null;

  useStoryAttention(activeStory, skipDirectionRef);

  // Lock body scroll while open.
  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  const advance = (dir: 'forward' | 'back') => {
    skipDirectionRef.current = dir;
    setIndex((i) => {
      if (dir === 'forward') {
        if (i >= stories.length - 1) {
          // Auto-close after last
          queueMicrotask(onClose);
          return i;
        }
        return i + 1;
      }
      return Math.max(0, i - 1);
    });
    setProgressKey((k) => k + 1);
  };

  // Auto-advance timer
  useEffect(() => {
    if (!activeStory) return;
    let rafId: number | null = null;
    const started = performance.now();
    const tick = () => {
      if (document.hidden) {
        rafId = requestAnimationFrame(tick);
        return;
      }
      const elapsed = performance.now() - started;
      if (elapsed >= activeStory.durationMs) {
        // Auto-advance: NOT a skip.
        skipDirectionRef.current = null;
        setIndex((i) => {
          if (i >= stories.length - 1) {
            queueMicrotask(onClose);
            return i;
          }
          return i + 1;
        });
        setProgressKey((k) => k + 1);
        return;
      }
      rafId = requestAnimationFrame(tick);
    };
    rafId = requestAnimationFrame(tick);
    return () => {
      if (rafId != null) cancelAnimationFrame(rafId);
    };
  }, [index, activeStory, stories.length, onClose]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
      else if (e.key === 'ArrowRight') advance('forward');
      else if (e.key === 'ArrowLeft') advance('back');
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [onClose]);

  if (!activeStory) return null;

  return (
    <div className="story-viewer" role="dialog" aria-modal="true">
      <div
        className="story-viewer-media"
        style={{ background: activeStory.gradient }}
      />
      <div className="story-progress">
        <div
          key={progressKey}
          className="story-progress-bar"
          style={{
            animation: `story-progress-fill ${activeStory.durationMs}ms linear forwards`,
          }}
        />
      </div>
      <div className="story-viewer-header">
        <div
          className="avatar"
          style={{ background: activeStory.avatarColor, width: 32, height: 32, fontSize: 14 }}
        >
          {activeStory.author[0].toUpperCase()}
        </div>
        <span style={{ fontWeight: 600 }}>@{activeStory.author}</span>
      </div>
      <button
        type="button"
        className="story-close"
        aria-label="Close"
        onClick={onClose}
      >
        ×
      </button>
      <div className="story-tap-left" onClick={() => advance('back')} />
      <div className="story-tap-right" onClick={() => advance('forward')} />
      <div className="story-title-big">{activeStory.title}</div>
    </div>
  );
}
