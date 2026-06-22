# Frontend Queue & Playlist Task — WatchParty Vue Frontend

> **For the frontend agent**: The backend now supports a video queue/playlist system. This document specifies how to build the Queue UI tab and wire up the add/remove/skip operations.

---

## Overview

The WatchParty platform now has a **playlist queue system** where:

1. **Anyone can add** videos to the queue
2. **Only the host can remove/skip/clear** queue items
3. **Auto-advance**: When a video ends, the next item in the queue automatically plays
4. **Rich metadata**: Queue items show anime title, episode, and thumbnail (pre-scraped by backend)

---

## Backend Changes (Already Implemented)

The backend broadcasts these new WebSocket events:

### 1. QUEUE_UPDATE Event

Sent whenever the queue changes (add/remove/clear):

```json
{
  "action": "QUEUE_UPDATE",
  "payload": {
    "roomId": "abc123",
    "queue": [
      {
        "id": "q_xyz789",
        "url": "https://cdn.example.com/stream.m3u8",
        "title": "Chainsaw Man",
        "episode": "Episode 13",
        "thumbnail": "https://otakudesu.blog/wp-content/uploads/2023/csm.jpg"
      },
      {
        "id": "q_abc456",
        "url": "https://youtube.com/watch?v=xyz",
        "title": "Attack on Titan",
        "episode": "Episode 5",
        "thumbnail": "https://i.ytimg.com/vi/xyz/maxresdefault.jpg"
      }
    ]
  }
}
```

### 2. CURRENT_VIDEO_CHANGED Event

Sent when auto-advancing to next video:

```json
{
  "action": "CURRENT_VIDEO_CHANGED",
  "payload": {
    "roomId": "abc123",
    "videoUrl": "https://cdn.example.com/stream.m3u8",
    "metadata": {
      "id": "q_xyz789",
      "url": "https://cdn.example.com/stream.m3u8",
      "title": "Chainsaw Man",
      "episode": "Episode 13",
      "thumbnail": "https://..."
    }
  }
}
```

### 3. ROOM_INIT Event (Updated)

Now includes the queue array:

```json
{
  "action": "ROOM_INIT",
  "payload": {
    "roomId": "abc123",
    "currentVideo": "https://...",
    "currentTime": 142.35,
    "isPlaying": true,
    "participants": [...],
    "metadata": { QueueItem },
    "queue": [
      { QueueItem },
      { QueueItem }
    ]
  }
}
```

---

## Client → Server Actions

### ADD_TO_QUEUE (Anyone)

**When:** User pastes a URL in the queue input box

```json
{
  "action": "ADD_TO_QUEUE",
  "payload": {
    "roomId": "abc123",
    "url": "https://otakudesu.blog/episode/..."
  }
}
```

**Backend response:**
1. Scrapes metadata for the URL
2. Creates a QueueItem with unique ID
3. Appends to queue
4. Broadcasts QUEUE_UPDATE to everyone

---

### REMOVE_FROM_QUEUE (Host Only)

**When:** Host clicks "Remove" button on a queue item

```json
{
  "action": "REMOVE_FROM_QUEUE",
  "payload": {
    "roomId": "abc123",
    "itemId": "q_xyz789"
  }
}
```

**Backend response:**
- Removes item from queue
- Broadcasts QUEUE_UPDATE

---

### SKIP_TO_NEXT (Host Only)

**When:** Host clicks "Skip to Next" button

```json
{
  "action": "SKIP_TO_NEXT",
  "payload": {
    "roomId": "abc123"
  }
}
```

**Backend response:**
1. Pops first item from queue
2. Sets it as CurrentVideo
3. Broadcasts CURRENT_VIDEO_CHANGED
4. Broadcasts QUEUE_UPDATE

---

### CLEAR_QUEUE (Host Only)

**When:** Host clicks "Clear Queue" button

```json
{
  "action": "CLEAR_QUEUE",
  "payload": {
    "roomId": "abc123"
  }
}
```

**Backend response:**
- Empties the queue array
- Broadcasts QUEUE_UPDATE with empty array

---

### EPISODE_ENDED (Auto-advance)

**When:** Video player fires `@ended` event

```json
{
  "action": "EPISODE_ENDED",
  "payload": {
    "roomId": "abc123"
  }
}
```

**Backend response:**
- Same as SKIP_TO_NEXT (auto-advances to next video)

---

## Frontend Implementation

### 1. Update State Management (Composables)

**File:** `src/composables/useRoom.js` or `src/composables/useWebSocket.js`

**Add queue state:**

```javascript
const queue = ref([])

// In WebSocket message handler:
function handleWebSocketMessage(data) {
  switch (data.action) {
    case 'ROOM_INIT':
      // ... existing logic
      queue.value = data.payload.queue || []
      break

    case 'QUEUE_UPDATE':
      queue.value = data.payload.queue || []
      break

    case 'CURRENT_VIDEO_CHANGED':
      currentVideo.value = data.payload.videoUrl
      currentMetadata.value = data.payload.metadata
      currentTime.value = 0
      isPlaying.value = false
      break

    // ... other cases
  }
}

// Queue operations
function addToQueue(url) {
  sendWebSocketMessage({
    action: 'ADD_TO_QUEUE',
    payload: {
      roomId: roomId.value,
      url: url
    }
  })
}

function removeFromQueue(itemId) {
  sendWebSocketMessage({
    action: 'REMOVE_FROM_QUEUE',
    payload: {
      roomId: roomId.value,
      itemId: itemId
    }
  })
}

function skipToNext() {
  sendWebSocketMessage({
    action: 'SKIP_TO_NEXT',
    payload: {
      roomId: roomId.value
    }
  })
}

function clearQueue() {
  sendWebSocketMessage({
    action: 'CLEAR_QUEUE',
    payload: {
      roomId: roomId.value
    }
  })
}

function onVideoEnded() {
  sendWebSocketMessage({
    action: 'EPISODE_ENDED',
    payload: {
      roomId: roomId.value
    }
  })
}

// Export queue and operations
return {
  // ... existing exports
  queue,
  addToQueue,
  removeFromQueue,
  skipToNext,
  clearQueue,
  onVideoEnded
}
```

---

### 2. Queue UI Component

**File:** `src/components/QueueSidebar.vue` (NEW)

**Create a Queue tab next to Chat:**

```vue
<template>
  <div class="queue-sidebar">
    <div class="queue-header">
      <h3>Queue ({{ queue.length }})</h3>
      <button 
        v-if="isHost && queue.length > 0" 
        @click="clearQueue"
        class="btn-clear"
      >
        Clear All
      </button>
    </div>

    <!-- Add to Queue Input -->
    <div class="queue-input">
      <input 
        v-model="newVideoUrl"
        @keyup.enter="handleAddToQueue"
        type="text" 
        placeholder="Paste video URL to add to queue..."
        class="input-field"
      />
      <button @click="handleAddToQueue" class="btn-add">Add</button>
    </div>

    <!-- Queue List -->
    <div class="queue-list">
      <div 
        v-for="item in queue" 
        :key="item.id"
        class="queue-item"
      >
        <img 
          v-if="item.thumbnail" 
          :src="item.thumbnail" 
          alt="Thumbnail"
          class="queue-thumbnail"
        />
        <div class="queue-info">
          <p class="queue-title">{{ item.title || 'Loading...' }}</p>
          <small class="queue-episode">{{ item.episode || item.url }}</small>
        </div>
        <button 
          v-if="isHost" 
          @click="removeFromQueue(item.id)"
          class="btn-remove"
        >
          ×
        </button>
      </div>

      <div v-if="queue.length === 0" class="queue-empty">
        <p>Queue is empty</p>
        <small>Add videos to create a playlist</small>
      </div>
    </div>

    <!-- Host Controls -->
    <div v-if="isHost && queue.length > 0" class="queue-controls">
      <button @click="skipToNext" class="btn-skip">
        Skip to Next Video
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoom } from '@/composables/useRoom'

const { 
  queue, 
  isHost, 
  addToQueue, 
  removeFromQueue, 
  skipToNext, 
  clearQueue 
} = useRoom()

const newVideoUrl = ref('')

function handleAddToQueue() {
  if (newVideoUrl.value.trim()) {
    addToQueue(newVideoUrl.value.trim())
    newVideoUrl.value = ''
  }
}
</script>

<style scoped>
.queue-sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a1a;
  color: white;
}

.queue-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid #333;
}

.queue-header h3 {
  margin: 0;
  font-size: 1.25rem;
}

.btn-clear {
  padding: 0.5rem 1rem;
  background: #dc2626;
  border: none;
  border-radius: 6px;
  color: white;
  cursor: pointer;
  font-size: 0.875rem;
}

.btn-clear:hover {
  background: #b91c1c;
}

.queue-input {
  display: flex;
  gap: 0.5rem;
  padding: 1rem;
  border-bottom: 1px solid #333;
}

.input-field {
  flex: 1;
  padding: 0.75rem;
  background: #2a2a2a;
  border: 1px solid #444;
  border-radius: 6px;
  color: white;
  font-size: 0.875rem;
}

.input-field:focus {
  outline: none;
  border-color: #667eea;
}

.btn-add {
  padding: 0.75rem 1.5rem;
  background: #667eea;
  border: none;
  border-radius: 6px;
  color: white;
  cursor: pointer;
  font-weight: 600;
}

.btn-add:hover {
  background: #5a67d8;
}

.queue-list {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
}

.queue-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: #2a2a2a;
  border-radius: 8px;
  margin-bottom: 0.75rem;
  transition: background 0.2s;
}

.queue-item:hover {
  background: #333;
}

.queue-thumbnail {
  width: 60px;
  height: 60px;
  object-fit: cover;
  border-radius: 6px;
  flex-shrink: 0;
}

.queue-info {
  flex: 1;
  min-width: 0;
}

.queue-title {
  margin: 0 0 0.25rem;
  font-weight: 600;
  font-size: 0.9rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.queue-episode {
  color: #999;
  font-size: 0.8rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

.btn-remove {
  width: 28px;
  height: 28px;
  background: #dc2626;
  border: none;
  border-radius: 50%;
  color: white;
  font-size: 1.25rem;
  cursor: pointer;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.btn-remove:hover {
  background: #b91c1c;
}

.queue-empty {
  text-align: center;
  padding: 3rem 1rem;
  color: #666;
}

.queue-empty p {
  margin: 0 0 0.5rem;
  font-size: 1rem;
}

.queue-empty small {
  font-size: 0.875rem;
}

.queue-controls {
  padding: 1rem;
  border-top: 1px solid #333;
}

.btn-skip {
  width: 100%;
  padding: 0.875rem;
  background: #10b981;
  border: none;
  border-radius: 6px;
  color: white;
  font-weight: 600;
  cursor: pointer;
}

.btn-skip:hover {
  background: #059669;
}
</style>
```

---

### 3. Video Player Auto-advance

**File:** `src/components/VideoPlayer.vue`

**Add @ended event listener:**

```vue
<template>
  <video
    ref="videoElement"
    @ended="handleVideoEnded"
    @timeupdate="handleTimeUpdate"
    @play="handlePlay"
    @pause="handlePause"
    ...
  >
  </video>
</template>

<script setup>
import { useRoom } from '@/composables/useRoom'

const { onVideoEnded } = useRoom()

function handleVideoEnded() {
  onVideoEnded()
}
</script>
```

---

### 4. Integrate Queue Tab

**File:** `src/views/RoomView.vue` or wherever your sidebar tabs are

**Add Queue tab next to Chat:**

```vue
<template>
  <div class="room-view">
    <div class="main-content">
      <VideoPlayer />
    </div>
    
    <div class="sidebar">
      <div class="tabs">
        <button 
          @click="activeTab = 'chat'" 
          :class="{ active: activeTab === 'chat' }"
        >
          Chat
        </button>
        <button 
          @click="activeTab = 'queue'" 
          :class="{ active: activeTab === 'queue' }"
        >
          Queue
        </button>
      </div>

      <ChatSidebar v-if="activeTab === 'chat'" />
      <QueueSidebar v-if="activeTab === 'queue'" />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import ChatSidebar from '@/components/ChatSidebar.vue'
import QueueSidebar from '@/components/QueueSidebar.vue'
import VideoPlayer from '@/components/VideoPlayer.vue'

const activeTab = ref('chat')
</script>
```

---

## Testing Checklist

After implementing:

- [ ] **Add to queue**: Paste an Otakudesu URL → should appear in queue with title/episode/thumbnail
- [ ] **Multiple items**: Add 3-4 videos → queue should display all items in order
- [ ] **Remove item** (host): Click remove button → item disappears, QUEUE_UPDATE received
- [ ] **Skip to next** (host): Click "Skip to Next" → current video changes, queue shifts
- [ ] **Auto-advance**: Let a video play to completion → automatically advances to next in queue
- [ ] **Clear queue** (host): Click "Clear All" → queue empties
- [ ] **Non-host permissions**: As non-host, should be able to add but NOT remove/skip/clear
- [ ] **New joiner**: Join room with existing queue → ROOM_INIT includes queue array

---

## Summary

**Backend provides:**
- Pre-scraped metadata for queue items
- Unique IDs for safe removal
- Auto-advance when video ends
- Permission checks (host-only operations)

**Frontend displays:**
- Beautiful queue list with thumbnails
- Add input (anyone)
- Remove/skip/clear controls (host only)
- Auto-advance on video end

**User Experience:**
1. User pastes anime URL → backend scrapes → queue item appears with title/thumbnail
2. Host clicks "Skip to Next" → next video loads instantly
3. Video ends → automatically plays next in queue
4. Queue tab shows upcoming episodes with rich metadata

---

**Implementation ready. Integrate with existing room/video components.**
