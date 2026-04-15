import { useEffect, useRef, useState } from 'react';
import PostCard from './PostCard';
import { generatePosts, type Post } from '../data/posts';

export default function Feed() {
  const [posts, setPosts] = useState<Post[]>(() => generatePosts(0, 10));
  const offsetRef = useRef(10);
  const loadingRef = useRef(false);

  useEffect(() => {
    const onScroll = () => {
      if (loadingRef.current) return;
      const h = document.documentElement;
      if (h.scrollTop + h.clientHeight >= h.scrollHeight - 400) {
        loadingRef.current = true;
        const next = generatePosts(offsetRef.current, 10);
        offsetRef.current += 10;
        setPosts((p) => [...p, ...next]);
        setTimeout(() => {
          loadingRef.current = false;
        }, 150);
      }
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  return (
    <div className="feed">
      {posts.map((p) => (
        <PostCard key={p.id} post={p} />
      ))}
    </div>
  );
}
