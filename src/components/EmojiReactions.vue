<template>
  <div class="emoji-reactions">
    <!-- Floating reactions -->
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

    <!-- Emoji picker trigger -->
    <div class="emoji-picker-wrapper">
      <button
        class="emoji-trigger"
        :class="{ 'emoji-trigger--open': showPicker }"
        @click="showPicker = !showPicker"
        data-tooltip="React"
        aria-label="Send emoji reaction"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
          <line x1="9" y1="9" x2="9.01" y2="9"/>
          <line x1="15" y1="9" x2="15.01" y2="9"/>
        </svg>
      </button>

      <!-- Picker panel -->
      <Transition name="fade">
        <div v-if="showPicker" class="emoji-picker glass">
          <button
            v-for="emoji in EMOJIS"
            :key="emoji"
            class="emoji-btn"
            @click="selectEmoji(emoji)"
            :aria-label="'React with ' + emoji"
          >
            {{ emoji }}
          </button>
        </div>
      </Transition>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const EMOJIS = ['😂', '🔥', '👏', '❤️', '😮', '🎉']

defineProps({
  reactions: { type: Array, default: () => [] },
})

const emit = defineEmits(['react'])

const showPicker = ref(false)

function selectEmoji(emoji) {
  emit('react', emoji)
  showPicker.value = false
}
</script>

<style scoped>
.emoji-reactions {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 5;
}

/* Floating reactions container */
.reactions-container {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.floating-emoji {
  position: absolute;
  bottom: 60px;
  font-size: 2rem;
  animation: float-up 3s ease-out forwards;
  pointer-events: none;
  filter: drop-shadow(0 2px 4px rgba(0,0,0,0.5));
}

@keyframes float-up {
  0% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
  50% {
    opacity: 1;
    transform: translateY(-120px) scale(1.15);
  }
  100% {
    opacity: 0;
    transform: translateY(-250px) scale(0.8);
  }
}

/* Transition group */
.float-up-enter-active {
  animation: float-up 3s ease-out forwards;
}
.float-up-leave-active {
  transition: opacity 0.3s;
}
.float-up-leave-to {
  opacity: 0;
}

/* Emoji trigger button */
.emoji-picker-wrapper {
  position: absolute;
  bottom: 70px;
  right: var(--space-3);
  pointer-events: auto;
  z-index: 10;
}

.emoji-trigger {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.8);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
}
.emoji-trigger:hover {
  background: rgba(0, 0, 0, 0.8);
  color: white;
  transform: scale(1.1);
}
.emoji-trigger--open {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

/* Emoji picker panel */
.emoji-picker {
  position: absolute;
  bottom: 48px;
  right: 0;
  display: flex;
  gap: var(--space-1);
  padding: var(--space-2);
  border-radius: var(--radius-lg);
  background: rgba(10, 11, 16, 0.9) !important;
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
}

.emoji-btn {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: none;
  font-size: 1.25rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}
.emoji-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  transform: scale(1.2);
}

@media (max-width: 600px) {
  .emoji-trigger {
    width: 36px;
    height: 36px;
  }
  .emoji-trigger svg {
    width: 16px;
    height: 16px;
  }
  .emoji-btn {
    width: 32px;
    height: 32px;
    font-size: 1.1rem;
  }
}
</style>
