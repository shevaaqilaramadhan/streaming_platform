# Frontend Lobby & Room Discovery Task — WatchParty Vue Frontend

> **For the frontend agent**: The backend now supports public room discovery with a live lobby dashboard. This document specifies how to build the Lobby homepage that displays active watch parties.

---

## Overview

The WatchParty platform now has a **Room Discovery** feature where:

1. **Rooms are private by default** - hosts can toggle public visibility
2. **Public rooms appear in lobby** - displayed as a grid of clickable cards
3. **Only active rooms shown** - rooms with at least 1 participant
4. **Sorted by popularity** - highest participant count first
5. **Auto-refreshing** - polls backend every 15 seconds
6. **Direct join flow** - click card to navigate to room

---

## Backend API (Already Implemented)

### GET /api/public-rooms

**Endpoint:** `http://localhost:8080/api/public-rooms`

**Method:** GET

**Response:** Array of public room objects

```json
[
  {
    "roomId": "abc123",
    "roomName": "Anime Marathon Night",
    "hostUsername": "MovieFan42",
    "participantCount": 5,
    "queueSize": 3,
    "currentMetadata": {
      "id": "",
      "url": "https://cdn.example.com/stream.m3u8",
      "title": "Chainsaw Man",
      "episode": "Episode 12",
      "thumbnail": "https://otakudesu.blog/wp-content/uploads/2023/csm.jpg"
    }
  },
  {
    "roomId": "xyz789",
    "roomName": "",
    "hostUsername": "AnimeKing",
    "participantCount": 3,
    "queueSize": 0,
    "currentMetadata": null
  }
]
```

**Response Fields:**

| Field | Type | Description |
|---|---|---|
| `roomId` | `string` | Unique room identifier (for joining) |
| `roomName` | `string` | Optional custom room name (empty if not set) |
| `hostUsername` | `string` | Username of room host |
| `participantCount` | `int` | Number of users watching (≥1) |
| `queueSize` | `int` | Number of videos in queue |
| `currentMetadata` | `object \| null` | Currently playing video metadata |

**Notes:**
- Rooms are sorted by `participantCount` (descending)
- Only includes rooms where `isPublic=true` and `participantCount≥1`
- `currentMetadata` is `null` if no video is playing
- `roomName` is empty string if host hasn't set a custom name

---

## WebSocket Event (Host Toggle Public)

### TOGGLE_PUBLIC (Host Only)

**Direction:** Client → Server

**When:** Host toggles room public/private visibility

**Payload:**
```json
{
  "action": "TOGGLE_PUBLIC",
  "payload": {
    "roomId": "abc123",
    "isPublic": true
  }
}
```

**Backend Response:**
- No broadcast (status change is reflected in next lobby poll)
- Only host can send this event
- Non-host attempts are logged and ignored

**Frontend Implementation:**
Add toggle button in room UI (host only) that emits this event.

---

## Frontend Implementation

### 1. Create Lobby Page/Component

**File:** `src/views/LobbyView.vue` or `src/pages/Lobby.vue` (NEW)

**Basic Structure:**

```vue
<template>
  <div class="lobby-container">
    <header class="lobby-header">
      <h1>WatchParty Lobby</h1>
      <p>Join an active watch party or create your own</p>
      <button @click="createRoom" class="btn-create">
        Create Watch Room
      </button>
    </header>

    <div v-if="loading" class="lobby-loading">
      <p>Loading active rooms...</p>
    </div>

    <div v-else-if="rooms.length === 0" class="lobby-empty">
      <p>No active public rooms right now</p>
      <small>Be the first to create a watch party!</small>
    </div>

    <div v-else class="room-grid">
      <div 
        v-for="room in rooms" 
        :key="room.roomId"
        @click="joinRoom(room.roomId)"
        class="room-card"
      >
        <div 
          v-if="room.currentMetadata?.thumbnail" 
          class="room-thumbnail"
          :style="{ backgroundImage: `url(${room.currentMetadata.thumbnail})` }"
        >
          <div class="room-overlay">
            <div class="room-participants">
              {{ room.participantCount }} watching
            </div>
          </div>
        </div>
        <div v-else class="room-thumbnail-empty">
          <div class="room-overlay">
            <div class="room-participants">
              {{ room.participantCount }} watching
            </div>
          </div>
          <p>No video playing</p>
        </div>

        <div class="room-info">
          <h3 class="room-title">
            {{ getRoomTitle(room) }}
          </h3>
          <p v-if="room.currentMetadata?.episode" class="room-episode">
            {{ room.currentMetadata.episode }}
          </p>
          <div class="room-meta">
            <span class="room-host">Host: {{ room.hostUsername }}</span>
            <span v-if="room.queueSize > 0" class="room-queue">
              {{ room.queueSize }} queued
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const rooms = ref([])
const loading = ref(true)
let pollInterval = null

async function fetchPublicRooms() {
  try {
    const response = await fetch('http://localhost:8080/api/public-rooms')
    if (response.ok) {
      rooms.value = await response.json()
    } else {
      console.error('Failed to fetch public rooms:', response.status)
    }
  } catch (error) {
    console.error('Error fetching public rooms:', error)
  } finally {
    loading.value = false
  }
}

function getRoomTitle(room) {
  if (room.roomName) {
    return room.roomName
  }
  if (room.currentMetadata?.title) {
    return room.currentMetadata.title
  }
  return `${room.hostUsername}'s Room`
}

function joinRoom(roomId) {
  router.push(`/room/${roomId}`)
}

async function createRoom() {
  try {
    const response = await fetch('http://localhost:8080/api/rooms', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    })
    if (response.ok) {
      const data = await response.json()
      router.push(`/room/${data.roomId}`)
    }
  } catch (error) {
    console.error('Error creating room:', error)
  }
}

onMounted(() => {
  fetchPublicRooms()
  pollInterval = setInterval(fetchPublicRooms, 15000)
})

onUnmounted(() => {
  if (pollInterval) {
    clearInterval(pollInterval)
  }
})
</script>

<style scoped>
.lobby-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 2rem;
  min-height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}

.lobby-header {
  text-align: center;
  margin-bottom: 3rem;
  color: white;
}

.lobby-header h1 {
  font-size: 3rem;
  margin: 0 0 0.5rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.lobby-header p {
  font-size: 1.2rem;
  color: #a0a0a0;
  margin: 0 0 2rem;
}

.btn-create {
  padding: 1rem 2rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 1.1rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.btn-create:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 16px rgba(102, 126, 234, 0.4);
}

.lobby-loading,
.lobby-empty {
  text-align: center;
  padding: 4rem 2rem;
  color: #a0a0a0;
}

.lobby-empty small {
  display: block;
  margin-top: 1rem;
  font-size: 0.9rem;
  color: #666;
}

.room-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 2rem;
  padding: 1rem 0;
}

.room-card {
  background: #0f3460;
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.room-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(102, 126, 234, 0.3);
}

.room-thumbnail {
  width: 100%;
  height: 180px;
  background-size: cover;
  background-position: center;
  position: relative;
}

.room-thumbnail-empty {
  width: 100%;
  height: 180px;
  background: linear-gradient(135deg, #2a2a3e 0%, #1a1a2e 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
  position: relative;
}

.room-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(to bottom, transparent 0%, rgba(0,0,0,0.7) 100%);
  display: flex;
  align-items: flex-end;
  padding: 1rem;
}

.room-participants {
  background: rgba(102, 126, 234, 0.9);
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-size: 0.9rem;
  font-weight: 600;
  color: white;
}

.room-info {
  padding: 1.25rem;
}

.room-title {
  margin: 0 0 0.5rem;
  font-size: 1.2rem;
  color: white;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.room-episode {
  margin: 0 0 0.75rem;
  color: #a0a0a0;
  font-size: 0.95rem;
}

.room-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.85rem;
  color: #777;
}

.room-host {
  color: #667eea;
}

.room-queue {
  color: #888;
}
</style>
```

---

### 2. Router Integration

**File:** `src/router/index.js` or `src/router/index.ts`

**Add Lobby route:**

```javascript
import LobbyView from '@/views/LobbyView.vue'

const routes = [
  {
    path: '/',
    name: 'Lobby',
    component: LobbyView
  },
  {
    path: '/room/:roomId',
    name: 'Room',
    component: () => import('@/views/RoomView.vue')
  },
  // ... other routes
]
```

---

### 3. Host Toggle Public UI

**File:** `src/components/RoomSettings.vue` or add to room header (NEW)

**Add toggle button for host:**

```vue
<template>
  <div v-if="isHost" class="room-settings">
    <label class="toggle-label">
      <input 
        type="checkbox" 
        v-model="isPublic" 
        @change="togglePublic"
      />
      <span>Make Room Public</span>
    </label>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoom } from '@/composables/useRoom'

const { isHost, roomId, sendWebSocketMessage } = useRoom()
const isPublic = ref(false)

function togglePublic() {
  sendWebSocketMessage({
    action: 'TOGGLE_PUBLIC',
    payload: {
      roomId: roomId.value,
      isPublic: isPublic.value
    }
  })
}
</script>

<style scoped>
.room-settings {
  padding: 1rem;
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}
</style>
```

---

## Testing Checklist

After implementing:

- [ ] **Lobby loads** - Navigate to `/` and see lobby page
- [ ] **Fetch public rooms** - Verify API call to `/api/public-rooms` succeeds
- [ ] **Display rooms** - See grid of room cards with thumbnails and metadata
- [ ] **Empty state** - If no public rooms, see "No active public rooms" message
- [ ] **Auto-refresh** - Wait 15 seconds, verify rooms list updates automatically
- [ ] **Click to join** - Click a room card, navigate to `/room/:roomId`
- [ ] **Create room** - Click "Create Watch Room", navigate to new room
- [ ] **Host toggle** - As host, toggle "Make Room Public" checkbox
- [ ] **Appear in lobby** - After toggling public, room appears in lobby (after next poll)
- [ ] **Participant count** - Verify count updates as users join/leave
- [ ] **Sorting** - Verify rooms sorted by participant count (highest first)
- [ ] **No video state** - Rooms with no video show "No video playing"
- [ ] **Custom room names** - If host sets room name, it displays in lobby
- [ ] **Queue size** - Rooms with queued videos show queue count

---

## Summary

**Backend provides:**
- GET /api/public-rooms endpoint (filtered, sorted)
- TOGGLE_PUBLIC WebSocket event (host-only)
- Automatic filtering (isPublic=true, participantCount≥1)
- Sorting by participant count (descending)

**Frontend displays:**
- Lobby homepage with grid of active rooms
- Room cards showing anime thumbnail, title, episode, participant count
- Auto-refreshing every 15 seconds (polling)
- Direct join by clicking cards
- Host toggle to make room public/private

**User Experience:**
1. Visitor lands on lobby (`/`)
2. Sees grid of active watch parties sorted by popularity
3. Clicks a room card to join directly
4. Or creates new room with "Create Watch Room" button
5. Host can toggle room public to appear in lobby

---

**Implementation ready. Build the lobby and integrate with existing room system.**
