<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue';
import PostCard from './PostCard.vue';
import { generatePosts, type Post } from '../data/posts';

const posts = ref<Post[]>(generatePosts(0, 10));
let offset = 10;
let loading = false;

function onScroll() {
  if (loading) return;
  const h = document.documentElement;
  if (h.scrollTop + h.clientHeight >= h.scrollHeight - 400) {
    loading = true;
    posts.value = [...posts.value, ...generatePosts(offset, 10)];
    offset += 10;
    setTimeout(() => (loading = false), 150);
  }
}

onMounted(() => window.addEventListener('scroll', onScroll, { passive: true }));
onBeforeUnmount(() => window.removeEventListener('scroll', onScroll));
</script>

<template>
  <div class="feed">
    <PostCard v-for="p in posts" :key="p.id" :post="p" />
  </div>
</template>
