<template>
  <ConstellationBackground />
  <RouterView :key="$route.fullPath" />
</template>

<script setup>
import { RouterView } from 'vue-router'
import { onMounted } from 'vue'
import ConstellationBackground from './components/ConstellationBackground.vue'

// Initialize theme from localStorage
onMounted(() => {
  const theme = localStorage.getItem('wp_theme') || 'dark'
  document.documentElement.setAttribute('data-theme', theme)
})
</script>
