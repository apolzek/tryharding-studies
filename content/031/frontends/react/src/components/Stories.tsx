import { useEffect, useMemo, useState } from 'react';
import { trace } from '@opentelemetry/api';
import { generateStories } from '../data/stories';
import { STACK_NAME } from '../telemetry';
import StoryViewer from './StoryViewer';

export default function Stories() {
  const stories = useMemo(() => generateStories(12), []);
  const [openIndex, setOpenIndex] = useState<number | null>(null);
  const [seen, setSeen] = useState<Set<string>>(() => new Set());

  useEffect(() => {
    const tracer = trace.getTracer('stories');
    const span = tracer.startSpan('stories.view_strip', {
      attributes: { 'frontend.stack': STACK_NAME },
    });
    span.end();
  }, []);

  const openAt = (i: number) => {
    setOpenIndex(i);
    setSeen((s) => {
      const next = new Set(s);
      next.add(stories[i].id);
      return next;
    });
  };

  return (
    <div className="stories">
      <div className="stories-strip">
        {stories.map((story, i) => (
          <div
            key={story.id}
            className="story-item"
            role="button"
            tabIndex={0}
            onClick={() => openAt(i)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') openAt(i);
            }}
          >
            <div className={`story-ring${seen.has(story.id) ? ' seen' : ''}`}>
              <div
                className="story-avatar"
                style={{ color: story.avatarColor }}
              >
                {story.author[0].toUpperCase()}
              </div>
            </div>
            <span className="story-author">@{story.author}</span>
          </div>
        ))}
      </div>
      {openIndex != null && (
        <StoryViewer
          stories={stories}
          startIndex={openIndex}
          onClose={() => setOpenIndex(null)}
        />
      )}
    </div>
  );
}
