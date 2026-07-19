<template>
  <header class="room-header glass">
    <!-- Left: Logo + Room Info -->
    <div class="room-header__left">
      <div class="logo">
        <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
          <circle cx="14" cy="14" r="14" fill="url(#logo-grad)"/>
          <polygon points="11,9 21,14 11,19" fill="white"/>
          <defs>
            <linearGradient id="logo-grad" x1="0" y1="0" x2="28" y2="28" gradientUnits="userSpaceOnUse">
              <stop offset="0%" stop-color="hsl(195,100%,45%)"/>
              <stop offset="100%" stop-color="hsl(215,90%,50%)"/>
            </linearGradient>
          </defs>
        </svg>
        <span class="logo-name gradient-text">WatchParty</span>
      </div>

      <div class="divider"></div>

      <div class="room-info">
        <!-- Room name display / inline edit -->
        <div v-if="isEditingName" class="room-name-edit">
          <input
            ref="nameInputRef"
            v-model="editingName"
            class="room-name-input"
            type="text"
            maxlength="40"
            placeholder="Room name"
            @keydown.enter="saveName"
            @keydown.esc="cancelEditName"
            @blur="saveName"
          />
        </div>
        <template v-else>
          <span v-if="displayName" class="room-name" @click="startEditName" :data-tooltip="isHost ? 'Click to edit' : ''">
            {{ displayName }}
          </span>
          <span class="room-label">Room</span>
          <span class="room-id">{{ roomId }}</span>
        </template>

        <button
          v-if="isHost && !isEditingName"
          class="btn btn-ghost btn-sm edit-name-btn"
          @click="startEditName"
          data-tooltip="Edit room name"
          aria-label="Edit room name"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
          </svg>
        </button>

        <button
          id="copy-room-link-btn"
          class="btn btn-ghost btn-sm copy-btn"
          :class="{ 'copy-btn--copied': copied }"
          @click="copyLink"
          data-tooltip="Copy invite link"
          aria-label="Copy room invite link"
        >
          <svg v-if="!copied" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
          </svg>
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          <span>{{ copied ? 'Copied!' : 'Share' }}</span>
        </button>

        <!-- Room Visibility Status / Toggle -->
        <button
          v-if="isHost"
          id="toggle-visibility-btn"
          class="btn btn-ghost btn-sm visibility-btn"
          :class="{ 'visibility-btn--public': isPublic }"
          @click="$emit('toggle-public', !isPublic)"
          :data-tooltip="isPublic ? 'Make room private' : 'Make room public'"
          aria-label="Toggle room public visibility"
        >
          <svg v-if="isPublic" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="2" y1="12" x2="22" y2="12"/>
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
          </svg>
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
          <span>{{ isPublic ? 'Public' : 'Private' }}</span>
        </button>

        <span
          v-else
          class="room-visibility-badge"
          :class="{ 'room-visibility-badge--public': isPublic }"
          :data-tooltip="isPublic ? 'This room is public and appears in the lobby' : 'This room is private and can only be joined via link'"
        >
          <svg v-if="isPublic" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="2" y1="12" x2="22" y2="12"/>
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
          </svg>
          <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
          <span>{{ isPublic ? 'Public' : 'Private' }}</span>
        </span>
      </div>
    </div>

    <!-- Right: User info + status -->
    <div class="room-header__right">
      <!-- Participant count (clickable) -->
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

      <!-- User badge -->
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

      <ConnectionStatus :status="wsStatus" />
      <ThemeToggle />
    </div>
  </header>
</template>

<script setup>
import { ref, nextTick, computed } from 'vue'
import ConnectionStatus from './ConnectionStatus.vue'
import ThemeToggle from './ThemeToggle.vue'

const props = defineProps({
  roomId:           { type: String, required: true },
  nickname:         { type: String, required: true },
  isHost:           { type: Boolean, default: false },
  wsStatus:         { type: String, required: true },
  participantCount: { type: Number, default: 1 },
  isPublic:         { type: Boolean, default: false },
  roomName:         { type: String, default: '' },
})

const emit = defineEmits(['toggle-public', 'toggle-user-list', 'set-room-name'])

const copied = ref(false)
const isEditingName = ref(false)
const editingName = ref('')
const nameInputRef = ref(null)

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

function startEditName() {
  if (!props.isHost) return
  editingName.value = props.roomName || ''
  isEditingName.value = true
  nextTick(() => {
    nameInputRef.value?.focus()
    nameInputRef.value?.select()
  })
}

function saveName() {
  if (!isEditingName.value) return
  const name = editingName.value.trim()
  isEditingName.value = false
  if (name !== (props.roomName || '')) {
    emit('set-room-name', name)
  }
}

function cancelEditName() {
  isEditingName.value = false
}
</script>

<style scoped>
.room-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-6);
  gap: var(--space-4);
  position: sticky;
  top: 0;
  z-index: var(--z-overlay);
  border-radius: 0;
  border-top: none;
  border-left: none;
  border-right: none;
}

.room-header__left,
.room-header__right {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.logo {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.logo-name {
  font-size: 1.1rem;
  font-weight: 800;
  letter-spacing: -0.03em;
}

.divider {
  width: 1px;
  height: 20px;
  background: var(--color-glass-border);
}

.room-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.room-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-weight: 600;
}
.room-id {
  font-size: 0.875rem;
  font-weight: 600;
  font-family: var(--font-mono);
  color: var(--text-primary);
  background: rgba(255,255,255,0.06);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-glass-border);
}

.copy-btn { gap: 4px; }
.copy-btn--copied { color: var(--color-accent-green) !important; }

.visibility-btn {
  gap: 4px;
  color: var(--text-muted);
}
.visibility-btn--public {
  color: var(--color-primary) !important;
  background: var(--grad-brand-subtle);
  border-color: hsla(195, 100%, 45%, 0.2);
}

.room-visibility-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--color-glass-border);
}
.room-visibility-badge--public {
  color: var(--color-primary);
  background: var(--grad-brand-subtle);
  border-color: hsla(195, 100%, 45%, 0.1);
}

.participant-count {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  font-weight: 600;
}

.participant-count-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  font-weight: 600;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-full);
  padding: 4px 10px;
  cursor: pointer;
  transition: all var(--transition-fast);
  font-family: var(--font-sans);
}
.participant-count-btn:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(255, 255, 255, 0.12);
}

/* Room name */
.room-name {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--color-primary);
  cursor: default;
  max-width: 12rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.is-host .room-name,
.room-name[data-tooltip] {
  cursor: pointer;
}

.edit-name-btn {
  padding: 4px;
  width: 28px;
  height: 28px;
}

.room-name-edit {
  display: flex;
  align-items: center;
}

.room-name-input {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: var(--font-sans);
  font-size: 0.85rem;
  font-weight: 600;
  padding: 3px 8px;
  width: 160px;
  outline: none;
  box-shadow: 0 0 0 3px hsla(195, 100%, 45%, 0.15);
}

@media (max-width: 600px) {
  .logo-name { display: none; }
  .divider { display: none; }
  .room-label { display: none; }
  .room-header { padding: var(--space-3) var(--space-4); }
}

@media (max-width: 414px) {
  .room-header {
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    flex-wrap: wrap;
  }
  .room-header__left {
    flex: 1 1 auto;
    min-width: 0;
    gap: var(--space-2);
    overflow: hidden;
  }
  .room-header__right {
    flex-shrink: 0;
    gap: var(--space-2);
  }
  .room-info {
    min-width: 0;
    flex-wrap: wrap;
  }
  .room-id {
    font-size: 0.75rem;
    padding: 2px 6px;
    max-width: 8rem;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .room-name {
    max-width: 6rem;
    font-size: 0.8rem;
  }
  .room-name-input {
    width: 120px;
  }
  .copy-btn span { display: none; }
  .visibility-btn span { display: none; }
  .room-visibility-badge span { display: none; }
  .badge { font-size: 0.6875rem; padding: 2px 6px; }
  .badge { max-width: 5.5rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
}
</style>
