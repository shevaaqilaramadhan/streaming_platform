<template>
  <div class="lobby-page">
    <!-- Nav (reuse from HomePage style) -->
    <nav class="home-nav">
      <div class="nav-logo">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
          <rect x="1" y="1" width="22" height="22" rx="5" stroke="currentColor" stroke-width="1.5"/>
          <polygon points="10,7 18,12 10,17" fill="currentColor"/>
        </svg>
        <span class="nav-brand">LitcqnWatch</span>
      </div>
      <div class="nav-right">
        <router-link to="/" class="nav-link">Home</router-link>
        <ThemeToggle />
      </div>
    </nav>

    <!-- Header -->
    <header class="lobby-header">
      <h1 class="lobby-title">Public Rooms</h1>
      <p class="lobby-subtitle">Join an active watch party or create your own</p>
    </header>

    <!-- Loading -->
    <div v-if="loading" class="lobby-loading">
      <div class="spinner"></div>
      <span>Loading active rooms…</span>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="lobby-empty glass">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
        <circle cx="12" cy="12" r="10"/>
        <line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>
      </svg>
      <p>Failed to load rooms</p>
      <small>{{ error }}</small>
      <button class="btn btn-primary" style="margin-top: 1rem;" @click="fetchPublicRooms">
        Retry
      </button>
    </div>

    <!-- Empty State -->
    <div v-else-if="rooms.length === 0" class="lobby-empty glass">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
        <circle cx="12" cy="12" r="10"/>
        <path d="M8 12h8"/>
      </svg>
      <p>No active public rooms right now</p>
      <small>Be the first to create a watch party!</small>
      <router-link to="/" class="btn btn-primary" style="margin-top: 1rem;">
        Create Room
      </router-link>
    </div>

    <!-- Room Grid -->
    <div v-else class="room-grid">
      <div
        v-for="room in rooms"
        :key="room.roomId"
        class="room-card glass"
        @click="joinRoom(room.roomId)"
      >
        <div class="room-card-thumbnail">
          <img
            v-if="room.currentMetadata?.thumbnail"
            :src="room.currentMetadata.thumbnail"
            :alt="room.currentMetadata?.title || 'Thumbnail'"
            referrerpolicy="no-referrer"
          />
          <div v-else class="room-card-thumb-fallback">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
              <polygon points="5 3 19 12 5 21 5 3"/>
            </svg>
          </div>
          <div class="room-card-overlay">
            <span class="room-card-viewers">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
              {{ room.participantCount }} watching
            </span>
          </div>
        </div>

        <div class="room-card-info">
          <h3 class="room-card-title">
            {{ getRoomTitle(room) }}
          </h3>
          <p v-if="room.currentMetadata?.episode" class="room-card-episode">
            {{ room.currentMetadata.episode }}
          </p>
          <div class="room-card-meta">
            <span class="room-card-host">{{ room.hostUsername }}</span>
            <span v-if="room.queueSize > 0" class="room-card-queue">
              {{ room.queueSize }} queued
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, onDeactivated } from 'vue'
import { useRouter } from 'vue-router'
import ThemeToggle from '../components/ThemeToggle.vue'
import { API_BASE } from '../config.js'

const router = useRouter()
const rooms = ref([])
const loading = ref(true)
const error = ref(null)
let pollInterval = null

function stopPolling() {
  if (pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
}

async function fetchPublicRooms() {
  try {
    const res = await fetch(`${API_BASE}/api/public-rooms`)
    if (res.ok) {
      rooms.value = await res.json()
      error.value = null
    } else {
      error.value = `Server returned ${res.status}`
    }
  } catch (err) {
    console.error('[Lobby] Failed to fetch rooms:', err)
    error.value = 'Unable to connect to the server. Please check your connection.'
  } finally {
    loading.value = false
  }
}

function getRoomTitle(room) {
  if (room.roomName) return room.roomName
  if (room.currentMetadata?.title) return room.currentMetadata.title
  return `${room.hostUsername}'s Room`
}

function joinRoom(roomId) {
  router.push({ name: 'room', params: { roomId } })
}

onMounted(() => {
  fetchPublicRooms()
  pollInterval = setInterval(fetchPublicRooms, 15000)
})

onUnmounted(() => {
  stopPolling()
})

// FE-L02: defense if KeepAlive is ever used (onUnmounted alone won't fire)
onDeactivated(() => {
  stopPolling()
})
</script>

<style scoped>
.lobby-page {
  min-height: 100vh;
  padding-bottom: var(--space-16);
}

.home-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--space-8);
  height: var(--nav-height);
  position: sticky;
  top: 0;
  z-index: var(--z-overlay);
  background: rgba(10,10,10,0.85);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--color-border);
}

.nav-logo {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-primary);
  font-weight: 600;
  font-size: 0.9375rem;
  letter-spacing: -0.02em;
}

.nav-right {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-left: auto;
}

.nav-link {
  color: var(--text-secondary);
  font-weight: 500;
  font-size: 0.8125rem;
  text-decoration: none;
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.nav-link:hover {
  color: var(--text-primary);
  background: var(--color-fill);
}

.lobby-header {
  text-align: center;
  padding: var(--space-10) var(--space-4) var(--space-6);
}

.lobby-title {
  font-size: clamp(1.8rem, 4vw, 2.8rem);
  font-weight: 800;
  letter-spacing: -0.04em;
  margin: 0 0 var(--space-2);
  color: var(--text-primary);
}

.lobby-subtitle {
  color: var(--text-secondary);
  font-size: 1rem;
  margin: 0;
}

.lobby-loading,
.lobby-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-12) var(--space-4);
  text-align: center;
  color: var(--text-muted);
  max-width: 400px;
  margin: 0 auto;
  border-radius: var(--radius-lg);
}

.spinner {
  width: 32px; height: 32px;
  border: 2px solid rgba(255,255,255,0.08);
  border-top-color: var(--text-secondary);
  border-radius: 50%;
  animation: spin-slow 0.8s linear infinite;
}

.room-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-5);
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 var(--space-6);
}

@media (max-width: 414px) {
  .room-grid {
    padding: 0 var(--space-3);
    gap: var(--space-4);
    grid-template-columns: 1fr;
  }
  .lobby-header {
    padding: var(--space-8) var(--space-3) var(--space-4);
  }
  .lobby-title {
    font-size: 1.6rem;
  }
}

.room-card {
  border-radius: var(--radius-lg);
  overflow: hidden;
  cursor: pointer;
  transition: all var(--transition-base);
}

.room-card:hover {
  transform: translateY(-4px);
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: var(--shadow-card);
}

.room-card-thumbnail {
  position: relative;
  height: 180px;
  overflow: hidden;
  background: #000;
}

.room-card-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.room-card-thumb-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(14,16,26,0.9) 0%, rgba(20,24,40,0.95) 100%);
  color: rgba(255,255,255,0.15);
}

.room-card-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: var(--space-3);
  background: linear-gradient(to top, rgba(0,0,0,0.8) 0%, transparent 100%);
  display: flex;
  justify-content: flex-end;
}

.room-card-viewers {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 600;
  background: rgba(0, 0, 0, 0.6);
  color: white;
}

.room-card-info {
  padding: var(--space-4);
}

.room-card-title {
  margin: 0 0 var(--space-1);
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.room-card-episode {
  margin: 0 0 var(--space-2);
  font-size: 0.8125rem;
  color: var(--text-secondary);
}

.room-card-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.room-card-host {
  color: var(--text-secondary);
  font-weight: 600;
}
</style>
