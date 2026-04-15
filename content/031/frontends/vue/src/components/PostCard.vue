<script setup lang="ts">
import { ref } from 'vue';
import { trace } from '@opentelemetry/api';
import type { Post } from '../data/posts';
import { useAttentionTracker } from '../composables/useAttentionTracker';
import { getMeters, emitLog, STACK_NAME } from '../telemetry';

const props = defineProps<{ post: Post }>();

const cardRef = ref<HTMLElement | null>(null);
const likes = ref(props.post.likes);
const liked = ref(false);

useAttentionTracker(cardRef, { id: props.post.id, author: props.post.author });

function interact(type: 'like' | 'comment' | 'share') {
  const tracer = trace.getTracer('interactions');
  const span = tracer.startSpan('post.interact', {
    attributes: {
      'post.id': props.post.id,
      'interaction.type': type,
      'frontend.stack': STACK_NAME,
    },
  });
  try {
    getMeters().postInteractionsCounter?.add(1, {
      type,
      'post.id': props.post.id,
      'frontend.stack': STACK_NAME,
    });
  } catch {}
  emitLog({ event: 'post.interact', post_id: props.post.id, type });
  span.end();
  if (type === 'like') {
    liked.value = !liked.value;
    likes.value += liked.value ? 1 : -1;
  }
}
</script>

<template>
  <article class="card" ref="cardRef" :data-post-id="post.id">
    <div class="card-header">
      <div class="avatar" :style="{ background: post.avatarColor }">
        {{ post.author[0].toUpperCase() }}
      </div>
      <div class="meta">
        <div class="username">@{{ post.author }}</div>
        <div class="timestamp">{{ new Date(post.timestamp).toLocaleString() }}</div>
      </div>
    </div>
    <div class="media" :style="{ background: post.gradient }">
      <span class="media-title">{{ post.title }}</span>
    </div>
    <div class="actions">
      <button :class="{ active: liked }" @click="interact('like')">
        <span class="icon">♥</span><span>{{ likes }}</span>
      </button>
      <button @click="interact('comment')">
        <span class="icon">💬</span><span>{{ post.comments }}</span>
      </button>
      <button @click="interact('share')">
        <span class="icon">↗</span><span>share</span>
      </button>
    </div>
    <p class="caption">{{ post.caption }}</p>
  </article>
</template>
