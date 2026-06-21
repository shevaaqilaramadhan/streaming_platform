<template>
  <!-- Backdrop -->
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="modelValue"
        class="modal-backdrop"
        role="dialog"
        aria-modal="true"
        aria-labelledby="nickname-modal-title"
        @keydown.esc="$emit('close')"
      >
        <div class="modal-card glass-strong animate-fade-in-scale">
          <!-- Icon -->
          <div class="modal-icon">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="url(#icon-grad)" stroke-width="1.5">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
              <defs>
                <linearGradient id="icon-grad" x1="0" y1="0" x2="24" y2="24" gradientUnits="userSpaceOnUse">
                  <stop offset="0%" stop-color="hsl(195,100%,45%)"/>
                  <stop offset="100%" stop-color="hsl(215,90%,50%)"/>
                </linearGradient>
              </defs>
            </svg>
          </div>

          <!-- Title -->
          <h2 id="nickname-modal-title" class="modal-title">Join the Watch Party</h2>
          <p class="modal-subtitle">
            Enter your nickname so others know who you are.
          </p>

          <!-- Room info chip -->
          <div class="room-chip">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2"/>
              <path d="M3 9h18M9 21V9"/>
            </svg>
            Room: <strong>{{ roomId }}</strong>
          </div>

          <!-- Input -->
          <form @submit.prevent="handleSubmit" class="modal-form">
            <label for="nickname-input" class="sr-only">Your nickname</label>
            <input
              id="nickname-input"
              ref="inputRef"
              v-model="localNickname"
              type="text"
              class="input"
              placeholder="e.g. MovieFan42"
              maxlength="24"
              autocomplete="off"
              spellcheck="false"
            />
            <div class="char-count" :class="{ 'near-limit': localNickname.length > 20 }">
              {{ localNickname.length }}/24
            </div>

            <button
              id="join-room-btn"
              type="submit"
              class="btn btn-primary btn-lg"
              :disabled="!localNickname.trim() || localNickname.trim().length < 2"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/>
                <polyline points="10 17 15 12 10 7"/>
                <line x1="15" y1="12" x2="3" y2="12"/>
              </svg>
              Join Room
            </button>
          </form>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  roomId:     { type: String, required: true },
})

const emit = defineEmits(['update:modelValue', 'join', 'close'])

const localNickname = ref('')
const inputRef = ref(null)

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) nextTick(() => inputRef.value?.focus())
  },
  { immediate: true }
)

function handleSubmit() {
  const name = localNickname.value.trim()
  if (!name || name.length < 2) return
  emit('join', name)
  emit('update:modelValue', false)
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(7, 7, 15, 0.75);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: var(--z-modal);
  padding: var(--space-4);
}

.modal-card {
  width: 100%;
  max-width: 420px;
  border-radius: var(--radius-xl);
  padding: var(--space-10) var(--space-8);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  box-shadow: 0 32px 80px rgba(0,0,0,0.6), var(--shadow-glow-sm);
}

.modal-icon {
  width: 64px;
  height: 64px;
  border-radius: var(--radius-lg);
  background: var(--grad-brand-subtle);
  border: 1px solid hsla(330, 100%, 55%, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: var(--space-2);
}

.modal-title {
  font-size: 1.5rem;
  text-align: center;
}

.modal-subtitle {
  font-size: 0.9375rem;
  text-align: center;
  color: var(--text-secondary);
  margin-top: -var(--space-2);
}

.room-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: rgba(255,255,255,0.05);
  border: 1px solid var(--color-glass-border);
  font-size: 0.8125rem;
  color: var(--text-secondary);
}
.room-chip strong {
  color: var(--text-primary);
  font-family: var(--font-mono);
}

.modal-form {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-top: var(--space-2);
}

.char-count {
  text-align: right;
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: -var(--space-2);
}
.char-count.near-limit { color: var(--color-accent-amber); }

.btn-lg { width: 100%; }

/* Screen-reader only */
.sr-only {
  position: absolute; width: 1px; height: 1px;
  padding: 0; margin: -1px; overflow: hidden;
  clip: rect(0,0,0,0); white-space: nowrap; border: 0;
}
</style>
