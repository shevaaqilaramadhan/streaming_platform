<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="modelValue"
        class="modal-backdrop"
        role="dialog"
        aria-modal="true"
        aria-labelledby="room-settings-title"
        @click.self="$emit('update:modelValue', false)"
        @keydown.esc="$emit('update:modelValue', false)"
      >
        <div class="modal-card glass-strong animate-fade-in-scale">
          <!-- Header -->
          <div class="modal-header">
            <div class="modal-title-group">
              <h2 id="room-settings-title" class="modal-title">Room Settings</h2>
              <span class="room-id-chip">ID: {{ roomId }}</span>
            </div>
            <button
              class="modal-close-btn"
              @click="$emit('update:modelValue', false)"
              aria-label="Close settings"
            >
              &times;
            </button>
          </div>

          <!-- Form body -->
          <form @submit.prevent="handleSave" class="settings-form">
            <!-- Setting 1: Room Name -->
            <div class="form-group">
              <label for="settings-room-name" class="form-label">Room Name</label>
              <input
                id="settings-room-name"
                v-model="localRoomName"
                type="text"
                class="input"
                placeholder="e.g. Anime Night 1080p"
                maxlength="40"
              />
              <span class="form-hint">Displayed in the header and public lobby</span>
            </div>

            <!-- Setting 2: Privacy / Lobby -->
            <div class="form-group">
              <span class="form-label">Visibility</span>
              <div class="visibility-options">
                <button
                  type="button"
                  class="vis-btn"
                  :class="{ 'vis-btn--active': localIsPublic }"
                  @click="localIsPublic = true"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="2" y1="12" x2="22" y2="12"/>
                    <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
                  </svg>
                  <div class="vis-text">
                    <strong>Public Room</strong>
                    <span>Listed in public lobby</span>
                  </div>
                </button>

                <button
                  type="button"
                  class="vis-btn"
                  :class="{ 'vis-btn--active': !localIsPublic }"
                  @click="localIsPublic = false"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                    <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                  </svg>
                  <div class="vis-text">
                    <strong>Private Room</strong>
                    <span>Join with link only</span>
                  </div>
                </button>
              </div>
            </div>

            <!-- Setting 3: Access PIN -->
            <div class="form-group">
              <div class="pin-toggle-header">
                <span class="form-label">Room PIN Protection</span>
                <span v-if="hasPin" class="status-badge status-badge--active">Active</span>
                <span v-else class="status-badge status-badge--off">Disabled</span>
              </div>

              <label class="checkbox-row">
                <input type="checkbox" v-model="localEnablePin" />
                <span>Require PIN to join this room</span>
              </label>

              <Transition name="fade">
                <div v-if="localEnablePin" class="pin-input-wrap">
                  <input
                    v-model="localPinValue"
                    type="password"
                    class="input pin-field"
                    :placeholder="hasPin ? 'Enter new PIN (or leave blank to keep current)' : 'Enter 4-16 char PIN'"
                    maxlength="16"
                    autocomplete="off"
                  />
                  <span class="form-hint">Only participants with this PIN can enter the room</span>
                </div>
              </Transition>
            </div>

            <!-- Modal Actions -->
            <div class="modal-actions">
              <button
                type="button"
                class="btn btn-ghost"
                @click="$emit('update:modelValue', false)"
              >
                Cancel
              </button>
              <button
                type="submit"
                class="btn btn-primary"
              >
                Save Settings
              </button>
            </div>
          </form>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  roomId:     { type: String, required: true },
  roomName:   { type: String, default: '' },
  isPublic:   { type: Boolean, default: false },
  hasPin:     { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'save'])

const localRoomName  = ref('')
const localIsPublic  = ref(false)
const localEnablePin = ref(false)
const localPinValue  = ref('')

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      localRoomName.value  = props.roomName || ''
      localIsPublic.value  = props.isPublic
      localEnablePin.value = props.hasPin
      localPinValue.value  = ''
    }
  },
  { immediate: true }
)

function handleSave() {
  const result = {
    roomName: localRoomName.value.trim(),
    isPublic: localIsPublic.value,
    pin: '',
    removePin: false,
  }

  if (localEnablePin.value) {
    if (localPinValue.value.trim()) {
      result.pin = localPinValue.value.trim()
    }
  } else if (props.hasPin) {
    // Host disabled PIN
    result.removePin = true
    result.pin = ''
  }

  emit('save', result)
  emit('update:modelValue', false)
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(4, 4, 5, 0.78);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: var(--z-modal, 1000);
  padding: var(--space-4);
}

.modal-card {
  width: 100%;
  max-width: 460px;
  border-radius: var(--radius-xl);
  padding: var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  box-shadow: 0 24px 60px rgba(0,0,0,0.5);
  background: var(--color-bg-surface, #141416);
  border: 1px solid var(--color-border);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.modal-title-group {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.modal-title {
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin: 0;
}

.room-id-chip {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-muted);
}

.modal-close-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  padding: 0 4px;
  transition: color var(--transition-fast);
}
.modal-close-btn:hover {
  color: var(--text-primary);
}

.settings-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.form-hint {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.visibility-options {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-2);
  margin-top: 2px;
}

.vis-btn {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--color-border);
  color: var(--text-muted);
  cursor: pointer;
  text-align: left;
  transition: all var(--transition-fast);
}
.vis-btn:hover {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-primary);
}
.vis-btn--active {
  background: rgba(255, 255, 255, 0.08);
  border-color: var(--color-primary);
  color: var(--text-primary);
}

.vis-text {
  display: flex;
  flex-direction: column;
  line-height: 1.25;
}
.vis-text strong {
  font-size: 0.8125rem;
}
.vis-text span {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.pin-toggle-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.status-badge {
  font-size: 0.7rem;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
}
.status-badge--active {
  background: rgba(245, 158, 11, 0.12);
  color: var(--color-accent-amber, #f59e0b);
  border: 1px solid rgba(245, 158, 11, 0.25);
}
.status-badge--off {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-muted);
  border: 1px solid var(--color-border);
}

.checkbox-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  color: var(--text-primary);
  cursor: pointer;
  user-select: none;
  margin-top: 2px;
}
.checkbox-row input[type="checkbox"] {
  accent-color: var(--color-primary);
  cursor: pointer;
}

.pin-input-wrap {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-top: var(--space-2);
}

.pin-field {
  letter-spacing: 0.08em;
}

.modal-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}
</style>
