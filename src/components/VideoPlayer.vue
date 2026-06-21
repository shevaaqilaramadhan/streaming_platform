<template>
  <div class="video-wrapper" :class="{ 'is-loading': isLoading }">
    <!-- Loading overlay -->
    <Transition name="fade">
      <div v-if="isLoading && videoUrl" class="video-loading">
        <div class="spinner"></div>
        <span>Loading video…</span>
      </div>
    </Transition>

    <!-- No video placeholder -->
    <div v-if="!videoUrl" class="video-placeholder">
      <div class="placeholder-icon">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <polygon points="23 7 16 12 23 17 23 7"/>
          <rect x="1" y="5" width="15" height="14" rx="2" ry="2"/>
        </svg>
      </div>
      <p v-if="isHost" class="placeholder-text">Set a video URL below to start watching</p>
      <p v-else class="placeholder-text">Waiting for the host to start a video…</p>
    </div>

    <!-- Actual video element -->
    <video
      v-show="videoUrl"
      ref="videoRef"
      id="watch-party-video"
      class="video-el"
      :src="videoUrl || undefined"
      preload="metadata"
      playsinline
      @timeupdate="onTimeUpdate"
      @play="onPlay"
      @pause="onPause"
      @seeking="onSeeking"
      @loadstart="isLoading = true"
      @canplay="isLoading = false"
      @waiting="isLoading = true"
      @playing="isLoading = false"
      @loadedmetadata="onLoadedMetadata"
    ></video>

    <!-- Custom controls overlay -->
    <Transition name="fade">
      <div
        v-show="videoUrl"
        class="controls-overlay"
        :class="{ 'controls-visible': showControls }"
        @mouseenter="showControls = true"
        @mouseleave="hideControlsDelayed"
        @mousemove="onMouseMove"
        @click.self="togglePlay"
      >
        <!-- Big play/pause center button -->
        <Transition name="fade">
          <button
            v-if="showPlayPulse"
            class="center-play-btn"
            @click="togglePlay"
            aria-label="Toggle play/pause"
          >
            <svg v-if="!isPlaying" width="36" height="36" viewBox="0 0 24 24" fill="white">
              <polygon points="5 3 19 12 5 21 5 3"/>
            </svg>
            <svg v-else width="36" height="36" viewBox="0 0 24 24" fill="white">
              <rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/>
            </svg>
          </button>
        </Transition>

        <!-- Bottom controls bar -->
        <div class="controls-bar">
          <!-- Progress bar -->
          <div class="progress-container">
            <div
              class="progress-bar"
              role="slider"
              :aria-valuenow="currentTime"
              :aria-valuemax="duration"
              aria-label="Video progress"
              @click="isHost ? onSeek($event) : null"
              :class="{ 'is-interactive': isHost }"
            >
              <div class="progress-bg"></div>
              <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
              <div class="progress-thumb" :style="{ left: progressPercent + '%' }" v-if="isHost"></div>
            </div>
            <div class="time-display">
              <span>{{ formatTime(currentTime) }}</span>
              <span class="time-sep">/</span>
              <span>{{ formatTime(duration) }}</span>
            </div>
          </div>

          <!-- Buttons row -->
          <div class="controls-buttons">
            <!-- Play / Pause -->
            <button
              id="play-pause-btn"
              class="ctrl-btn"
              @click="togglePlay"
              :disabled="!isHost"
              :data-tooltip="isHost ? (isPlaying ? 'Pause' : 'Play') : 'Only the host can control playback'"
              :aria-label="isPlaying ? 'Pause' : 'Play'"
            >
              <svg v-if="!isPlaying" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <polygon points="5 3 19 12 5 21 5 3"/>
              </svg>
              <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/>
              </svg>
            </button>

            <!-- Volume -->
            <button
              id="volume-btn"
              class="ctrl-btn"
              @click="toggleMute"
              :data-tooltip="isMuted ? 'Unmute' : 'Mute'"
              :aria-label="isMuted ? 'Unmute' : 'Mute'"
            >
              <svg v-if="!isMuted" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
                <path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
                <path d="M19.07 4.93a10 10 0 0 1 0 14.14"/>
              </svg>
              <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
                <line x1="23" y1="9" x2="17" y2="15"/><line x1="17" y1="9" x2="23" y2="15"/>
              </svg>
            </button>

            <input
              id="volume-slider"
              type="range"
              class="volume-slider"
              min="0" max="1" step="0.05"
              v-model.number="volume"
              @input="onVolumeChange"
              aria-label="Volume"
            />

            <!-- Spacer -->
            <span style="flex:1"></span>

            <!-- Host indicator for guests -->
            <span v-if="!isHost" class="guest-indicator">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
              Watching
            </span>

            <!-- Fullscreen -->
            <button
              id="fullscreen-btn"
              class="ctrl-btn"
              @click="toggleFullscreen"
              data-tooltip="Fullscreen"
              aria-label="Toggle fullscreen"
            >
              <svg v-if="!isFullscreen" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="15 3 21 3 21 9"/>
                <polyline points="9 21 3 21 3 15"/>
                <line x1="21" y1="3" x2="14" y2="10"/>
                <line x1="3" y1="21" x2="10" y2="14"/>
              </svg>
              <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="4 14 10 14 10 20"/>
                <polyline points="20 10 14 10 14 4"/>
                <line x1="10" y1="14" x2="3" y2="21"/>
                <line x1="21" y1="3" x2="14" y2="10"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Host: Set Video URL panel -->
    <div v-if="isHost" class="video-url-panel glass">
      <input
        id="video-url-input"
        v-model="newVideoUrl"
        type="url"
        class="input"
        placeholder="Paste a direct .mp4 video URL…"
        @keydown.enter="submitVideoUrl"
      />
      <button
        id="set-video-btn"
        class="btn btn-primary btn-sm"
        :disabled="!newVideoUrl.trim()"
        @click="submitVideoUrl"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><polygon points="10 8 16 12 10 16 10 8"/>
        </svg>
        Load Video
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  isHost:      { type: Boolean, default: false },
  videoUrl:    { type: String,  default: '' },
  playerState: { type: Object,  default: () => ({ isPlaying: false, currentTime: 0 }) },
})

const emit = defineEmits(['sync', 'set-video'])

/* ---- Refs ---- */
const videoRef   = ref(null)
const isPlaying  = ref(false)
const isLoading  = ref(false)
const isMuted    = ref(false)
const isFullscreen = ref(false)
const volume     = ref(1)
const currentTime = ref(0)
const duration   = ref(0)
const showControls = ref(true)
const showPlayPulse = ref(false)
const newVideoUrl  = ref('')
const pendingSeekTime = ref(null)
let controlsTimer = null
let isSeeking = false
let lastSyncTime = 0

/* ---- Computed ---- */
const progressPercent = computed(() => {
  if (!duration.value) return 0
  return (currentTime.value / duration.value) * 100
})

/* ---- Watch incoming player state from WebSocket (guests only) ---- */
watch(
  () => props.playerState,
  (state) => {
    if (!videoRef.value || props.isHost) return
    const vid = videoRef.value

    // Sync time if drift > 1s
    if (Math.abs(vid.currentTime - state.currentTime) > 1) {
      if (vid.readyState >= 1) { // HAVE_METADATA or higher
        vid.currentTime = state.currentTime
      } else {
        // Delay seek until metadata is loaded
        pendingSeekTime.value = state.currentTime
      }
    }

    if (state.isPlaying && vid.paused) {
      vid.play().catch(() => {})
    } else if (!state.isPlaying && !vid.paused) {
      vid.pause()
    }
  },
  { deep: true }
)

watch(
  () => props.videoUrl,
  () => {
    pendingSeekTime.value = null
  }
)

/* ---- Video Event Handlers ---- */
function onLoadedMetadata() {
  if (videoRef.value && pendingSeekTime.value !== null) {
    videoRef.value.currentTime = pendingSeekTime.value
    pendingSeekTime.value = null
  }
}

function onTimeUpdate() {
  if (!videoRef.value) return
  currentTime.value = videoRef.value.currentTime
  duration.value    = videoRef.value.duration || 0

  // Throttle sync events to avoid flooding
  if (props.isHost && Date.now() - lastSyncTime > 2000) {
    lastSyncTime = Date.now()
    // Only emit if playing (seeking emits its own)
    if (!videoRef.value.paused) {
      emit('sync', { isPlaying: true, currentTime: videoRef.value.currentTime })
    }
  }
}

function onPlay() {
  isPlaying.value = true
  if (props.isHost) emit('sync', { isPlaying: true, currentTime: videoRef.value?.currentTime || 0 })
}

function onPause() {
  isPlaying.value = false
  if (props.isHost) emit('sync', { isPlaying: false, currentTime: videoRef.value?.currentTime || 0 })
}

function onSeeking() {
  if (props.isHost) {
    emit('sync', { isPlaying: !videoRef.value?.paused, currentTime: videoRef.value?.currentTime || 0 })
  }
}

/* ---- Controls ---- */
function togglePlay() {
  if (!props.isHost || !videoRef.value) return
  if (videoRef.value.paused) {
    videoRef.value.play().catch(() => {})
  } else {
    videoRef.value.pause()
  }
  flashPlayPulse()
}

function flashPlayPulse() {
  showPlayPulse.value = true
  setTimeout(() => { showPlayPulse.value = false }, 600)
}

function toggleMute() {
  if (!videoRef.value) return
  isMuted.value = !isMuted.value
  videoRef.value.muted = isMuted.value
}

function onVolumeChange() {
  if (!videoRef.value) return
  videoRef.value.volume = volume.value
  isMuted.value = volume.value === 0
}

function onSeek(event) {
  if (!videoRef.value || !props.isHost) return
  const bar = event.currentTarget
  const rect = bar.getBoundingClientRect()
  const ratio = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width))
  videoRef.value.currentTime = ratio * duration.value
}

function toggleFullscreen() {
  const el = videoRef.value?.parentElement
  if (!el) return
  if (!document.fullscreenElement) {
    el.requestFullscreen()
    isFullscreen.value = true
  } else {
    document.exitFullscreen()
    isFullscreen.value = false
  }
}

function onMouseMove() {
  showControls.value = true
  clearTimeout(controlsTimer)
  hideControlsDelayed()
}

function hideControlsDelayed() {
  controlsTimer = setTimeout(() => {
    if (isPlaying.value) showControls.value = false
  }, 3000)
}

function submitVideoUrl() {
  const url = newVideoUrl.value.trim()
  if (!url) return
  emit('set-video', url)
  newVideoUrl.value = ''
}

/* ---- Format time helper ---- */
function formatTime(secs) {
  if (!secs || isNaN(secs)) return '0:00'
  const m = Math.floor(secs / 60)
  const s = Math.floor(secs % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

onUnmounted(() => clearTimeout(controlsTimer))
</script>

<style scoped>
.video-wrapper {
  position: relative;
  width: 100%;
  background: #000;
  border-radius: var(--radius-lg);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-card), var(--shadow-glow-sm);
}

.video-el {
  width: 100%;
  aspect-ratio: 16 / 9;
  display: block;
  object-fit: contain;
  background: #000;
}

/* Placeholder */
.video-placeholder {
  width: 100%;
  aspect-ratio: 16 / 9;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-4);
  background: radial-gradient(ellipse at center, rgba(80,60,140,0.15) 0%, #000 70%);
}
.placeholder-icon {
  color: rgba(255,255,255,0.15);
  animation: float 4s ease-in-out infinite;
}
.placeholder-text {
  color: var(--text-muted);
  font-size: 0.9375rem;
  text-align: center;
  padding: 0 var(--space-4);
}

/* Loading */
.video-loading {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  background: rgba(0,0,0,0.6);
  color: var(--text-secondary);
  font-size: 0.875rem;
  z-index: 2;
  pointer-events: none;
}
.spinner {
  width: 32px; height: 32px;
  border: 3px solid rgba(255,255,255,0.1);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin-slow 0.8s linear infinite;
}

/* Controls overlay */
.controls-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  background: linear-gradient(to top, rgba(0,0,0,0.75) 0%, transparent 40%);
  opacity: 0;
  transition: opacity var(--transition-base);
}
.controls-visible { opacity: 1; }

.center-play-btn {
  position: absolute;
  top: 50%; left: 50%;
  transform: translate(-50%, -50%);
  width: 72px; height: 72px;
  border-radius: 50%;
  background: rgba(0,0,0,0.6);
  border: 2px solid rgba(255,255,255,0.3);
  display: flex; align-items: center; justify-content: center;
  cursor: pointer;
  transition: all var(--transition-base);
}
.center-play-btn:hover { background: rgba(0,0,0,0.8); transform: translate(-50%,-50%) scale(1.08); }

.controls-bar {
  padding: 0 var(--space-4) var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

/* Progress bar */
.progress-container {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.progress-bar {
  position: relative;
  height: 4px;
  border-radius: var(--radius-full);
  overflow: visible;
  cursor: default;
}
.progress-bar.is-interactive { cursor: pointer; }
.progress-bar.is-interactive:hover .progress-fill { height: 6px; }
.progress-bar.is-interactive:hover { height: 6px; }
.progress-bg {
  position: absolute; inset: 0;
  background: rgba(255,255,255,0.2);
  border-radius: var(--radius-full);
}
.progress-fill {
  position: absolute; top: 0; left: 0; height: 100%;
  background: var(--grad-brand);
  border-radius: var(--radius-full);
  transition: width 0.1s linear;
  pointer-events: none;
}
.progress-thumb {
  position: absolute;
  top: 50%; transform: translate(-50%, -50%);
  width: 14px; height: 14px;
  background: white;
  border-radius: 50%;
  box-shadow: 0 0 6px rgba(0,0,0,0.5);
  transition: left 0.1s linear;
  pointer-events: none;
}

.time-display {
  font-size: 0.75rem;
  color: rgba(255,255,255,0.7);
  font-variant-numeric: tabular-nums;
  display: flex;
  gap: 3px;
}
.time-sep { color: rgba(255,255,255,0.3); }

/* Buttons row */
.controls-buttons {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.ctrl-btn {
  width: 36px; height: 36px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: none;
  color: rgba(255,255,255,0.85);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}
.ctrl-btn:hover { background: rgba(255,255,255,0.1); color: white; }
.ctrl-btn:disabled { opacity: 0.4; cursor: not-allowed; }

.volume-slider {
  width: 80px;
  accent-color: var(--color-primary);
  cursor: pointer;
}

.guest-indicator {
  display: flex; align-items: center; gap: 4px;
  font-size: 0.75rem;
  color: rgba(255,255,255,0.5);
}

/* URL panel */
.video-url-panel {
  display: flex;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-top: 1px solid var(--color-glass-border);
  border-radius: 0 0 var(--radius-lg) var(--radius-lg);
}
.video-url-panel .input {
  flex: 1;
  border-radius: var(--radius-sm);
  padding: var(--space-2) var(--space-3);
  font-size: 0.875rem;
}
</style>
