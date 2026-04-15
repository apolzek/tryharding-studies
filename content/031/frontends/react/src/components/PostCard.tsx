import { useRef, useState } from 'react';
import { trace } from '@opentelemetry/api';
import type { Post } from '../data/posts';
import { useAttentionTracker } from '../hooks/useAttentionTracker';
import { getMeters, emitLog, STACK_NAME } from '../telemetry';

export default function PostCard({ post }: { post: Post }) {
  const ref = useRef<HTMLElement>(null);
  useAttentionTracker(ref, post);

  const [likes, setLikes] = useState(post.likes);
  const [liked, setLiked] = useState(false);

  const interact = (type: 'like' | 'comment' | 'share') => {
    const tracer = trace.getTracer('interactions');
    const span = tracer.startSpan('post.interact', {
      attributes: {
        'post.id': post.id,
        'interaction.type': type,
        'frontend.stack': STACK_NAME,
      },
    });
    try {
      getMeters().postInteractionsCounter?.add(1, {
        type,
        'post.id': post.id,
        'frontend.stack': STACK_NAME,
      });
    } catch {}
    emitLog({ event: 'post.interact', post_id: post.id, type });
    span.end();
    if (type === 'like') {
      setLiked((v) => !v);
      setLikes((n) => (liked ? n - 1 : n + 1));
    }
  };

  return (
    <article className="card" ref={ref} data-post-id={post.id}>
      <div className="card-header">
        <div className="avatar" style={{ background: post.avatarColor }}>
          {post.author[0].toUpperCase()}
        </div>
        <div className="meta">
          <div className="username">@{post.author}</div>
          <div className="timestamp">{new Date(post.timestamp).toLocaleString()}</div>
        </div>
      </div>
      <div className="media" style={{ background: post.gradient }}>
        <span className="media-title">{post.title}</span>
      </div>
      <div className="actions">
        <button className={liked ? 'active' : ''} onClick={() => interact('like')}>
          <span className="icon">♥</span>
          <span>{likes}</span>
        </button>
        <button onClick={() => interact('comment')}>
          <span className="icon">💬</span>
          <span>{post.comments}</span>
        </button>
        <button onClick={() => interact('share')}>
          <span className="icon">↗</span>
          <span>share</span>
        </button>
      </div>
      <p className="caption">{post.caption}</p>
    </article>
  );
}
