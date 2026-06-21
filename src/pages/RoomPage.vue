<template>
  <div class="room-page">
    <!-- Nickname modal for guests -->
    <NicknameModal
      v-model="showModal"
      :room-id="roomId"
      @join="onNicknameChosen"
      @close="showModal = false"
    />

    <template v-if="hasJoined">
      <!-- Header -->
      <RoomHeader
        :room-id="roomId"
        :nickname="nickname"
        :is-host="isHost"
        :ws-status="wsStatus"
        :participant-count="participants.length || 1"
      />

      <!-- Main content -->
      <div class="room-layout">
        <!-- Video column -->
        <section class="video-col" aria-label="Video player">
          <!-- Now Watching Header -->
          <div v-if="currentMetadata && (currentMetadata.title || currentMetadata.source)" class="now-watching-header glass animate-fade-in">
            <img 
              v-if="currentMetadata.thumbnailUrl" 
              :src="currentMetadata.thumbnailUrl" 
              alt="Anime Cover"
              class="anime-thumbnail"
              referrerpolicy="no-referrer"
            />
            <div class="metadata-info">
              <span class="watching-label">NOW WATCHING</span>
              <h2 class="anime-title">
                {{ currentMetadata.title || 'Stream Source' }}
              </h2>
              <span v-if="currentMetadata.episode" class="episode-info">
                {{ currentMetadata.episode }}
              </span>
              <span v-if="!currentMetadata.title && currentMetadata.source" class="stream-source">
                {{ currentMetadata.source }}
              </span>
            </div>
          </div>

          <VideoPlayer
            :is-host="isHost"
            :video-url="playerState.videoUrl"
            :player-state="playerState"
            @sync="onSyncEvent"
            @set-video="onSetVideo"
          />
        </section>

        <!-- Chat column -->
        <aside class="chat-col" aria-label="Chat panel">
          <ChatPanel
            :messages="messages"
            :is-connected="isConnected"
            :nickname="nickname"
            @send="onSendChat"
          />
        </aside>
      </div>
    </template>

    <!-- Disconnected banner -->
    <Transition name="slide-up">
      <div
        v-if="hasJoined && wsStatus === 'disconnected'"
        class="disconnected-banner glass"
        role="alert"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        Lost connection to the room. Attempting to reconnect…
      </div>
    </Transition>

    <!-- Scrape error toast -->
    <Transition name="slide-up">
      <div v-if="scrapeError" class="scrape-error-toast glass" role="alert">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
        <div class="scrape-error-text">
          <strong>Failed to extract stream</strong>
          <span>{{ scrapeError.error }}</span>
        </div>
        <button class="scrape-error-dismiss" @click="scrapeError = null" aria-label="Dismiss">&times;</button>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import RoomHeader   from '../components/RoomHeader.vue'
import VideoPlayer  from '../components/VideoPlayer.vue'
import ChatPanel    from '../components/ChatPanel.vue'
import NicknameModal from '../components/NicknameModal.vue'
import { useRoom } from '../composables/useRoom.js'

const route   = useRoute()
const roomId  = route.params.roomId
const isHost  = route.query.host === 'true'

/* ---- Local State ---- */
const nickname   = ref(isHost ? 'Host' : '')
const showModal  = ref(!isHost)
const hasJoined  = ref(isHost)   // host enters immediately, guest waits for nickname

/* ---- Room composable ---- */
// We initialize lazily (after nickname is known for guests)
let room = null

const wsStatus        = ref('disconnected')
const messages        = ref([])
const playerState     = ref({ videoUrl: '', currentTime: 0, isPlaying: false })
const participants    = ref([])
const scrapeError     = ref(null)
const currentMetadata = ref(null)
const isConnected     = computed(() => wsStatus.value === 'connected')

function initRoom() {
  room = useRoom(roomId, nickname.value, isHost)
  // Sync reactive refs initially
  wsStatus.value        = room.status.value
  messages.value        = room.messages.value
  playerState.value     = room.playerState.value
  participants.value    = room.participants.value
  currentMetadata.value = room.currentMetadata.value

  // Keep local refs in sync with the composable's reactive state
  watch(room.status,          v => { wsStatus.value    = v })
  watch(room.messages,        v => { messages.value    = v }, { deep: true })
  watch(room.playerState,     v => { playerState.value = v }, { deep: true })
  watch(room.participants,    v => { participants.value = v }, { deep: true })
  watch(room.scrapeError,     v => { scrapeError.value  = v })
  watch(room.currentMetadata, v => { currentMetadata.value = v }, { deep: true })

  room.joinRoom()
}

/* ---- Lifecycle ---- */
onMounted(() => {
  if (isHost) {
    initRoom()
  }
})

/* ---- Event Handlers ---- */
function onNicknameChosen(name) {
  nickname.value  = name
  hasJoined.value = true
  showModal.value = false
  initRoom()
}

function onSyncEvent({ isPlaying, currentTime }) {
  room?.sendSyncEvent({ isPlaying, currentTime })
}

function onSetVideo(url) {
  room?.setVideo(url)
}

function onSendChat(text) {
  room?.sendChatMessage(text)
}
</script>

<style scoped>
.room-page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--color-bg-base);
}

.room-layout {
  display: grid;
  grid-template-columns: 1fr 385px;
  grid-template-rows: 1fr;
  gap: var(--space-6);
  flex: 1;
  padding: var(--space-6);
  min-height: 0;
  max-height: calc(100vh - 72px);
  overflow: hidden;
}

.video-col {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

/* Now Watching Header Styles */
.now-watching-header {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4);
  border-radius: var(--radius-lg);
  border-left: 3px solid var(--color-primary);
  box-shadow: var(--shadow-card);
}

.anime-thumbnail {
  width: 60px;
  height: 80px;
  object-fit: cover;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-glass-border);
}

.metadata-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.watching-label {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--color-primary);
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.anime-title {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 800;
  color: var(--text-primary);
  line-height: 1.2;
}

.episode-info {
  font-size: 0.875rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.stream-source {
  font-size: 0.75rem;
  color: var(--text-muted);
  font-style: italic;
}

.chat-col {
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* Disconnected banner */
.disconnected-banner {
  position: fixed;
  bottom: var(--space-6);
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-5);
  border-radius: var(--radius-full);
  font-size: 0.875rem;
  color: var(--color-accent-amber);
  border-color: hsla(38, 95%, 58%, 0.25) !important;
  background: rgba(14,14,28,0.9) !important;
  white-space: nowrap;
  z-index: var(--z-toast);
  box-shadow: var(--shadow-card);
}

/* Scrape error toast */
.scrape-error-toast {
  position: fixed;
  bottom: var(--space-6);
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-5);
  border-radius: var(--radius-lg);
  font-size: 0.875rem;
  color: var(--color-primary);
  border-color: hsla(330, 100%, 55%, 0.3) !important;
  background: rgba(14, 14, 28, 0.95) !important;
  z-index: var(--z-toast);
  box-shadow: var(--shadow-card), 0 0 20px hsla(330, 100%, 55%, 0.15);
  max-width: 500px;
}
.scrape-error-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.scrape-error-text strong {
  font-size: 0.8125rem;
  color: white;
}
.scrape-error-text span {
  font-size: 0.75rem;
  color: var(--text-muted);
  word-break: break-word;
}
.scrape-error-dismiss {
  background: none;
  border: none;
  color: rgba(255,255,255,0.4);
  font-size: 1.25rem;
  cursor: pointer;
  padding: 0 4px;
  transition: color var(--transition-fast);
}
.scrape-error-dismiss:hover {
  color: white;
}

/* Responsive */
@media (max-width: 900px) {
  .room-layout {
    grid-template-columns: 1fr;
    grid-template-rows: auto 1fr;
    max-height: none;
    overflow: auto;
  }
  .chat-col {
    height: 400px;
  }
}
</style>
