export const API_BASE = (import.meta.env.VITE_API_URL || '').replace(/\/+$/, '')

export const WS_BASE = (import.meta.env.VITE_WS_URL || '').replace(/\/+$/, '') || (
  typeof window !== 'undefined'
    ? `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}`
    : ''
)
