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
        :is-public="isPublic"
        :room-name="roomName"
        @toggle-public="onTogglePublic"
        @toggle-user-list="showUserList = !showUserList"
        @set-room-name="onSetRoomName"
      />

      <!-- User List Panel -->
      <UserListPanel
        :visible="showUserList"
        :participants="participants"
        :host-id="hostId"
        :local-user-id="localUserId"
        :is-host="isHost"
        @close="showUserList = false"
        @transfer-host="onTransferHost"
        @kick-user="onKickUser"
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
            @ended="onVideoEnded"
            @error="onStreamError"
          />

          <!-- Single host URL input (VideoPlayer no longer duplicates this) -->
          <div v-if="isHost" class="room-set-video glass">
            <input
              id="room-video-url-input"
              v-model="setVideoUrl"
              type="url"
              class="input"
              placeholder="YouTube, .m3u8, .mp4, or anime page URL…"
              @keydown.enter="submitRoomVideoUrl"
            />
            <button
              id="set-video-btn"
              class="btn btn-primary btn-sm"
              :disabled="!setVideoUrl.trim()"
              @click="submitRoomVideoUrl"
            >
              Load Video
            </button>
          </div>

          <!-- Emoji Reactions overlay -->
          <EmojiReactions
            :reactions="reactions"
            @react="onReaction"
          />
        </section>

        <!-- Sidebar column (Chat & Queue) -->
        <aside class="sidebar-col" aria-label="Sidebar panel">
          <!-- Sidebar Tabs -->
          <div class="sidebar-tabs glass">
            <button 
              class="tab-btn" 
              :class="{ 'tab-btn--active': activeTab === 'chat' }"
              @click="activeTab = 'chat'"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
              Chat
              <span v-if="messages.length > 0" class="tab-badge">{{ messages.length }}</span>
            </button>
            <button 
              class="tab-btn" 
              :class="{ 'tab-btn--active': activeTab === 'queue' }"
              @click="activeTab = 'queue'"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="8" y1="6" x2="21" y2="6"/>
                <line x1="8" y1="12" x2="21" y2="12"/>
                <line x1="8" y1="18" x2="21" y2="18"/>
                <line x1="3" y1="6" x2="3.01" y2="6"/>
                <line x1="3" y1="12" x2="3.01" y2="12"/>
                <line x1="3" y1="18" x2="3.01" y2="18"/>
              </svg>
              Queue
              <span v-if="queue.length > 0" class="tab-badge tab-badge--primary">{{ queue.length }}</span>
            </button>
          </div>

          <!-- Panel Containers -->
          <div class="panel-container">
            <Transition name="fade" mode="out-in">
              <ChatPanel
                v-if="activeTab === 'chat'"
                :messages="messages"
                :is-connected="isConnected"
                :nickname="nickname"
                :typing-users="typingUsers"
                @send="onSendChat"
                @typing="onTyping"
              />
              <QueuePanel
                v-else
                :queue="queue"
                :is-host="isHost"
                @add="onAddToQueue"
                @remove="onRemoveFromQueue"
                @skip="onSkipToNext"
                @clear="onClearQueue"
              />
            </Transition>
          </div>
        </aside>
      </div>
    </template>

    <!-- Disconnected / Reconnecting banner -->
    <Transition name="slide-up">
      <div
        v-if="hasJoined && showReconnectBanner"
        class="disconnected-banner glass"
        role="alert"
        :class="{ 'disconnected-banner--offline': wsStatus === 'disconnected' }"
      >
        <div class="banner-spinner"></div>
        <span v-if="wsStatus === 'reconnecting'">Connection lost, reconnecting…</span>
        <span v-else-if="wsStatus === 'connecting'">Connecting to room…</span>
        <span v-else>Lost connection to the room. Attempting to reconnect…</span>
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

    <!-- Scrape loading banner -->
    <Transition name="slide-up">
      <div v-if="scrapeLoading" class="scrape-loading-banner glass" role="status">
        <div class="banner-spinner"></div>
        <div class="scrape-loading-text">
          <strong>Extracting stream…</strong>
          <span>This may take up to a minute for some sites</span>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import RoomHeader   from '../components/RoomHeader.vue'
import VideoPlayer  from '../components/VideoPlayer.vue'
import ChatPanel    from '../components/ChatPanel.vue'
import QueuePanel   from '../components/QueuePanel.vue'
import NicknameModal from '../components/NicknameModal.vue'
import UserListPanel from '../components/UserListPanel.vue'
import EmojiReactions from '../components/EmojiReactions.vue'

import { useRoom } from '../composables/useRoom.js'
import { isValidVideoInput } from '../utils/videoInput.js'

const route   = useRoute()
const router  = useRouter()
const roomId  = computed(() => route.params.roomId)

function hostTokenStorageKey(id) {
  return `wp_host_${id}`
}

function peekHostToken(id) {
  if (!id || typeof sessionStorage === 'undefined') return ''
  return sessionStorage.getItem(hostTokenStorageKey(id)) || ''
}

function clearHostToken(id) {
  if (!id || typeof sessionStorage === 'undefined') return
  sessionStorage.removeItem(hostTokenStorageKey(id))
}

/* ---- Local State ---- */
const nickname   = ref('')
const showModal  = ref(true)
const hasJoined  = ref(false)
const isHost     = ref(false)
let pendingHostToken = ''

/* ---- Room composable ---- */
let room = null
let roomWatchers = []

const wsStatus        = ref('disconnected')
const messages        = ref([])
const playerState     = ref({ videoUrl: '', currentTime: 0, isPlaying: false })
const participants    = ref([])
const scrapeError     = ref(null)
const scrapeLoading   = ref(false)
const currentMetadata = ref(null)
const queue           = ref([])
const isPublic        = ref(false)
const roomName        = ref('')
const typingUsers     = ref([])
const reactions       = ref([])
const activeTab       = ref('chat')
const showUserList    = ref(false)
const hostId          = ref('')
const localUserId     = ref('')
const setVideoUrl     = ref('')
const isConnected     = computed(() => wsStatus.value === 'connected')
const showReconnectBanner = computed(() =>
  ['disconnected', 'reconnecting', 'connecting'].includes(wsStatus.value)
)

function initRoom() {
  const token = pendingHostToken || peekHostToken(roomId.value)
  pendingHostToken = token
  room = useRoom(roomId.value, nickname.value, token)

  // 1. Setup watchers FIRST (before any state changes)
  roomWatchers.push(
    watch(room.status,          v => { wsStatus.value    = v }),
    watch(room.messages,        v => { messages.value    = v }, { deep: true }),
    watch(room.playerState,     v => { playerState.value = v }, { deep: true }),
    watch(room.participants,    v => { participants.value = v }, { deep: true }),
    watch(room.scrapeError,     v => { scrapeError.value  = v }),
    watch(room.scrapeLoading,   v => { scrapeLoading.value = v }),
    watch(room.currentMetadata, v => { currentMetadata.value = v }, { deep: true }),
    watch(room.queue,           v => { queue.value = v }, { deep: true }),
    watch(room.isPublic,        v => { isPublic.value = v }),
    watch(room.isHost,          v => {
      isHost.value = v
      // Clear host token only after server confirms we are host (single-use claim done)
      if (v && roomId.value) clearHostToken(roomId.value)
    }),
    watch(room.hostId,          v => { hostId.value = v }),
    watch(room.localUserId,     v => { localUserId.value = v }),
    watch(room.roomName,        v => { roomName.value = v }),
    watch(room.typingUsers,     v => { typingUsers.value = v }, { deep: true }),
    watch(room.reactions,       v => { reactions.value = v }, { deep: true }),
  )

  // 2. THEN sync initial values (any changes after this point are caught by watchers)
  wsStatus.value        = room.status.value
  messages.value        = room.messages.value
  playerState.value     = room.playerState.value
  participants.value    = room.participants.value
  currentMetadata.value = room.currentMetadata.value
  queue.value           = room.queue.value
  isPublic.value        = room.isPublic.value
  isHost.value          = room.isHost.value
  hostId.value          = room.hostId.value
  localUserId.value     = room.localUserId.value
  roomName.value        = room.roomName.value

  // Handle kicked: WS already permanently closed in useRoom; just navigate home
  room.onKicked((message) => {
    try {
      sessionStorage.setItem('wp_flash', JSON.stringify({
        type: 'kicked',
        message: message || 'You have been removed from the room by the host.',
      }))
    } catch { /* ignore */ }
    // leave() is safe (idempotent permanent disconnect)
    try { room?.leave() } catch { /* ignore */ }
    router.replace({ name: 'home', query: { kicked: '1' } })
  })

  // 3. LAST: connect
  room.joinRoom()
}

/* ---- Lifecycle ---- */
onMounted(() => {
  pendingHostToken = peekHostToken(roomId.value)
})

onUnmounted(() => {
  roomWatchers.forEach(stop => stop())
  roomWatchers = []
  if (room) {
    room.leave()
    room = null
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

function submitRoomVideoUrl() {
  const url = setVideoUrl.value.trim()
  if (!url || !isHost.value) return
  if (!isValidVideoInput(url)) {
    scrapeError.value = {
      originalUrl: url,
      error: 'Invalid URL. Use YouTube, a direct stream (.mp4/.m3u8), or a supported anime page link.',
    }
    return
  }
  scrapeError.value = null
  room?.setVideo(url)
  setVideoUrl.value = ''
}

function onSendChat(text) {
  room?.sendChatMessage(text)
}

function onAddToQueue(url) {
  room?.addToQueue(url)
}

function onRemoveFromQueue(itemId) {
  room?.removeFromQueue(itemId)
}

function onSkipToNext() {
  room?.skipToNext()
}

function onClearQueue() {
  room?.clearQueue()
}

function onTogglePublic(val) {
  room?.togglePublic(val)
}

function onVideoEnded() {
  room?.onVideoEnded()
}

function onStreamError(msg) {
  console.error('[Stream]', msg)
}

function onTransferHost(targetUserId) {
  room?.transferHost(targetUserId)
  showUserList.value = false
}

function onKickUser(targetUserId) {
  if (confirm('Are you sure you want to kick this user?')) {
    room?.kickUser(targetUserId)
    showUserList.value = false
  }
}

function onSetRoomName(name) {
  room?.setRoomName(name)
}

function onReaction(emoji) {
  room?.sendReaction(emoji)
}

function onTyping() {
  room?.sendTyping()
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
  height: calc(100vh - 72px);
  max-height: calc(100vh - 72px);
  overflow: hidden;
}

.video-col {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  overflow-y: auto;
  position: relative;
}

.room-set-video {
  display: flex;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-lg);
  align-items: center;
  flex-shrink: 0;
}
.room-set-video .input {
  flex: 1;
  min-width: 0;
}
@media (max-width: 480px) {
  .room-set-video {
    flex-direction: column;
    align-items: stretch;
  }
  .room-set-video .btn {
    width: 100%;
  }
}

/* Now Watching Header Styles */
.now-watching-header {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4);
  border-radius: var(--radius-lg);
  border-left: 2px solid rgba(255,255,255,0.2);
  box-shadow: var(--shadow-card);
}

.anime-thumbnail {
  width: 60px;
  height: 80px;
  object-fit: cover;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
}

.metadata-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.watching-label {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--text-muted);
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

.sidebar-col {
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.sidebar-tabs {
  display: flex;
  padding: 4px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255,255,255,0.05);
  flex-shrink: 0;
}

.tab-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tab-btn:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.03);
}

.tab-btn--active {
  color: var(--text-primary);
  background: var(--color-fill-hover);
  border: 1px solid var(--color-border);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.tab-badge {
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.05);
  border-radius: var(--radius-full);
  padding: 1px 6px;
  margin-left: 2px;
}

.tab-badge--primary {
  color: var(--text-inverse);
  background: var(--text-primary);
}

.panel-container {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  position: relative;
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
  border-color: rgba(240,173,78,0.2) !important;
  background: rgba(20,20,22,0.92) !important;
  white-space: nowrap;
  z-index: var(--z-toast);
  box-shadow: var(--shadow-card);
}
.banner-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(240,173,78,0.15);
  border-top-color: var(--color-accent-amber);
  border-radius: 50%;
  flex-shrink: 0;
  animation: spin-slow 0.7s linear infinite;
}
.disconnected-banner--offline {
  color: var(--color-accent-red);
  border-color: rgba(217,83,79,0.2) !important;
}
.disconnected-banner--offline .banner-spinner {
  border-color: rgba(217,83,79,0.15);
  border-top-color: var(--color-accent-red);
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
  color: var(--text-secondary);
  border-color: rgba(217,83,79,0.2) !important;
  background: rgba(20,20,22,0.95) !important;
  z-index: var(--z-toast);
  box-shadow: var(--shadow-card);
  max-width: 500px;
}
.scrape-error-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.scrape-error-text strong {
  font-size: 0.8125rem;
  color: var(--text-primary);
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

/* Scrape loading banner */
.scrape-loading-banner {
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
  color: var(--text-secondary);
  border-color: rgba(255,255,255,0.08) !important;
  background: rgba(20,20,22,0.95) !important;
  z-index: var(--z-toast);
  box-shadow: var(--shadow-card);
  white-space: nowrap;
}
.scrape-loading-banner .banner-spinner {
  border-color: rgba(255,255,255,0.1);
  border-top-color: var(--text-secondary);
}
.scrape-loading-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.scrape-loading-text strong {
  font-size: 0.8125rem;
  color: var(--text-primary);
}
.scrape-loading-text span {
  font-size: 0.75rem;
  color: var(--text-muted);
}

/* Responsive */
@media (max-width: 900px) {
  .room-layout {
    grid-template-columns: 1fr;
    grid-template-rows: auto 1fr;
    height: auto;
    max-height: none;
    overflow: auto;
    padding: var(--space-4);
    gap: var(--space-4);
  }
  .video-col {
    overflow-y: visible;
  }
  .sidebar-col {
    height: 480px;
  }
}

@media (max-width: 480px) {
  .room-layout {
    padding: var(--space-3);
    gap: var(--space-3);
  }
  .now-watching-header {
    padding: var(--space-3);
    gap: var(--space-3);
  }
  .anime-thumbnail {
    width: 44px;
    height: 60px;
  }
  .anime-title {
    font-size: 1rem;
  }
  .sidebar-col {
    height: 420px;
  }
  .disconnected-banner {
    left: var(--space-3);
    right: var(--space-3);
    transform: none;
    width: auto;
    white-space: normal;
    font-size: 0.8125rem;
    padding: var(--space-3) var(--space-4);
  }
  .scrape-error-toast {
    left: var(--space-3);
    right: var(--space-3);
    transform: none;
    max-width: none;
  }
  .scrape-loading-banner {
    left: var(--space-3);
    right: var(--space-3);
    transform: none;
    white-space: normal;
    font-size: 0.8125rem;
    padding: var(--space-3) var(--space-4);
  }
}

@media (max-width: 375px) {
  .sidebar-col {
    height: 380px;
  }
  .room-layout {
    padding: var(--space-2);
  }
}
</style>
