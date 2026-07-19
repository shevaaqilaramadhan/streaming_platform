/**
 * useRoom — composable for room state management.
 * Wraps the WebSocket composable and manages local player/chat state.
 */

import { ref, computed, onMounted, watch } from 'vue'
import { useWebSocket, markRoomKicked, isRoomKicked, clearRoomKicked } from './useWebSocket.js'

/** Kicked callback — set externally so RoomPage can react */
let onKickedCallback = null

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
  TOGGLE_PUBLIC: 'TOGGLE_PUBLIC',
  HOST_CHANGED: 'HOST_CHANGED',
  // P2 — Host management
  TRANSFER_HOST: 'TRANSFER_HOST',
  KICK_USER: 'KICK_USER',
  KICKED: 'KICKED',
  // P2 — Room name
  SET_ROOM_NAME: 'SET_ROOM_NAME',
  ROOM_NAME_CHANGED: 'ROOM_NAME_CHANGED',
  // Scrape loading
  SCRAPE_STARTED: 'SCRAPE_STARTED',
  SCRAPE_FINISHED: 'SCRAPE_FINISHED',
  // P3 — Reactions & Typing
  REACTION: 'REACTION',
  TYPING: 'TYPING',
}

export function useRoom(roomId, nickname, hostToken = '') {
  const { status, WS_STATUS, connect, disconnect, send, onMessage } = useWebSocket(roomId)

  // After kick: never re-JOIN even if a stale CONNECTED status fires
  let wasKicked = isRoomKicked(roomId)

  // Explicit join into a room the user chose (not a kick-ban forever if they re-create)
  // clear ban only when joining intentionally via joinRoom()
  function allowJoin() {
    wasKicked = false
    clearRoomKicked(roomId)
  }

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

  /* ---- Local user ID (set from ROOM_INIT) ---- */
  const localUserId = ref('')

  /* ---- Server-authoritative host (never trust URL/query) ---- */
  const isHost = ref(false)
  const hostId = ref('')

  /* ---- Video Metadata ---- */
  const currentMetadata = ref(null)

  /* ---- Play Queue ---- */
  const queue = ref([])

  /* ---- Public visibility state ---- */
  const isPublic = ref(false)

  /* ---- Room name ---- */
  const roomName = ref('')

  /* ---- Scrape loading state ---- */
  const scrapeLoading = ref(false)

  /* ---- Typing indicator state ---- */
  const typingUsers = ref([])  // [{ userId, username, timeout }]

  /* ---- Reactions state ---- */
  const reactions = ref([])  // [{ id, emoji, x, timestamp }]

  /* ---- Computed ---- */
  const isConnected = computed(() => status.value === WS_STATUS.CONNECTED)

  // Send JOIN_EVENT every time we connect (including reconnects) — never after kick
  watch(status, (newStatus) => {
    if (newStatus === WS_STATUS.CONNECTED) {
      if (wasKicked || isRoomKicked(roomId)) {
        disconnect({ permanent: true })
        return
      }
      const payload = {
        roomId,
        username: nickname,
      }
      if (hostToken) {
        payload.hostToken = hostToken
      }
      send({
        action: MSG_TYPES.JOIN_EVENT,
        payload,
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
        currentMetadata.value = data.payload?.metadata || null
        queue.value = data.payload?.queue || []
        isPublic.value = data.payload?.isPublic || false
        roomName.value = data.payload?.roomName || ''

        if (data.payload?.hostId) {
          hostId.value = data.payload.hostId
        }

        // Server-authoritative host status
        if (typeof data.payload?.isHost === 'boolean') {
          isHost.value = data.payload.isHost
        }

        // Resolve local userId from participants (prefer exact match, then host if isHost)
        if (data.payload?.participants?.length) {
          const self = data.payload.participants.find(p => p.username === nickname)
            || (data.payload.isHost && data.payload.hostId
              ? data.payload.participants.find(p => p.userId === data.payload.hostId)
              : null)
          if (self) {
            localUserId.value = self.userId
          }
          if (data.payload.hostId) {
            isHost.value = localUserId.value === data.payload.hostId
              || !!data.payload.isHost
          }
        }
        break

      case MSG_TYPES.HOST_CHANGED:
        if (data.payload?.newHostId) {
          hostId.value = data.payload.newHostId
          const amHost = data.payload.newHostId === localUserId.value
          isHost.value = amHost
          messages.value.push({
            id:     Date.now() + Math.random(),
            type:   'system',
            text:   amHost
              ? 'You are now the host'
              : `${data.payload.username || 'Someone'} is now the host`,
            sentAt: Date.now(),
          })
        }
        break

      case MSG_TYPES.TOGGLE_PUBLIC:
        if (typeof data.payload?.isPublic === 'boolean') {
          isPublic.value = data.payload.isPublic
        }
        break

      case MSG_TYPES.KICKED: {
        wasKicked = true
        markRoomKicked(roomId)
        // Stop auto-reconnect immediately (WS layer also marks room kicked on KICKED)
        disconnect({ permanent: true })
        unregister?.()

        const reason = data.payload?.reason || 'kicked_by_host'
        const msg = reason === 'kicked_by_host' || reason === ''
          ? 'You have been removed from the room by the host.'
          : (typeof reason === 'string' && reason.length < 200
            ? reason
            : 'You have been removed from the room by the host.')
        try {
          sessionStorage.setItem('wp_flash', JSON.stringify({
            type: 'kicked',
            message: msg,
          }))
        } catch { /* ignore */ }
        if (onKickedCallback) {
          onKickedCallback(msg)
        } else {
          try {
            window.location.assign('/?kicked=1')
          } catch { /* ignore */ }
        }
        break
      }

      case MSG_TYPES.ROOM_NAME_CHANGED:
        if (data.payload?.roomName !== undefined) {
          roomName.value = data.payload.roomName
        }
        break

      case MSG_TYPES.REACTION:
        if (data.payload?.emoji) {
          const reaction = {
            id: Date.now() + Math.random(),
            emoji: data.payload.emoji,
            x: data.payload.x ?? Math.random() * 80 + 10,
            timestamp: Date.now(),
          }
          reactions.value.push(reaction)
          // Auto-remove after animation (3s)
          setTimeout(() => {
            reactions.value = reactions.value.filter(r => r.id !== reaction.id)
          }, 3000)
        }
        break

      case MSG_TYPES.TYPING:
        if (data.payload?.userId && data.payload.userId !== localUserId.value) {
          const existing = typingUsers.value.find(t => t.userId === data.payload.userId)
          if (existing) {
            clearTimeout(existing.timeout)
            existing.timeout = setTimeout(() => {
              typingUsers.value = typingUsers.value.filter(t => t.userId !== data.payload.userId)
            }, 3500)
          } else {
            const timeout = setTimeout(() => {
              typingUsers.value = typingUsers.value.filter(t => t.userId !== data.payload.userId)
            }, 3500)
            typingUsers.value.push({
              userId: data.payload.userId,
              username: data.payload.username,
              timeout,
            })
          }
        }
        break

      case MSG_TYPES.SYNC_EVENT:
        // Received from Go when another user (the host) changes player state
        if (!isHost.value) {
          const latency = data.payload.serverAt
            ? (Date.now() - data.payload.serverAt) / 1000
            : 0
          playerState.value.isPlaying   = data.payload.playerState === 'PLAYING'
          playerState.value.currentTime = data.payload.currentTime + latency
        }
        break

      case MSG_TYPES.CHAT_EVENT:
        messages.value.push({
          id:       Date.now() + Math.random(),
          userId:   data.payload.userId,
          username: data.payload.username,
          text:     data.payload.text,
          sentAt:   data.payload.sentAt,
          isOwn:    data.payload.userId === localUserId.value,
        })
        if (messages.value.length > 200) {
          messages.value.splice(0, messages.value.length - 200)
        }
        break

      case MSG_TYPES.JOIN_EVENT:
        if (data.payload?.username) {
          const exists = participants.value.some(p => p.userId === data.payload.userId)
          if (!exists) {
            participants.value.push({ username: data.payload.username, userId: data.payload.userId })
          }
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
            text:     `${data.payload.username || 'Someone'} left the room`,
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
        scrapeLoading.value = false  // clear scrape loading
        break

      case MSG_TYPES.SCRAPE_ERROR:
        scrapeError.value = {
          originalUrl: data.payload.originalUrl,
          error:       data.payload.error,
        }
        scrapeLoading.value = false  // clear scrape loading
        console.error('[Scrape] Failed:', data.payload.error, 'URL:', data.payload.originalUrl)
        // Auto-clear after 8 seconds
        setTimeout(() => { scrapeError.value = null }, 8000)
        break

      case MSG_TYPES.SCRAPE_STARTED:
        scrapeLoading.value = true
        break

      case MSG_TYPES.SCRAPE_FINISHED:
        scrapeLoading.value = false
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
        scrapeLoading.value           = false
        break

      default:
        break
    }
  })

  /* ---- Actions ---- */

  function joinRoom() {
    // Explicit user action (opened room page / chose nickname) — clear kick ban
    // so they can re-enter the same room if they still have the link.
    // Auto-reconnect after kick is still blocked via wasKicked until this call.
    wasKicked = false
    clearRoomKicked(roomId)
    allowJoin()
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
    if (!isHost.value) return
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
    if (!isHost.value) return
    playerState.value.videoUrl = url
    send({
      action: MSG_TYPES.SET_VIDEO,
      payload: { roomId, url },
    })
  }

  function leave() {
    unregister?.()
    // permanent if kicked so online-event / timers cannot reopen the socket
    disconnect(wasKicked ? { permanent: true } : {})
  }

  function addToQueue(url) {
    send({
      action: MSG_TYPES.ADD_TO_QUEUE,
      payload: { roomId, url }
    })
  }

  function removeFromQueue(itemId) {
    if (!isHost.value) return
    send({
      action: MSG_TYPES.REMOVE_FROM_QUEUE,
      payload: { roomId, itemId }
    })
  }

  function skipToNext() {
    if (!isHost.value) return
    send({
      action: MSG_TYPES.SKIP_TO_NEXT,
      payload: { roomId }
    })
  }

  function clearQueue() {
    if (!isHost.value) return
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

  function togglePublic(isPublicVal) {
    send({
      action: MSG_TYPES.TOGGLE_PUBLIC,
      payload: { roomId, isPublic: isPublicVal }
    })
    isPublic.value = isPublicVal
  }

  function transferHost(targetUserId) {
    if (!isHost.value) return
    send({
      action: MSG_TYPES.TRANSFER_HOST,
      payload: { roomId, targetUserId }
    })
  }

  function kickUser(targetUserId) {
    if (!isHost.value) return
    send({
      action: MSG_TYPES.KICK_USER,
      payload: { roomId, targetUserId }
    })
  }

  function setRoomName(name) {
    send({
      action: MSG_TYPES.SET_ROOM_NAME,
      payload: { roomId, roomName: name }
    })
    roomName.value = name
  }

  function sendReaction(emoji) {
    send({
      action: MSG_TYPES.REACTION,
      payload: { roomId, emoji, x: Math.random() * 80 + 10 }
    })
  }

  function sendTyping() {
    send({
      action: MSG_TYPES.TYPING,
      payload: { roomId }
    })
  }

  function onKicked(cb) {
    onKickedCallback = cb
  }

  return {
    status,
    WS_STATUS,
    isConnected,
    messages,
    playerState,
    participants,
    localUserId,
    isHost,
    hostId,
    scrapeError,
    scrapeLoading,
    currentMetadata,
    queue,
    isPublic,
    roomName,
    typingUsers,
    reactions,
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
    togglePublic,
    transferHost,
    kickUser,
    setRoomName,
    sendReaction,
    sendTyping,
    onKicked,
  }
}
