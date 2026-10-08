<template>
  <div class="emoji-reactions" aria-hidden="true">
    <!-- Floating reactions over video stream -->
    <TransitionGroup name="float-up" tag="div" class="reactions-container">
      <div
        v-for="r in reactions"
        :key="r.id"
        class="floating-emoji"
        :style="{ left: r.x + '%' }"
      >
        {{ r.emoji }}
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
defineProps({
  reactions: { type: Array, default: () => [] },
})
</script>

<style scoped>
.emoji-reactions {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
  z-index: 15;
  border-radius: var(--radius-lg);
}

.reactions-container {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.floating-emoji {
  position: absolute;
  bottom: 20px;
  font-size: 2.2rem;
  animation: float-up 2.8s ease-out forwards;
  pointer-events: none;
  filter: drop-shadow(0 2px 6px rgba(0, 0, 0, 0.6));
  user-select: none;
}

@keyframes float-up {
  0% {
    opacity: 1;
    transform: translateY(0) scale(0.9);
  }
  40% {
    opacity: 1;
    transform: translateY(-80px) scale(1.15);
  }
  80% {
    opacity: 0.8;
    transform: translateY(-180px) scale(1);
  }
  100% {
    opacity: 0;
    transform: translateY(-240px) scale(0.7);
  }
}

.float-up-enter-active {
  animation: float-up 2.8s ease-out forwards;
}
.float-up-leave-active {
  transition: opacity 0.2s;
}
.float-up-leave-to {
  opacity: 0;
}
</style>
