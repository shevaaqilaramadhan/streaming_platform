# Frontend Metadata Display Task — WatchParty Vue Frontend

> **For the frontend agent**: The backend now broadcasts video metadata (anime title, episode number, thumbnail) along with the stream URL. This document specifies how to display "Now Watching: Attack on Titan - Episode 12" above the video player.

---

## Overview

The Go backend scraper now extracts metadata from anime pages (Otakudesu and generic sites) and broadcasts it via WebSocket. The frontend needs to:

1. **Store metadata** in room state
2. **Display "Now Watching" header** with anime title, episode, and thumbnail
3. **Handle fallback** when metadata is unavailable (show stream source domain)

---

## Backend Changes (Already Implemented)

The backend now sends this payload structure for `SET_VIDEO` and `ROOM_INIT` events:

### SET_VIDEO Event Payload

```json
{
  "action": "SET_VIDEO",
  "payload": {
    "roomId": "abc123",
    "videoUrl": "https://cdn.example.com/stream.m3u8",
    "metadata": {
      "videoUrl": "https://cdn.example.com/stream.m3u8",
      "title": "Chainsaw Man",
      "episode": "Episode 12",
      "thumbnailUrl": "https://otakudesu.blog/wp-content/uploads/2023/chainsaw-man.jpg",
      "source": "otakudesu.blog"
    }
  }
}
```

### ROOM_INIT Event Payload

```json
{
  "action": "ROOM_INIT",
  "payload": {
    "roomId": "abc123",
    "currentVideo": "https://cdn.example.com/stream.m3u8",
    "currentTime": 142.35,
    "isPlaying": true,
    "participants": [...],
    "metadata": {
      "videoUrl": "https://cdn.example.com/stream.m3u8",
      "title": "Chainsaw Man",
      "episode": "Episode 12",
      "thumbnailUrl": "https://otakudesu.blog/wp-content/uploads/2023/chainsaw-man.jpg",
      "source": "otakudesu.blog"
    }
  }
}
```

### Metadata Fields

| Field | Type | Description | Example |
|---|---|---|---|
| `videoUrl` | `string` | The resolved stream URL | `"https://cdn.example.com/stream.m3u8"` |
| `title` | `string` (optional) | Anime title | `"Chainsaw Man"` |
| `episode` | `string` (optional) | Episode identifier | `"Episode 12"` |
| `thumbnailUrl` | `string` (optional) | Cover art URL | `"https://..."` |
| `source` | `string` (optional) | Domain fallback | `"otakudesu.blog"` |

**Fallback behavior:**
- If `title` is missing → show `source` as "[Stream Source: otakudesu.blog]"
- If `thumbnailUrl` is missing → omit image
- If `episode` is missing → show only title

---

## Frontend Implementation

### 1. Update Room State (Composables)

**File:** `src/composables/useRoom.js` or similar

**Add metadata field to room state:**

```javascript
const currentMetadata = ref(null)

// In the WebSocket message handler:
function handleWebSocketMessage(data) {
  switch (data.action) {
    case 'ROOM_INIT':
      currentVideo.value = data.payload.currentVideo
      currentTime.value = data.payload.currentTime
      isPlaying.value = data.payload.isPlaying
      participants.value = data.payload.participants
      currentMetadata.value = data.payload.metadata || null // NEW
      break

    case 'SET_VIDEO':
      currentVideo.value = data.payload.videoUrl
      currentMetadata.value = data.payload.metadata || null // NEW
      currentTime.value = 0
      isPlaying.value = false
      break

    // ... other cases
  }
}

// Export metadata
return {
  // ... existing exports
  currentMetadata
}
```

---

### 2. Display Component (VideoPlayer or RoomView)

**File:** `src/components/VideoPlayer.vue` or `src/views/RoomView.vue`

**Add "Now Watching" header above the video player:**

```vue
<template>
  <div class="video-player-container">
    <!-- Now Watching Header -->
    <div v-if="currentMetadata" class="now-watching-header">
      <img 
        v-if="currentMetadata.thumbnailUrl" 
        :src="currentMetadata.thumbnailUrl" 
        alt="Anime Cover"
        class="anime-thumbnail"
      />
      <div class="metadata-info">
        <h2 v-if="currentMetadata.title" class="anime-title">
          Now Watching: {{ currentMetadata.title }}
        </h2>
        <p v-if="currentMetadata.episode" class="episode-info">
          {{ currentMetadata.episode }}
        </p>
        <small v-else-if="currentMetadata.source" class="stream-source">
          Stream Source: {{ currentMetadata.source }}
        </small>
      </div>
    </div>

    <!-- Video Player -->
    <div class="video-wrapper">
      <!-- Your existing VideoPlayer component -->
    </div>
  </div>
</template>

<script setup>
import { useRoom } from '@/composables/useRoom'

const { currentMetadata } = useRoom()
</script>

<style scoped>
.now-watching-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 8px;
  margin-bottom: 1rem;
  color: white;
}

.anime-thumbnail {
  width: 80px;
  height: 80px;
  object-fit: cover;
  border-radius: 8px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
}

.metadata-info {
  flex: 1;
}

.anime-title {
  margin: 0;
  font-size: 1.5rem;
  font-weight: bold;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.episode-info {
  margin: 0.25rem 0 0;
  font-size: 1rem;
  opacity: 0.9;
}

.stream-source {
  font-size: 0.85rem;
  opacity: 0.7;
  font-style: italic;
}
</style>
```

---

## Testing Checklist

After implementing the frontend changes:

- [ ] **Otakudesu URL**: Paste `https://otakudesu.blog/episode/gcms-episode-8-sub-indo/`
  - Should display: Title, Episode, Thumbnail
- [ ] **YouTube URL**: Paste a YouTube link
  - Should display: "[Stream Source: youtube.com]"
- [ ] **Direct .m3u8 URL**: Paste a direct stream URL
  - Should display: "[Stream Source: cdn.example.com]"
- [ ] **New joiner**: Have another user join the room
  - They should see the same "Now Watching" header via ROOM_INIT

---

## Summary

**Backend broadcasts:**
```json
{
  "metadata": {
    "title": "Anime Title",
    "episode": "Episode 12",
    "thumbnailUrl": "https://...",
    "source": "otakudesu.blog"
  }
}
```

**Frontend displays:**
```
┌─────────────────────────────────────┐
│ [Thumbnail] Now Watching: Anime     │
│             Episode 12              │
└─────────────────────────────────────┘
```

**Fallback (no metadata):**
```
┌─────────────────────────────────────┐
│ Stream Source: otakudesu.blog       │
└─────────────────────────────────────┘
```

---

**Implementation complete. Ready for frontend agent to integrate.**
