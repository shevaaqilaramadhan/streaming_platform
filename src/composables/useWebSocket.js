/**
 * useWebSocket — composable for managing a WebSocket connection to the Go backend.
 *
 * Usage:
 *   const { status, send, onMessage, connect, disconnect } = useWebSocket(roomId)
 *
 * The composable handles:
 *  - Auto-connect on mount
 *  - Auto-reconnect with exponential backoff on unexpected close
 *  - Sending typed message payloads
 *  - Registering message handler callbacks
 */

import { ref, onUnmounted } from 'vue'

/** WebSocket URL builder — uses same host so Vite proxy forwards to Go backend */
function getWsBase() {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}`
}
const WS_BASE = import.meta.env.VITE_WS_URL || getWsBase()

const WS_STATUS = {
  CONNECTING: 'connecting',
  CONNECTED:  'connected',
  DISCONNECTED: 'disconnected',
  RECONNECTING: 'reconnecting',
}

/** Rooms this browser was kicked from — survives reconnect races across WS instances */
const kickedRoomIds = new Set()

export function markRoomKicked(id) {
  if (id) kickedRoomIds.add(String(id))
}

export function isRoomKicked(id) {
  return id ? kickedRoomIds.has(String(id)) : false
}

export function clearRoomKicked(id) {
  if (id) kickedRoomIds.delete(String(id))
}

export function useWebSocket(roomId) {
  const status = ref(WS_STATUS.DISCONNECTED)
  const ws = ref(null)
  let messageHandlers = []
  let reconnectTimer = null
  let reconnectAttempts = 0
  const MAX_RECONNECT_ATTEMPTS = 5
  let intentionalClose = false
  // Permanent: kick / ban — never auto-reconnect this socket instance
  let permanentClose = false
  const roomKey = String(roomId || '')

  function clearReconnect() {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }

  function shouldStayDead() {
    return permanentClose || intentionalClose || isRoomKicked(roomKey)
  }

  function connect() {
    if (shouldStayDead()) {
      status.value = WS_STATUS.DISCONNECTED
      return
    }
    if (ws.value && (
      ws.value.readyState === WebSocket.OPEN ||
      ws.value.readyState === WebSocket.CONNECTING
    )) return

    intentionalClose = false
    status.value = reconnectAttempts > 0 ? WS_STATUS.RECONNECTING : WS_STATUS.CONNECTING

    const url = `${WS_BASE}/ws/${roomId}`
    const socket = new WebSocket(url)
    ws.value = socket

    socket.onopen = () => {
      if (shouldStayDead()) {
        try { socket.close() } catch { /* ignore */ }
        status.value = WS_STATUS.DISCONNECTED
        return
      }
      status.value = WS_STATUS.CONNECTED
      reconnectAttempts = 0
    }

    socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        // Process KICKED before any other handlers can trigger side effects
        if (data?.action === 'KICKED') {
          permanentClose = true
          intentionalClose = true
          markRoomKicked(roomKey)
          clearReconnect()
        }
        messageHandlers.forEach(fn => {
          try { fn(data) } catch (e) { console.error('[WS] handler error', e) }
        })
      } catch (err) {
        console.warn('[WS] Failed to parse message:', event.data)
      }
    }

    socket.onclose = (ev) => {
      // Ignore stale sockets
      if (ws.value && ws.value !== socket) return
      status.value = WS_STATUS.DISCONNECTED
      ws.value = null

      // Close code 1000 with reason kicked, or permanent flag
      const reason = (ev && ev.reason) || ''
      if (reason.toLowerCase().includes('kick')) {
        permanentClose = true
        markRoomKicked(roomKey)
      }

      if (shouldStayDead()) return

      if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
        const delay = Math.min(1000 * 2 ** reconnectAttempts, 30000)
        reconnectAttempts++
        status.value = WS_STATUS.RECONNECTING
        reconnectTimer = setTimeout(() => {
          if (!shouldStayDead()) connect()
        }, delay)
      }
    }

    socket.onerror = (err) => {
      console.error('[WS] Error:', err)
    }
  }

  function disconnect(opts = {}) {
    if (opts.permanent) {
      permanentClose = true
      markRoomKicked(roomKey)
    }
    intentionalClose = true
    clearReconnect()
    reconnectAttempts = MAX_RECONNECT_ATTEMPTS
    const socket = ws.value
    ws.value = null
    if (socket) {
      try {
        socket.onclose = null
        socket.onmessage = null
        socket.onerror = null
        socket.close()
      } catch { /* ignore */ }
    }
    status.value = WS_STATUS.DISCONNECTED
  }

  /**
   * Send a typed message over the WebSocket.
   * @param {object} payload — must match a schema defined in WEBSOCKET_CONTRACT.md
   */
  function send(payload) {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify(payload))
    } else {
      console.warn('[WS] Cannot send — socket not open. Status:', status.value)
    }
  }

  /**
   * Register a callback for incoming messages.
   * @param {function} fn — receives parsed JSON object
   * @returns {function} — unregister function
   */
  function onMessage(fn) {
    messageHandlers.push(fn)
    return () => {
      messageHandlers = messageHandlers.filter(h => h !== fn)
    }
  }

  // Handle browser online/offline events for auto-reconnection
  const handleOnline = () => {
    if (permanentClose || intentionalClose) return
    if (!ws.value || ws.value.readyState !== WebSocket.OPEN) {
      console.log('[WS] Internet connection restored. Reconnecting WebSocket immediately...')
      reconnectAttempts = 0
      clearReconnect()
      connect()
    }
  }

  const handleOffline = () => {
    console.warn('[WS] Internet connection lost.')
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)
  }

  onUnmounted(() => {
    disconnect()
    if (typeof window !== 'undefined') {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
    }
  })

  return {
    status,
    WS_STATUS,
    connect,
    disconnect,
    send,
    onMessage,
  }
}
