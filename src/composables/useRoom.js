/**
 * useRoom — composable for room state management.
 * Wraps the WebSocket composable and manages local player/chat state.
 */

import { ref, computed, onMounted, watch } from 'vue'
import { useWebSocket } from './useWebSocket.js'

export const MSG_TYPES = {
  SYNC_EVENT:   'SYNC_EVENT',
  CHAT_EVENT:   'CHAT_EVENT',
  JOIN_EVENT:   'JOIN_EVENT',
  ROOM_INIT:    'ROOM_INIT',
  SET_VIDEO:    'SET_VIDEO',
  USER_LEFT:    'USER_LEFT',
  SCRAPE_ERROR: 'SCRAPE_ERROR',
  QUEUE_UPDATE: 'QUEUE_UPDATE',
  CURRENT_VIDEO_CHANGED: 'CURRENT_VIDEO_CHANGED',
  ADD_TO_QUEUE: 'ADD_TO_QUEUE',
  REMOVE_FROM_QUEUE: 'REMOVE_FROM_QUEUE',
  SKIP_TO_NEXT: 'SKIP_TO_NEXT',
  CLEAR_QUEUE: 'CLEAR_QUEUE',
  EPISODE_ENDED: 'EPISODE_ENDED',
}

export function useRoom(roomId, nickname, isHost) {
  const { status, WS_STATUS, connect, disconnect, send, onMessage } = useWebSocket(roomId)

  /* ---- Chat State ---- */
  const messages = ref([])

  /* ---- Player State ---- */
  const playerState = ref({
    videoUrl: '',
    currentTime: 0,
    isPlaying: false,
  })

  /* ---- Scrape Error State ---- */
  const scrapeError = ref(null)

  /* ---- Participants ---- */
  const participants = ref([])

  /* ---- Video Metadata ---- */
  const currentMetadata = ref(null)

  /* ---- Play Queue ---- */
  const queue = ref([])

  /* ---- Computed ---- */
  const isConnected = computed(() => status.value === WS_STATUS.CONNECTED)

  // Watch status to send JOIN_EVENT when connected
  watch(status, (newStatus) => {
    if (newStatus === WS_STATUS.CONNECTED) {
      send({
        action: MSG_TYPES.JOIN_EVENT,
        payload: {
          roomId,
          username: nickname,
          isHost,
        },
      })
    }
  })

  /* ---- Incoming Message Handler ---- */
  const unregister = onMessage((data) => {
    switch (data.action) {
      case MSG_TYPES.ROOM_INIT:
        // Server tells the new joiner the current room state
        if (data.payload) {
          playerState.value.videoUrl    = data.payload.currentVideo || ''
          playerState.value.currentTime = data.payload.currentTime  || 0
          playerState.value.isPlaying   = data.payload.isPlaying    || false
        }
        if (data.payload?.participants) {
          participants.value = data.payload.participants
        }
        currentMetadata.value = data.payload.metadata || null
        queue.value = data.payload.queue || []
        break

      case MSG_TYPES.SYNC_EVENT:
        // Received from Go when another user (the host) changes player state
        if (!isHost) {
          playerState.value.isPlaying   = data.payload.playerState === 'PLAYING'
          playerState.value.currentTime = data.payload.currentTime
        }
        break

      case MSG_TYPES.CHAT_EVENT:
        messages.value.push({
          id:       Date.now() + Math.random(),
          userId:   data.payload.userId,
          username: data.payload.username,
          text:     data.payload.text,
          sentAt:   data.payload.sentAt,
          isOwn:    data.payload.userId === nickname, // simplistic, backend will confirm
        })
        break

      case MSG_TYPES.JOIN_EVENT:
        if (data.payload?.username) {
          participants.value.push({ username: data.payload.username, userId: data.payload.userId })
          messages.value.push({
            id:       Date.now() + Math.random(),
            type:     'system',
            text:     `${data.payload.username} joined the room`,
            sentAt:   Date.now(),
          })
        }
        break

      case MSG_TYPES.USER_LEFT:
        if (data.payload?.userId) {
          participants.value = participants.value.filter(p => p.userId !== data.payload.userId)
          messages.value.push({
            id:       Date.now() + Math.random(),
            type:     'system',
            text:     `${data.payload.username} left the room`,
            sentAt:   Date.now(),
          })
        }
        break

      case MSG_TYPES.SET_VIDEO:
        playerState.value.videoUrl    = data.payload.videoUrl
        currentMetadata.value         = data.payload.metadata || null
        playerState.value.currentTime = 0
        playerState.value.isPlaying   = false
        scrapeError.value = null  // clear previous error on successful video load
        break

      case MSG_TYPES.SCRAPE_ERROR:
        scrapeError.value = {
          originalUrl: data.payload.originalUrl,
          error:       data.payload.error,
        }
        console.error('[Scrape] Failed:', data.payload.error, 'URL:', data.payload.originalUrl)
        // Auto-clear after 8 seconds
        setTimeout(() => { scrapeError.value = null }, 8000)
        break

      case MSG_TYPES.QUEUE_UPDATE:
        queue.value = data.payload.queue || []
        break

      case MSG_TYPES.CURRENT_VIDEO_CHANGED:
        playerState.value.videoUrl    = data.payload.videoUrl
        currentMetadata.value         = data.payload.metadata || null
        playerState.value.currentTime = 0
        playerState.value.isPlaying   = false
        scrapeError.value             = null
        break

      default:
        break
    }
  })

  /* ---- Actions ---- */

  function joinRoom() {
    connect()
  }

  function sendChatMessage(text) {
    if (!text.trim()) return
    send({
      action: MSG_TYPES.CHAT_EVENT,
      payload: {
        roomId,
        userId:   nickname,
        username: nickname,
        text:     text.trim(),
        sentAt:   Date.now(),
      },
    })
  }

  function sendSyncEvent({ isPlaying, currentTime }) {
    if (!isHost) return
    send({
      action: MSG_TYPES.SYNC_EVENT,
      payload: {
        roomId,
        playerState: isPlaying ? 'PLAYING' : 'PAUSED',
        currentTime,
        sentAt: Date.now(),
      },
    })
  }

  function setVideo(url) {
    if (!isHost) return
    playerState.value.videoUrl = url
    send({
      action: MSG_TYPES.SET_VIDEO,
      payload: { roomId, url },
    })
  }

  function leave() {
    disconnect()
  }

  function addToQueue(url) {
    send({
      action: MSG_TYPES.ADD_TO_QUEUE,
      payload: { roomId, url }
    })
  }

  function removeFromQueue(itemId) {
    if (!isHost) return
    send({
      action: MSG_TYPES.REMOVE_FROM_QUEUE,
      payload: { roomId, itemId }
    })
  }

  function skipToNext() {
    if (!isHost) return
    send({
      action: MSG_TYPES.SKIP_TO_NEXT,
      payload: { roomId }
    })
  }

  function clearQueue() {
    if (!isHost) return
    send({
      action: MSG_TYPES.CLEAR_QUEUE,
      payload: { roomId }
    })
  }

  function onVideoEnded() {
    send({
      action: MSG_TYPES.EPISODE_ENDED,
      payload: { roomId }
    })
  }

  return {
    status,
    WS_STATUS,
    isConnected,
    messages,
    playerState,
    participants,
    scrapeError,
    currentMetadata,
    queue,
    joinRoom,
    leave,
    sendChatMessage,
    sendSyncEvent,
    setVideo,
    addToQueue,
    removeFromQueue,
    skipToNext,
    clearQueue,
    onVideoEnded,
  }
}
