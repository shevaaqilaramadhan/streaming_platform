<template>
  <header class="room-header glass">
    <!-- Left: Brand + Room Identity -->
    <div class="room-header__left">
      <div class="logo">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
          <rect x="1" y="1" width="22" height="22" rx="5" stroke="currentColor" stroke-width="1.5"/>
          <polygon points="10,7 18,12 10,17" fill="currentColor"/>
        </svg>
        <span class="logo-name">WatchParty</span>
      </div>

      <div class="divider"></div>

      <!-- Room identity items -->
      <div class="room-info">
        <span v-if="displayName" class="room-name" :title="displayName">
          {{ displayName }}
        </span>

        <!-- Room code chip with 1-click copy -->
        <button
          id="copy-room-link-btn"
          class="room-code-chip"
          :class="{ 'room-code-chip--copied': copied }"
          @click="copyLink"
          data-tooltip="Click to copy invite link"
          aria-label="Copy room invite link"
        >
          <span class="room-code-label">Room:</span>
          <strong>{{ roomId }}</strong>
          <svg v-if="!copied" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
          </svg>
          <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        </button>

        <!-- Status badges -->
        <span
          class="status-pill"
          :class="isPublic ? 'status-pill--public' : 'status-pill--private'"
          :data-tooltip="isPublic ? 'Public room (listed in lobby)' : 'Private room (link only)'"
        >
          {{ isPublic ? 'Public' : 'Private' }}
        </span>

        <span
          v-if="hasPin"
          class="status-pill status-pill--pin"
          data-tooltip="Protected by Room PIN"
        >
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
          PIN
        </span>

        <!-- Host Room Settings trigger -->
        <button
          v-if="isHost"
          id="room-settings-btn"
          class="btn btn-ghost btn-sm settings-trigger-btn"
          @click="showSettingsModal = true"
          data-tooltip="Room settings & security"
          aria-label="Room Settings"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
          </svg>
          <span class="settings-btn-label">Settings</span>
        </button>
      </div>
    </div>

    <!-- Right: User Info + Status -->
    <div class="room-header__right">
      <!-- Participant count button -->
      <button
        class="participant-count-btn"
        data-tooltip="View participants"
        aria-label="View participants"
        @click="$emit('toggle-user-list')"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
          <circle cx="9" cy="7" r="4"/>
          <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
        </svg>
        <span>{{ participantCount }}</span>
      </button>

      <!-- User role badge -->
      <span class="badge" :class="isHost ? 'badge-host' : 'badge-guest'">
        <svg v-if="isHost" width="10" height="10" viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
        </svg>
        <svg v-else width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
          <circle cx="12" cy="7" r="4"/>
        </svg>
        {{ isHost ? 'Host' : nickname }}
      </span>

      <!-- Connection Status -->
      <ConnectionStatus :status="wsStatus" />

      <!-- Theme Toggle -->
      <ThemeToggle />
    </div>

    <!-- Host Settings Modal -->
    <RoomSettingsModal
      v-if="isHost"
      v-model="showSettingsModal"
      :room-id="roomId"
      :room-name="roomName"
      :is-public="isPublic"
      :has-pin="hasPin"
      @save="onSaveSettings"
    />
  </header>
</template>

<script setup>
import { ref, computed } from 'vue'
import ConnectionStatus from './ConnectionStatus.vue'
import ThemeToggle from './ThemeToggle.vue'
import RoomSettingsModal from './RoomSettingsModal.vue'

const props = defineProps({
  roomId:           { type: String, required: true },
  nickname:         { type: String, required: true },
  isHost:           { type: Boolean, default: false },
  wsStatus:         { type: String, required: true },
  participantCount: { type: Number, default: 1 },
  isPublic:         { type: Boolean, default: false },
  hasPin:           { type: Boolean, default: false },
  roomName:         { type: String, default: '' },
})

const emit = defineEmits(['toggle-public', 'toggle-user-list', 'set-room-name', 'set-room-pin'])

const copied = ref(false)
const showSettingsModal = ref(false)

const displayName = computed(() => props.roomName || '')

function copyLink() {
  const url = new URL(window.location.href)
  url.search = ''
  navigator.clipboard.writeText(url.toString()).then(() => {
    copied.value = true
    setTimeout(() => { copied.value = false }, 2500)
  }).catch((err) => {
    console.warn('[RoomHeader] Clipboard write failed:', err)
  })
}

function onSaveSettings(settings) {
  if (settings.roomName !== (props.roomName || '')) {
    emit('set-room-name', settings.roomName)
  }
  if (settings.isPublic !== props.isPublic) {
    emit('toggle-public', settings.isPublic)
  }
  if (settings.removePin) {
    emit('set-room-pin', '')
  } else if (settings.pin) {
    emit('set-room-pin', settings.pin)
  }
}
</script>

<style scoped>
.room-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--space-6);
  height: 58px;
  gap: var(--space-4);
  position: sticky;
  top: 0;
  z-index: var(--z-overlay);
  border-radius: 0;
  border-top: none;
  border-left: none;
  border-right: none;
  border-bottom: 1px solid var(--color-border);
  background: rgba(10, 10, 12, 0.85);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

.room-header__left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
  flex: 1;
}

.room-header__right {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-shrink: 0;
}

.logo {
  display: flex;
  align-items: center;
  gap: 7px;
  flex-shrink: 0;
}
.logo-name {
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--text-primary);
}

.divider {
  width: 1px;
  height: 18px;
  background: var(--color-border);
  flex-shrink: 0;
}

.room-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
  overflow: hidden;
}

.room-name {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--text-primary);
  max-width: 14rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 1;
}

/* Room Code Chip (clickable copy) */
.room-code-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: 3px 8px;
  font-size: 0.8125rem;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}
.room-code-chip:hover {
  background: rgba(255, 255, 255, 0.09);
  color: var(--text-primary);
  border-color: rgba(255, 255, 255, 0.15);
}
.room-code-chip strong {
  font-family: var(--font-mono);
  color: var(--text-primary);
}
.room-code-label {
  font-size: 0.72rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.room-code-chip--copied {
  color: var(--color-accent-green) !important;
  border-color: var(--color-accent-green) !important;
}

/* Status pills */
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 7px;
  border-radius: var(--radius-sm);
  font-size: 0.72rem;
  font-weight: 600;
  flex-shrink: 0;
}
.status-pill--public {
  color: var(--text-secondary);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--color-border);
}
.status-pill--private {
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--color-border);
}
.status-pill--pin {
  color: var(--color-accent-amber, #f59e0b);
  background: rgba(245, 158, 11, 0.08);
  border: 1px solid rgba(245, 158, 11, 0.25);
}

/* Host settings button */
.settings-trigger-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  font-size: 0.78rem;
  flex-shrink: 0;
  color: var(--text-secondary);
  border: 1px solid transparent;
}
.settings-trigger-btn:hover {
  color: var(--text-primary);
  border-color: var(--color-border);
  background: rgba(255, 255, 255, 0.05);
}

.participant-count-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  font-weight: 600;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  padding: 4px 10px;
  cursor: pointer;
  transition: all var(--transition-fast);
  font-family: var(--font-sans);
  flex-shrink: 0;
}
.participant-count-btn:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(255, 255, 255, 0.12);
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 600;
  flex-shrink: 0;
}
.badge-host {
  background: rgba(245, 158, 11, 0.12);
  color: var(--color-accent-amber, #f59e0b);
  border: 1px solid rgba(245, 158, 11, 0.25);
}
.badge-guest {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-secondary);
  border: 1px solid var(--color-border);
}

@media (max-width: 768px) {
  .logo-name { display: none; }
  .settings-btn-label { display: none; }
  .room-header { padding: 0 var(--space-4); }
}

@media (max-width: 520px) {
  .divider { display: none; }
  .room-code-label { display: none; }
  .status-pill { display: none; }
}
</style>
