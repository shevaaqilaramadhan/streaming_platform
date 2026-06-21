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
              <stop offset="0%" stop-color="hsl(260,80%,62%)"/>
              <stop offset="100%" stop-color="hsl(220,85%,60%)"/>
            </linearGradient>
          </defs>
        </svg>
        <span class="logo-name gradient-text">WatchParty</span>
      </div>

      <div class="divider"></div>

      <div class="room-info">
        <span class="room-label">Room</span>
        <span class="room-id">{{ roomId }}</span>
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
      </div>
    </div>

    <!-- Right: User info + status -->
    <div class="room-header__right">
      <!-- Participant count -->
      <div class="participant-count" data-tooltip="Viewers in room">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
          <circle cx="9" cy="7" r="4"/>
          <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
        </svg>
        <span>{{ participantCount }}</span>
      </div>

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
    </div>
  </header>
</template>

<script setup>
import { ref } from 'vue'
import ConnectionStatus from './ConnectionStatus.vue'

const props = defineProps({
  roomId:           { type: String, required: true },
  nickname:         { type: String, required: true },
  isHost:           { type: Boolean, default: false },
  wsStatus:         { type: String, required: true },
  participantCount: { type: Number, default: 1 },
})

const copied = ref(false)

function copyLink() {
  const url = window.location.href
  navigator.clipboard.writeText(url).then(() => {
    copied.value = true
    setTimeout(() => { copied.value = false }, 2500)
  })
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

.participant-count {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  font-weight: 600;
}

@media (max-width: 600px) {
  .logo-name { display: none; }
  .divider { display: none; }
  .room-label { display: none; }
  .room-header { padding: var(--space-3) var(--space-4); }
}
</style>
