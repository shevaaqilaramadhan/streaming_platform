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
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.5">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
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
            <span v-if="requiresPin" class="pin-required-tag">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
              PIN Protected
            </span>
          </div>

          <!-- Input -->
          <form @submit.prevent="handleSubmit" class="modal-form">
            <!-- Error Alert -->
            <div v-if="errorMessage" class="modal-error">
              {{ errorMessage }}
            </div>

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

            <!-- PIN Field when room is PIN-protected -->
            <div v-if="requiresPin" class="pin-field-wrap">
              <label for="pin-input" class="field-label">Room PIN</label>
              <input
                id="pin-input"
                v-model="localPin"
                type="password"
                class="input"
                placeholder="Enter room PIN"
                maxlength="16"
                autocomplete="off"
              />
            </div>

            <button
              id="join-room-btn"
              type="submit"
              class="btn btn-primary btn-lg"
              :disabled="!localNickname.trim() || localNickname.trim().length < 2 || (requiresPin && !localPin.trim())"
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
  modelValue:   { type: Boolean, default: false },
  roomId:       { type: String, required: true },
  requiresPin:  { type: Boolean, default: false },
  errorMessage: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'join', 'close'])

const localNickname = ref('')
const localPin = ref('')
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
  if (props.requiresPin && !localPin.value.trim()) return
  emit('join', { nickname: name, pin: localPin.value.trim() })
  emit('update:modelValue', false)
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(4, 4, 5, 0.8);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: var(--z-modal);
  padding: var(--space-4);
}

.modal-card {
  width: 100%;
  max-width: 400px;
  border-radius: var(--radius-xl);
  padding: var(--space-10) var(--space-8);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  box-shadow: 0 24px 60px rgba(0,0,0,0.5);
}

.modal-icon {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-lg);
  background: rgba(255,255,255,0.05);
  border: 1px solid rgba(255,255,255,0.08);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: var(--space-2);
}

.modal-title {
  font-size: 1.4rem;
  text-align: center;
  letter-spacing: -0.02em;
}

.modal-subtitle {
  font-size: 0.9rem;
  text-align: center;
  color: var(--text-secondary);
  margin-top: calc(-1 * var(--space-2));
}

.room-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.06);
  font-size: 0.8125rem;
  color: var(--text-muted);
}
.room-chip strong {
  color: var(--text-primary);
  font-family: var(--font-mono);
}
.pin-required-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
  color: #fbbf24;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: 0.6875rem;
  font-weight: 600;
  margin-left: 4px;
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
  margin-top: calc(-1 * var(--space-2));
}
.char-count.near-limit { color: var(--color-accent-amber); }

.btn-lg { width: 100%; }

/* Screen-reader only */
.sr-only {
  position: absolute; width: 1px; height: 1px;
  padding: 0; margin: -1px; overflow: hidden;
  clip: rect(0,0,0,0); white-space: nowrap; border: 0;
}

.modal-error {
  width: 100%;
  padding: 0.5rem 0.75rem;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: var(--radius-md);
  color: #fca5a5;
  font-size: 0.8rem;
  text-align: center;
}

.pin-field-wrap {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.field-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
</style>
