import { trace } from '@opentelemetry/api';
import { generatePosts } from './data.js';
import { trackPostElement } from './attention-tracker.js';
import { getMeters, emitLog, STACK_NAME } from './telemetry.js';

let offset = 0;
let loading = false;

function interact(post, type, btn, likesNode, state) {
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
    state.liked = !state.liked;
    state.likes += state.liked ? 1 : -1;
    btn.classList.toggle('active', state.liked);
    likesNode.textContent = String(state.likes);
  }
}

function renderPost(post) {
  const article = document.createElement('article');
  article.className = 'card';
  article.dataset.postId = post.id;

  const state = { likes: post.likes, liked: false };

  const header = document.createElement('div');
  header.className = 'card-header';
  const avatar = document.createElement('div');
  avatar.className = 'avatar';
  avatar.style.background = post.avatarColor;
  avatar.textContent = post.author[0].toUpperCase();
  const meta = document.createElement('div');
  meta.className = 'meta';
  const username = document.createElement('div');
  username.className = 'username';
  username.textContent = '@' + post.author;
  const ts = document.createElement('div');
  ts.className = 'timestamp';
  ts.textContent = new Date(post.timestamp).toLocaleString();
  meta.append(username, ts);
  header.append(avatar, meta);

  const media = document.createElement('div');
  media.className = 'media';
  media.style.background = post.gradient;
  const mediaTitle = document.createElement('span');
  mediaTitle.className = 'media-title';
  mediaTitle.textContent = post.title;
  media.append(mediaTitle);

  const actions = document.createElement('div');
  actions.className = 'actions';

  const likeBtn = document.createElement('button');
  const likeIcon = document.createElement('span');
  likeIcon.className = 'icon';
  likeIcon.textContent = '♥';
  const likesCount = document.createElement('span');
  likesCount.textContent = String(state.likes);
  likeBtn.append(likeIcon, likesCount);
  likeBtn.addEventListener('click', () => interact(post, 'like', likeBtn, likesCount, state));

  const commentBtn = document.createElement('button');
  const commentIcon = document.createElement('span');
  commentIcon.className = 'icon';
  commentIcon.textContent = '💬';
  const commentCount = document.createElement('span');
  commentCount.textContent = String(post.comments);
  commentBtn.append(commentIcon, commentCount);
  commentBtn.addEventListener('click', () => interact(post, 'comment', commentBtn, commentCount, state));

  const shareBtn = document.createElement('button');
  const shareIcon = document.createElement('span');
  shareIcon.className = 'icon';
  shareIcon.textContent = '↗';
  const shareLabel = document.createElement('span');
  shareLabel.textContent = 'share';
  shareBtn.append(shareIcon, shareLabel);
  shareBtn.addEventListener('click', () => interact(post, 'share', shareBtn, shareLabel, state));

  actions.append(likeBtn, commentBtn, shareBtn);

  const caption = document.createElement('p');
  caption.className = 'caption';
  caption.textContent = post.caption;

  article.append(header, media, actions, caption);
  trackPostElement(article, post);
  return article;
}

function appendBatch(container, n) {
  const batch = generatePosts(offset, n);
  offset += n;
  const frag = document.createDocumentFragment();
  for (const p of batch) frag.appendChild(renderPost(p));
  container.appendChild(frag);
}

export function initFeed() {
  const container = document.getElementById('feed');
  if (!container) return;
  appendBatch(container, 10);

  window.addEventListener(
    'scroll',
    () => {
      if (loading) return;
      const h = document.documentElement;
      if (h.scrollTop + h.clientHeight >= h.scrollHeight - 400) {
        loading = true;
        appendBatch(container, 10);
        setTimeout(() => (loading = false), 150);
      }
    },
    { passive: true }
  );
}
