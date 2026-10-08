<template>
  <div ref="wrapperRef" class="video-wrapper" :class="{ 'is-loading': isLoading, 'is-iframe': videoMode === 'iframe' }">
    <!-- Loading overlay -->
    <Transition name="fade">
      <div v-if="isLoading && videoUrl" class="video-loading">
        <div class="spinner"></div>
        <span>Loading video…</span>
      </div>
    </Transition>

    <!-- Error overlay (playback failure or invalid URL input) -->
    <Transition name="fade">
      <div v-if="streamError" class="video-error">
        <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
        <p class="error-title">{{ videoUrl ? 'Unable to play video' : 'Invalid video URL' }}</p>
        <p class="error-detail">{{ streamError }}</p>
        <button v-if="isHost" class="btn btn-primary btn-sm error-cta" @click="clearErrorAndFocus">
          Try another URL
        </button>
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
      <div v-if="isHost" class="placeholder-content">
        <p class="placeholder-text">Paste a URL to start watching</p>
        <p class="placeholder-hint">Supports YouTube, direct .mp4/.m3u8 links, and anime page URLs</p>
      </div>
      <div v-else class="placeholder-content">
        <p class="placeholder-text">Waiting for the host to start a video…</p>
        <p class="placeholder-hint">Sit tight — the host will pick something to watch</p>
      </div>
    </div>

    <!-- YouTube iframe container -->
    <div v-show="videoUrl && videoMode === 'youtube'" class="player-container">
      <div id="youtube-player"></div>
    </div>

    <!-- Generic Embed iframe container (Blogger, Mega, xtwap, etc.) -->
    <div v-if="videoUrl && videoMode === 'iframe'" class="player-container iframe-container">
      <iframe
        ref="iframeRef"
        :src="videoUrl"
        class="embed-iframe"
        allow="autoplay; fullscreen; encrypted-media; picture-in-picture"
        referrerpolicy="no-referrer"
        @load="onIframeLoad"
      ></iframe>
    </div>

    <!-- Native HTML5 video element (for HLS & MP4) -->
    <video
      v-show="videoUrl && videoMode !== 'youtube' && videoMode !== 'iframe'"
      ref="videoRef"
      id="watch-party-video"
      class="video-el"
      preload="auto"
      playsinline
      referrerpolicy="no-referrer"
      @timeupdate="onNativeTimeUpdate"
      @play="onNativePlay"
      @pause="onNativePause"
      @seeking="onNativeSeeking"
      @loadstart="isLoading = true"
      @canplay="isLoading = false"
      @waiting="isLoading = true"
      @playing="isLoading = false"
      @loadedmetadata="onNativeLoadedMetadata"
      @ended="onNativeEnded"
      @error="onNativeError"
    ></video>

    <!-- Custom controls overlay (HTML5 and HLS only; iframe embeds render their own player controls) -->
    <Transition name="fade">
      <div
        v-show="videoUrl && videoMode !== 'iframe'"
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
            v-if="(!isPlaying && videoUrl && !isLoading) || showPlayPulse"
            class="center-play-btn"
            @click="togglePlay"
            :disabled="!isHost"
            :data-tooltip="isHost ? null : 'Only the host can control playback'"
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
              @pointerdown="isInteracting = true"
              @pointerup="isInteracting = false"
              aria-label="Volume"
            />

            <!-- Spacer -->
            <span style="flex:1"></span>

            <!-- Stream mode badge -->
            <span v-if="videoMode" class="stream-badge" :data-tooltip="streamTooltip">
              {{ videoMode === 'youtube' ? 'YT' : videoMode === 'hls' ? 'HLS' : 'MP4' }}
            </span>

            <!-- Host indicator for guests -->
            <span v-if="!isHost" class="guest-indicator">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
              Watching
            </span>

            <!-- Next Episode Quick Button (Host only) -->
            <button
              v-if="isHost && nextEpisodeUrl"
              id="next-episode-btn"
              class="ctrl-btn ctrl-btn--next-ep"
              @click="$emit('play-next-episode', nextEpisodeUrl)"
              data-tooltip="Play Next Episode ▶"
              aria-label="Play Next Episode"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="5 4 15 12 5 20 5 4"/>
                <line x1="19" y1="5" x2="19" y2="19"/>
              </svg>
            </button>

            <!-- Picture-in-Picture -->
            <button
              v-if="supportsPiP"
              id="pip-btn"
              class="ctrl-btn"
              :class="{ 'ctrl-btn--active': isPiP }"
              @click="togglePiP"
              :data-tooltip="isPiP ? 'Exit PiP (P)' : 'Picture-in-Picture (P)'"
              aria-label="Toggle Picture-in-Picture"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                <rect x="11" y="9" width="9" height="6" rx="1" ry="1" :fill="isPiP ? 'currentColor' : 'none'"/>
              </svg>
            </button>

            <!-- Fullscreen Chat Toggle -->
            <button
              v-if="isFullscreen"
              id="fs-chat-toggle-btn"
              class="ctrl-btn"
              :class="{ 'ctrl-btn--active': showFsChat }"
              @click="showFsChat = !showFsChat"
              :data-tooltip="showFsChat ? 'Hide Chat Overlay (C)' : 'Show Chat Overlay (C)'"
              aria-label="Toggle Fullscreen Chat"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
            </button>

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

    <!-- Fullscreen Floating Chat Overlay -->
    <Transition name="fade">
      <div 
        v-if="isFullscreen && showFsChat" 
        class="fs-chat-overlay" 
        @click.stop
        @pointerdown.stop
      >
        <div class="fs-chat-header">
          <div class="fs-chat-title">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
            <span>Live Chat</span>
          </div>
          <button class="fs-chat-close-btn" @click="showFsChat = false" title="Close (C)">×</button>
        </div>
        <div ref="fsChatScrollRef" class="fs-chat-messages">
          <div v-if="messages.length === 0" class="fs-chat-empty">
            No messages yet. Say hello!
          </div>
          <div 
            v-for="msg in messages.slice(-40)" 
            :key="msg.id || msg.sentAt" 
            class="fs-chat-msg"
            :class="{ 'fs-chat-msg--system': msg.isSystem || msg.type === 'system' }"
          >
            <span v-if="!msg.isSystem && msg.type !== 'system'" class="fs-chat-username" :class="{ 'fs-chat-username--host': msg.isHost }">
              {{ msg.username }}:
            </span>
            <span class="fs-chat-text">{{ msg.text }}</span>
          </div>
        </div>
        <form class="fs-chat-input-wrap" @submit.prevent="submitFsChat">
          <input
            v-model="fsChatText"
            type="text"
            class="fs-chat-input"
            placeholder="Ketik pesan (Enter)..."
            maxlength="300"
            @keydown.stop
          />
          <button type="submit" class="fs-chat-send-btn" :disabled="!fsChatText.trim()">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="22" y1="2" x2="11" y2="13"/>
              <polygon points="22 2 15 22 11 13 2 9 22 2"/>
            </svg>
          </button>
        </form>
      </div>
    </Transition>

    <!-- URL input lives on RoomPage (single host input — avoid double fields) -->
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import Hls from 'hls.js'
import { API_BASE } from '../config.js'

const props = defineProps({
  isHost:         { type: Boolean, default: false },
  videoUrl:       { type: String,  default: '' },
  playerState:    { type: Object,  default: () => ({ isPlaying: false, currentTime: 0 }) },
  messages:       { type: Array,   default: () => [] },
  nextEpisodeUrl: { type: String,  default: '' },
})

const emit = defineEmits(['sync', 'ended', 'error', 'send-chat', 'play-next-episode'])

/* ================================================================
   State
   ================================================================ */
const wrapperRef   = ref(null)
const videoRef     = ref(null)
const isPlaying    = ref(false)
const isLoading    = ref(false)
const isMuted      = ref(false)
const isFullscreen = ref(false)
const volume       = ref(1)
const currentTime  = ref(0)
const duration     = ref(0)
const showControls = ref(true)
const showPlayPulse = ref(false)
const pendingSeekTime = ref(null)
const videoMode    = ref(null)  // 'youtube' | 'hls' | 'native'
const streamError  = ref(null)
const isInteracting = ref(false)

// PiP state
const isPiP = ref(false)
const supportsPiP = computed(() => {
  return typeof document !== 'undefined' &&
         'pictureInPictureEnabled' in document &&
         document.pictureInPictureEnabled &&
         videoMode.value !== 'youtube'
})

// Fullscreen Chat Overlay state
const showFsChat = ref(true)
const fsChatText = ref('')
const fsChatScrollRef = ref(null)

function submitFsChat() {
  const text = fsChatText.value.trim()
  if (!text) return
  emit('send-chat', text)
  fsChatText.value = ''
  nextTick(() => {
    if (fsChatScrollRef.value) {
      fsChatScrollRef.value.scrollTop = fsChatScrollRef.value.scrollHeight
    }
  })
}

async function togglePiP() {
  if (!videoRef.value || videoMode.value === 'youtube') return
  try {
    if (document.pictureInPictureElement) {
      await document.exitPictureInPicture()
    } else if (videoRef.value.requestPictureInPicture) {
      await videoRef.value.requestPictureInPicture()
    }
  } catch (err) {
    console.warn('[PiP] Error toggling Picture-in-Picture:', err)
  }
}

let ytPlayer = null         // YouTube IFrame player instance
let hlsInstance = null       // hls.js instance
let controlsTimer = null
let lastSyncTime = 0
let timeUpdateTimer = null
let ignoreStateChange = false

/* ================================================================
   Computed
   ================================================================ */
const progressPercent = computed(() => {
  if (!duration.value) return 0
  return (currentTime.value / duration.value) * 100
})

const streamTooltip = computed(() => {
  if (videoMode.value === 'youtube') return 'YouTube IFrame Player'
  if (videoMode.value === 'hls') return 'HLS stream via hls.js'
  if (videoMode.value === 'iframe') return 'Embedded Player'
  return 'Native HTML5 video'
})

/* ================================================================
   URL Detection Helpers
   ================================================================ */
function extractYoutubeId(url) {
  if (!url) return null
  const trimmed = url.trim()
  if (trimmed.length === 11 && /^[a-zA-Z0-9_-]{11}$/.test(trimmed)) return trimmed
  const regExp = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|&v=)([^#&?]*).*/
  const match = trimmed.match(regExp)
  return (match && match[2].length === 11) ? match[2] : null
}

function isHlsUrl(url) {
  if (!url) return false
  const clean = url.split('?')[0].split('#')[0]
  return clean.endsWith('.m3u8')
}

function isEmbedUrl(url) {
  if (!url) return false
  const lower = String(url).toLowerCase()
  return lower.includes('blogger.com/video') ||
    lower.includes('blogspot.com/video') ||
    lower.includes('mega.nz/embed') ||
    lower.includes('play.xtwap.top') ||
    lower.includes('/embed/') ||
    lower.includes('embed.php') ||
    lower.includes('player.php')
}

function isLikelyHtmlPageUrl(url) {
  if (!url) return false
  try {
    const u = new URL(url)
    const path = u.pathname.toLowerCase()
    // Embed URLs are handled via iframe player
    if (isEmbedUrl(url)) return false
    // Anime episode pages are not playable media
    if (/\.(mp4|m3u8|webm|mkv|ogg)$/i.test(path)) return false
    if (u.hostname.includes('youtube') || u.hostname.includes('youtu.be')) return false
    if (u.hostname.includes('googlevideo') || path.includes('videoplayback')) return false
    // path looks like a site page (no media extension)
    return !path.includes('.') || path.endsWith('/') || path.endsWith('.html') || path.endsWith('.php')
  } catch {
    return false
  }
}

function detectMode(url) {
  if (!url) return null
  if (extractYoutubeId(url)) return 'youtube'
  if (isHlsUrl(url)) return 'hls'
  if (isEmbedUrl(url)) return 'iframe'
  // Refuse to init native player on scraped anime page URLs (HTML)
  if (isLikelyHtmlPageUrl(url)) return null
  return 'native'  // .mp4 or other direct media
}

/* ================================================================
   YouTube IFrame Player API
   ================================================================ */
let apiLoadedPromise = null
function loadYoutubeApi() {
  if (apiLoadedPromise) return apiLoadedPromise
  apiLoadedPromise = new Promise((resolve) => {
    if (window.YT && window.YT.Player) { resolve(window.YT); return }
    const prev = window.onYouTubeIframeAPIReady
    window.onYouTubeIframeAPIReady = () => { if (prev) prev(); resolve(window.YT) }
    const tag = document.createElement('script')
    tag.src = 'https://www.youtube.com/iframe_api'
    const first = document.getElementsByTagName('script')[0]
    first.parentNode.insertBefore(tag, first)
  })
  return apiLoadedPromise
}

async function initYoutube(videoId) {
  isLoading.value = true
  const YT = await loadYoutubeApi()

  let target = document.getElementById('youtube-player')
  if (!target) {
    const container = document.querySelector('.player-container')
    if (container) {
      target = document.createElement('div')
      target.id = 'youtube-player'
      container.appendChild(target)
    }
  }

  if (ytPlayer) {
    try { ytPlayer.destroy() } catch (e) { console.warn('[YT] destroy error:', e) }
    ytPlayer = null
  }

  ytPlayer = new YT.Player('youtube-player', {
    height: '100%', width: '100%',
    videoId,
    playerVars: { autoplay: 0, controls: 0, rel: 0, modestbranding: 1, fs: 0, disablekb: 1, iv_load_policy: 3 },
    events: { onReady: onYtReady, onStateChange: onYtStateChange },
  })
}

function onYtReady() {
  isLoading.value = false
  duration.value = ytPlayer.getDuration() || 0
  ytPlayer.setVolume(volume.value * 100)
  if (isMuted.value) ytPlayer.mute()

  if (pendingSeekTime.value !== null) {
    ytPlayer.seekTo(pendingSeekTime.value, true)
    currentTime.value = pendingSeekTime.value
    pendingSeekTime.value = null
  }
  if (!props.isHost) {
    ignoreStateChange = true
    props.playerState.isPlaying ? ytPlayer.playVideo() : ytPlayer.pauseVideo()
    setTimeout(() => { ignoreStateChange = false }, 200)
  }
}

function onYtStateChange(event) {
  if (!ytPlayer) return
  const state = event.data
  const time = ytPlayer.getCurrentTime()
  if (state === 1) {
    isPlaying.value = true; startYtTimePolling()
    if (props.isHost && !ignoreStateChange) emit('sync', { isPlaying: true, currentTime: time })
  } else {
    isPlaying.value = false; stopYtTimePolling()
    if (state === 2 && props.isHost && !ignoreStateChange) emit('sync', { isPlaying: false, currentTime: time })
    if (state === 0 && props.isHost) emit('ended')
  }
}

function startYtTimePolling() {
  stopYtTimePolling()
  timeUpdateTimer = setInterval(() => {
    if (!ytPlayer) return
    currentTime.value = ytPlayer.getCurrentTime()
    duration.value = ytPlayer.getDuration() || 0
    if (props.isHost && Date.now() - lastSyncTime > 2000) {
      lastSyncTime = Date.now()
      if (ytPlayer.getPlayerState() === 1) emit('sync', { isPlaying: true, currentTime: currentTime.value })
    }
  }, 500)
}

function stopYtTimePolling() {
  if (timeUpdateTimer) { clearInterval(timeUpdateTimer); timeUpdateTimer = null }
}

/* ================================================================
   HLS via hls.js
   ================================================================ */
function sanitizeStreamUrl(url) {
  if (!url) return url
  // Guard against residual JSON escapes from scrapers (expire\= … \&ei\=)
  let s = String(url)
  s = s.replace(/\\u003d/gi, '=').replace(/\\u0026/gi, '&').replace(/\\u002f/gi, '/')
  s = s.replace(/\\=/g, '=').replace(/\\&/g, '&').replace(/\\\//g, '/')
  return s
}

function getProxiedUrl(url, referer) {
  if (!url) return url
  if (url.startsWith('/api/proxy')) return url
  if (API_BASE && url.startsWith(`${API_BASE}/api/proxy`)) return url
  // Relative URL from hls.js segment — already handled by m3u8 rewriter on backend
  if (!url.startsWith('http')) return url
  url = sanitizeStreamUrl(url)
  let proxied = `${API_BASE}/api/proxy?url=${encodeURIComponent(url)}`
  if (referer) {
    proxied += `&referer=${encodeURIComponent(referer)}`
  } else {
    try {
      const host = new URL(url).hostname.toLowerCase()
      // Sokuja progressive MP4 CDN prefers site origin over storage host
      if (host.includes('sokuja')) {
        proxied += `&referer=${encodeURIComponent('https://sokuja.uk/')}`
      }
      // Blogger / Anoboy streams on googlevideo require blogger referer
      if (host.includes('googlevideo') || host.includes('googleusercontent') || url.includes('videoplayback')) {
        proxied += `&referer=${encodeURIComponent('https://www.blogger.com/')}`
      }
    } catch (_) { /* ignore */ }
  }
  return proxied
}

function initHls(url) {
  destroyHls()
  const vid = videoRef.value
  if (!vid) return

  isLoading.value = true

  if (Hls.isSupported()) {
    hlsInstance = new Hls({
      enableWorker: true,
      lowLatencyMode: false,
      maxBufferLength: 30,
      maxMaxBufferLength: 60,
      maxBufferSize: 60 * 1000 * 1000,
      backBufferLength: 30,
      manifestLoadingTimeOut: 15000,
      manifestLoadingMaxRetry: 4,
      levelLoadingTimeOut: 15000,
      fragLoadingTimeOut: 20000,
      fragLoadingMaxRetry: 4,
      xhrSetup: (xhr, url) => {
        try {
          const parsed = new URL(url)
          xhr.setRequestHeader('Referer', parsed.origin + '/')
        } catch (e) { /* ignore */ }
      }
    })
    hlsInstance.loadSource(getProxiedUrl(url))
    hlsInstance.attachMedia(vid)
    hlsInstance.on(Hls.Events.MANIFEST_PARSED, () => {
      isLoading.value = false
      if (pendingSeekTime.value !== null) {
        vid.currentTime = pendingSeekTime.value
        pendingSeekTime.value = null
      }
      if (!props.isHost && props.playerState.isPlaying) {
        vid.play().catch(() => {})
      }
    })
    hlsInstance.on(Hls.Events.ERROR, (_, data) => {
      console.error('[HLS] Error:', data.type, data.details, data.fatal)
      if (data.fatal) {
        switch (data.type) {
          case Hls.ErrorTypes.NETWORK_ERROR:
            if (data.details === Hls.ErrorDetails.MANIFEST_LOAD_ERROR ||
                data.details === Hls.ErrorDetails.MANIFEST_LOAD_TIMEOUT) {
              console.error('[HLS] Cannot load manifest.')
              streamError.value = 'Failed to load stream. The video source may be unavailable.'
              emit('error', streamError.value)
              isLoading.value = false
            } else {
              console.warn('[HLS] Attempting network recovery...')
              hlsInstance.startLoad()
            }
            break
          case Hls.ErrorTypes.MEDIA_ERROR:
            console.warn('[HLS] Attempting media error recovery...')
            hlsInstance.recoverMediaError()
            break
          default:
            console.error('[HLS] Unrecoverable error')
            streamError.value = 'An unrecoverable playback error occurred. The stream may be incompatible.'
            emit('error', streamError.value)
            destroyHls()
            isLoading.value = false
            break
        }
      }
    })
  } else if (vid.canPlayType('application/vnd.apple.mpegurl')) {
    // Safari native HLS support — also route through proxy
    vid.src = getProxiedUrl(url)
    isLoading.value = false
  } else {
    console.error('[HLS] This browser does not support HLS playback')
    isLoading.value = false
  }
}

function destroyHls() {
  if (hlsInstance) {
    hlsInstance.destroy()
    hlsInstance = null
  }
}

/* ================================================================
   Native HTML5 <video> (direct MP4, etc.)
   ================================================================ */
function initNative(url) {
  const vid = videoRef.value
  if (!vid) return
  isLoading.value = true
  // Progressive MP4 (Sokuja ~80MB+) streams via proxy; Range requests must stay open
  vid.preload = 'auto'
  vid.src = getProxiedUrl(url)
  vid.load()
}

/* ================================================================
   Native <video> Event Handlers (shared by HLS and Native modes)
   ================================================================ */
function onNativeTimeUpdate() {
  const vid = videoRef.value
  if (!vid) return
  currentTime.value = vid.currentTime
  duration.value = vid.duration || 0

  if (props.isHost && Date.now() - lastSyncTime > 1000) {
    lastSyncTime = Date.now()
    if (!vid.paused) emit('sync', {
      isPlaying: true,
      currentTime: vid.currentTime,
      duration: vid.duration || 0
    })
  }
}

function onNativePlay() {
  isPlaying.value = true
  if (props.isHost) emit('sync', { isPlaying: true, currentTime: videoRef.value?.currentTime || 0 })
}

function onNativePause() {
  isPlaying.value = false
  if (props.isHost) emit('sync', { isPlaying: false, currentTime: videoRef.value?.currentTime || 0 })
}

function onNativeSeeking() {
  if (props.isHost) {
    emit('sync', { isPlaying: !videoRef.value?.paused, currentTime: videoRef.value?.currentTime || 0 })
  }
}

function onNativeLoadedMetadata() {
  const vid = videoRef.value
  if (vid && pendingSeekTime.value !== null) {
    vid.currentTime = pendingSeekTime.value
    pendingSeekTime.value = null
  }
}

function onNativeEnded() {
  if (props.isHost) {
    emit('ended')
  }
}

function onNativeError() {
  const vid = videoRef.value
  const code = vid?.error?.code
  // If native player fails with code 4 (unsupported format) and URL could be an embed:
  if (code === 4 && props.videoUrl) {
    const raw = String(props.videoUrl).toLowerCase()
    if (isEmbedUrl(props.videoUrl) || raw.includes('video.g') || raw.includes('blogger') || raw.includes('mega.nz')) {
      console.info('[Stream] Falling back to embedded iframe player')
      videoMode.value = 'iframe'
      isLoading.value = true
      return
    }
  }
  const messages = {
    1: 'Video playback was aborted.',
    2: 'A network error occurred while loading the video.',
    3: 'The video could not be decoded.',
    4: 'The video format or source is not supported.',
  }
  streamError.value = messages[code] || 'An unknown error occurred while loading the video.'
  isLoading.value = false
  emit('error', streamError.value)
}

function onIframeLoad() {
  isLoading.value = false
  streamError.value = null
}

function clearErrorAndFocus() {
  streamError.value = null
  const urlInput = document.getElementById('room-video-url-input')
  if (urlInput) {
    urlInput.focus()
    urlInput.select()
  }
}

/* ================================================================
   Unified Player Lifecycle
   ================================================================ */
function destroyAllPlayers() {
  // YouTube
  if (ytPlayer) {
    try { ytPlayer.destroy() } catch (e) { /* noop */ }
    ytPlayer = null
  }
  stopYtTimePolling()

  // HLS
  destroyHls()

  // Native
  if (videoRef.value) {
    videoRef.value.pause()
    videoRef.value.removeAttribute('src')
    videoRef.value.load()
  }
}

function loadVideo(url) {
  destroyAllPlayers()
  pendingSeekTime.value = null
  streamError.value = null

  url = sanitizeStreamUrl(url)
  const mode = detectMode(url)
  videoMode.value = mode

  if (!mode) {
    // Anime page URL still in player state (scrape not applied) — stay idle, no error
    if (isLikelyHtmlPageUrl(url)) {
      isLoading.value = false
      return
    }
    streamError.value = 'This video source is not supported.'
    isLoading.value = false
    return
  }

  switch (mode) {
    case 'youtube':
      initYoutube(extractYoutubeId(url))
      break
    case 'hls':
      initHls(url)
      break
    case 'iframe':
      isLoading.value = true
      break
    case 'native':
      initNative(url)
      break
  }
}

/* ================================================================
   Watch: Incoming player state from WebSocket (guests)
   ================================================================ */

// Watch play state changes
watch(
  () => props.playerState.isPlaying,
  (newPlaying) => {
    if (props.isHost) return
    const vid = videoRef.value
    if (!vid) return
    if (videoMode.value === 'youtube') {
      if (!ytPlayer || typeof ytPlayer.getPlayerState !== 'function') return
      ignoreStateChange = true
      const ytState = ytPlayer.getPlayerState()
      if (newPlaying) { if (ytState !== 1) ytPlayer.playVideo() }
      else { if (ytState !== 2) ytPlayer.pauseVideo() }
      setTimeout(() => { ignoreStateChange = false }, 200)
    } else {
      if (newPlaying && vid.paused) vid.play().catch(() => {})
      else if (!newPlaying && !vid.paused) vid.pause()
    }
  }
)

// Watch current time changes (debounced seek)
let seekTimeout = null
watch(
  () => props.playerState.currentTime,
  (newTime) => {
    if (props.isHost) return
    clearTimeout(seekTimeout)
    seekTimeout = setTimeout(() => {
      if (videoMode.value === 'youtube') {
        if (!ytPlayer || typeof ytPlayer.getPlayerState !== 'function') return
        const ytTime = ytPlayer.getCurrentTime()
        if (Math.abs(ytTime - newTime) > 1.5) {
          ytPlayer.seekTo(newTime, true)
          currentTime.value = newTime
        }
      } else {
        const vid = videoRef.value
        if (!vid) return
        if (Math.abs(vid.currentTime - newTime) > 1.5) {
          if (vid.readyState >= 1) {
            vid.currentTime = newTime
          } else {
            pendingSeekTime.value = newTime
          }
        }
      }
    }, 300)
  }
)

watch(() => props.videoUrl, (newUrl) => { loadVideo(newUrl) })

/* ================================================================
   Unified Controls
   ================================================================ */
function togglePlay() {
  if (!props.isHost) return

  if (videoMode.value === 'youtube') {
    if (!ytPlayer) return
    ytPlayer.getPlayerState() === 1 ? ytPlayer.pauseVideo() : ytPlayer.playVideo()
  } else {
    const vid = videoRef.value
    if (!vid) return
    vid.paused ? vid.play().catch(() => {}) : vid.pause()
  }
  flashPlayPulse()
}

function flashPlayPulse() {
  showPlayPulse.value = true
  setTimeout(() => { showPlayPulse.value = false }, 600)
}

function toggleMute() {
  isMuted.value = !isMuted.value
  if (videoMode.value === 'youtube' && ytPlayer) {
    isMuted.value ? ytPlayer.mute() : ytPlayer.unMute()
  } else if (videoRef.value) {
    videoRef.value.muted = isMuted.value
  }
}

function onVolumeChange() {
  if (videoMode.value === 'youtube' && ytPlayer) {
    ytPlayer.setVolume(volume.value * 100)
    isMuted.value = volume.value === 0
    isMuted.value ? ytPlayer.mute() : ytPlayer.unMute()
  } else if (videoRef.value) {
    videoRef.value.volume = volume.value
    isMuted.value = volume.value === 0
  }
}

function onSeek(event) {
  if (!props.isHost) return
  const bar = event.currentTarget
  const rect = bar.getBoundingClientRect()
  const ratio = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width))
  const targetTime = ratio * duration.value

  if (videoMode.value === 'youtube' && ytPlayer) {
    ytPlayer.seekTo(targetTime, true)
  } else if (videoRef.value) {
    videoRef.value.currentTime = targetTime
  }
  currentTime.value = targetTime
  emit('sync', { isPlaying: isPlaying.value, currentTime: targetTime })
}

function toggleFullscreen() {
  const el = wrapperRef.value
  if (!el) return
  if (!document.fullscreenElement) {
    el.requestFullscreen(); isFullscreen.value = true
  } else {
    document.exitFullscreen(); isFullscreen.value = false
  }
}

function onMouseMove() {
  showControls.value = true
  clearTimeout(controlsTimer)
  hideControlsDelayed()
}

function hideControlsDelayed() {
  controlsTimer = setTimeout(() => {
    if (isPlaying.value && !isInteracting.value) showControls.value = false
  }, 3000)
}

function formatTime(secs) {
  if (!secs || isNaN(secs)) return '0:00'
  const m = Math.floor(secs / 60)
  const s = Math.floor(secs % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

/* ================================================================
   Lifecycle
   ================================================================ */
function onFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
}

function handleKeyDown(e) {
  const tag = document.activeElement?.tagName?.toLowerCase()
  if (tag === 'input' || tag === 'textarea' || document.activeElement?.isContentEditable) {
    return
  }

  if (e.key === 'p' || e.key === 'P') {
    e.preventDefault()
    togglePiP()
  } else if ((e.key === 'c' || e.key === 'C') && isFullscreen.value) {
    e.preventDefault()
    showFsChat.value = !showFsChat.value
  } else if (e.key === 'f' || e.key === 'F') {
    e.preventDefault()
    toggleFullscreen()
  }
}

onMounted(() => {
  if (props.videoUrl) loadVideo(props.videoUrl)
  document.addEventListener('fullscreenchange', onFullscreenChange)
  window.addEventListener('keydown', handleKeyDown)
  if (videoRef.value) {
    videoRef.value.addEventListener('enterpictureinpicture', () => { isPiP.value = true })
    videoRef.value.addEventListener('leavepictureinpicture', () => { isPiP.value = false })
  }
})

onUnmounted(() => {
  destroyAllPlayers()
  clearTimeout(controlsTimer)
  clearTimeout(seekTimeout)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  window.removeEventListener('keydown', handleKeyDown)
})
</script>

<style scoped>
.video-wrapper {
  position: relative;
  width: 100%;
  background: #000;
  border-radius: var(--radius-lg);
  overflow: visible;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-card);
}
.video-wrapper > .player-container,
.video-wrapper > .video-el,
.video-wrapper > .video-placeholder,
.video-wrapper > .video-loading,
.video-wrapper > .video-error,
.video-wrapper > .controls-overlay {
  overflow: hidden;
  border-radius: var(--radius-lg);
}

/* Player Container & IFrame sizing */
.player-container {
  width: 100%;
  aspect-ratio: 16 / 9;
  background: #000;
}
.player-container :deep(iframe),
.embed-iframe {
  width: 100% !important;
  height: 100% !important;
  display: block;
  border: none;
}
.video-wrapper.is-iframe .controls-overlay {
  display: none !important;
}

/* Native video element */
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
  background: radial-gradient(ellipse at center, rgba(255,255,255,0.04) 0%, #080808 70%);
}
.placeholder-icon {
  color: rgba(255,255,255,0.15);
}
.placeholder-text {
  color: var(--text-muted);
  font-size: 0.9375rem;
  text-align: center;
  padding: 0 var(--space-4);
}

.placeholder-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
}

.placeholder-hint {
  color: var(--text-muted);
  font-size: 0.8125rem;
  text-align: center;
  opacity: 0.6;
  max-width: 320px;
  line-height: 1.4;
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

/* Error overlay */
.video-error {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  background: rgba(0,0,0,0.85);
  color: var(--text-secondary);
  z-index: 3;
  padding: var(--space-6);
  text-align: center;
}
.video-error svg {
  color: var(--color-accent-red);
  opacity: 0.8;
}
.error-title {
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}
.error-detail {
  font-size: 0.8125rem;
  color: var(--text-muted);
  margin: 0;
  max-width: 360px;
  line-height: 1.5;
}
.error-cta {
  margin-top: var(--space-2);
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
  width: 68px; height: 68px;
  border-radius: 50%;
  background: rgba(255,255,255,0.08);
  border: 1.5px solid rgba(255,255,255,0.2);
  display: flex; align-items: center; justify-content: center;
  cursor: pointer;
  transition: all var(--transition-base);
}
.center-play-btn:hover { background: rgba(255,255,255,0.14); transform: translate(-50%,-50%) scale(1.06); }
.center-play-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.center-play-btn:disabled:hover {
  background: rgba(0,0,0,0.6);
  transform: translate(-50%,-50%);
}

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
  background: rgba(255,255,255,0.5);
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
  accent-color: rgba(255,255,255,0.5);
  cursor: pointer;
}

.stream-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-secondary);
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.1);
}

.guest-indicator {
  display: flex; align-items: center; gap: 4px;
  font-size: 0.75rem;
  color: rgba(255,255,255,0.5);
}

/* Mobile responsive */
@media (max-width: 600px) {
  .controls-buttons {
    gap: 3px;
  }
  .ctrl-btn {
    width: 32px;
    height: 32px;
  }
  .volume-slider { display: none; }
  .stream-badge { display: none; }
  .guest-indicator { display: none; }
  .center-play-btn {
    width: 52px;
    height: 52px;
  }
  .controls-bar {
    padding: 0 var(--space-3) var(--space-3);
  }
  .progress-thumb {
    width: 12px;
    height: 12px;
  }
}

/* Fullscreen Chat Overlay */
.fs-chat-overlay {
  position: absolute;
  top: 1.5rem;
  right: 1.5rem;
  bottom: 5.5rem;
  width: 330px;
  max-width: 85vw;
  background: rgba(13, 17, 23, 0.78);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: var(--radius-lg, 12px);
  display: flex;
  flex-direction: column;
  z-index: 50;
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.7);
  pointer-events: auto;
}

.fs-chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.7rem 0.9rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.fs-chat-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--color-primary, #38bdf8);
}

.fs-chat-close-btn {
  background: transparent;
  border: none;
  color: rgba(255, 255, 255, 0.5);
  font-size: 1.3rem;
  cursor: pointer;
  line-height: 1;
  padding: 0 0.25rem;
}
.fs-chat-close-btn:hover {
  color: #fff;
}

.fs-chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.fs-chat-empty {
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.4);
  text-align: center;
  margin-top: 2rem;
}

.fs-chat-msg {
  font-size: 0.82rem;
  line-height: 1.4;
  word-break: break-word;
}

.fs-chat-msg--system {
  font-style: italic;
  color: rgba(255, 255, 255, 0.45);
  font-size: 0.75rem;
}

.fs-chat-username {
  font-weight: 600;
  color: #38bdf8;
  margin-right: 0.35rem;
}

.fs-chat-username--host {
  color: #f59e0b;
}

.fs-chat-text {
  color: rgba(255, 255, 255, 0.92);
}

.fs-chat-input-wrap {
  display: flex;
  gap: 0.4rem;
  padding: 0.55rem 0.75rem;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(0, 0, 0, 0.35);
  border-radius: 0 0 var(--radius-lg, 12px) var(--radius-lg, 12px);
}

.fs-chat-input {
  flex: 1;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 6px;
  padding: 0.45rem 0.65rem;
  color: #fff;
  font-size: 0.8rem;
  outline: none;
}
.fs-chat-input:focus {
  border-color: var(--color-primary, #38bdf8);
}

.fs-chat-send-btn {
  background: var(--color-primary, #38bdf8);
  color: #000;
  border: none;
  border-radius: 6px;
  padding: 0.45rem 0.75rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
}
.fs-chat-send-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.ctrl-btn--active {
  color: var(--color-primary, #38bdf8) !important;
}
.ctrl-btn--next-ep {
  color: #34d399 !important;
}
.ctrl-btn--next-ep:hover {
  color: #6ee7b7 !important;
  transform: scale(1.1);
}
</style>
