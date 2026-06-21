# WebSocket Contract — WatchParty

This document defines the WebSocket message protocol between the **Vue 3 frontend** and the **Go backend**.
The frontend (`useWebSocket.js`) connects to this endpoint and speaks this exact JSON schema.

---

## Backend File → Contract Mapping

| Backend file | Responsible for |
|---|---|
| `backend/main.go` | Register `POST /api/rooms` and `GET /ws/:roomId` HTTP routes |
| `backend/handlers/websocket.go` | Upgrade HTTP → WS; call `room_manager` + `message_handler` per message |
| `backend/models/types.go` | Define `User`, `WatchRoom`, `Message` structs matching schemas below |
| `backend/services/room_manager.go` | `POST /api/rooms` logic; thread-safe room create/get/delete |
| `backend/services/message_handler.go` | Route `action` field to correct broadcast/targeted send logic |
| `backend/utils/crypto.go` | Generate short room IDs returned by `POST /api/rooms` |

---

## Endpoint

```
ws://<host>:8080/ws/:roomId
```

**Examples:**
- `ws://localhost:8080/ws/abc123` (development)
- `wss://watchparty.com/ws/abc123` (production, TLS)

---

## General Envelope

Every message (in both directions) uses this wrapper:

```json
{
  "action": "<MSG_TYPE>",
  "payload": { ... }
}
```

---

## REST Endpoint (needed by frontend)

### `POST /api/rooms`

Called by `HomePage.vue` when the user clicks "Create Watch Room".

**Response:**
```json
{ "roomId": "xyz123" }
```

The backend should:
1. Generate a short cryptographic string as `roomId`
2. Initialize a `WatchRoom` in Go memory
3. Return the `roomId`

---

## Message Types

### 1. `JOIN_EVENT`
**Direction:** Client → Server  
**When:** Sent immediately after WS connection is established.

```json
{
  "action": "JOIN_EVENT",
  "payload": {
    "roomId":   "xyz123",
    "username": "MovieFan42",
    "isHost":   false
  }
}
```

**Server should:**
- Add the user to `WatchRoom.Clients`
- Broadcast a `JOIN_EVENT` to all other clients in the room
- Send back a `ROOM_INIT` to the newly connected client

---

### 2. `ROOM_INIT`
**Direction:** Server → Client (targeted, only to the new joiner)  
**When:** Sent immediately after the server processes a `JOIN_EVENT`.

```json
{
  "action": "ROOM_INIT",
  "payload": {
    "roomId":       "xyz123",
    "currentVideo": "https://cdn.example.com/movie.mp4",
    "currentTime":  142.35,
    "isPlaying":    true,
    "participants": [
      { "userId": "host-uuid", "username": "Host" },
      { "userId": "abc-uuid",  "username": "MovieFan42" }
    ]
  }
}
```

---

### 3. `SYNC_EVENT`
**Direction:** Client (host only) → Server → broadcast to all other clients  
**When:** Host plays, pauses, or seeks the video.

**Client sends:**
```json
{
  "action": "SYNC_EVENT",
  "payload": {
    "roomId":      "xyz123",
    "playerState": "PAUSED",
    "currentTime": 142.35,
    "sentAt":      1718987482
  }
}
```

**`playerState`** values: `"PLAYING"` | `"PAUSED"`

**Server should:**
- Update `WatchRoom.CurrentTime` and `WatchRoom.IsPlaying`
- Broadcast this exact message to all *other* clients in the room (not back to sender)

---

### 4. `CHAT_EVENT`
**Direction:** Client → Server → broadcast to all clients  
**When:** Any user sends a chat message.

**Client sends:**
```json
{
  "action": "CHAT_EVENT",
  "payload": {
    "roomId":   "xyz123",
    "userId":   "MovieFan42",
    "username": "MovieFan42",
    "text":     "omg this scene!!",
    "sentAt":   1718987520000
  }
}
```

**Server should:**
- Broadcast this message to **all** clients in the room (including sender for confirmation)
- No persistence needed — pure reflector behaviour

---

### 5. `SET_VIDEO`
**Direction:** Client (host only) → Server → broadcast to all clients  
**When:** Host pastes a new video URL.

**Client sends:**
```json
{
  "action": "SET_VIDEO",
  "payload": {
    "roomId": "xyz123",
    "url":    "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4"
  }
}
```

**Server should:**
- Update `WatchRoom.CurrentVideo`
- Reset `CurrentTime` to 0 and `IsPlaying` to false
- Broadcast to all clients

---

### 6. `JOIN_EVENT` (broadcast)
**Direction:** Server → all other clients in room  
**When:** A new user joins.

```json
{
  "action": "JOIN_EVENT",
  "payload": {
    "userId":   "some-uuid",
    "username": "NewUser"
  }
}
```

---

### 7. `USER_LEFT`
**Direction:** Server → all clients in room  
**When:** A WebSocket connection closes.

```json
{
  "action": "USER_LEFT",
  "payload": {
    "userId":   "some-uuid",
    "username": "MovieFan42"
  }
}
```

**Server should:**
- Remove user from `WatchRoom.Clients` on WS disconnect
- Broadcast this message to remaining clients

---

## CORS / Connection Notes

- The Go server must allow WebSocket upgrades from the Vue dev server origin (`http://localhost:5173`)
- For production, configure appropriate `Origin` checking
- Vite dev server proxy config (`vite.config.js`) can forward `/api` and `/ws` to `localhost:8080`

---

## Vite Proxy (already configured in `vite.config.js`)

```js
server: {
  proxy: {
    '/api': 'http://localhost:8080',
    '/ws': {
      target: 'ws://localhost:8080',
      ws: true,
    },
  },
}
```

This means the frontend uses relative paths (`/api/rooms`, `ws://localhost:5173/ws/...` → proxied to Go).
