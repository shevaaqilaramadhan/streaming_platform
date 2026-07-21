<template>
  <aside class="queue-panel glass">
    <!-- Panel Header -->
    <div class="queue-header">
      <h2 class="queue-title">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="8" y1="6" x2="21" y2="6"/>
          <line x1="8" y1="12" x2="21" y2="12"/>
          <line x1="8" y1="18" x2="21" y2="18"/>
          <line x1="3" y1="6" x2="3.01" y2="6"/>
          <line x1="3" y1="12" x2="3.01" y2="12"/>
          <line x1="3" y1="18" x2="3.01" y2="18"/>
        </svg>
        Play Queue
      </h2>
      <div class="header-actions">
        <span class="item-count">{{ queue.length }}</span>
        <button 
          v-if="isHost && queue.length > 0" 
          @click="emit('clear')"
          class="btn-clear-all"
          title="Clear all queue items"
        >
          Clear All
        </button>
      </div>
    </div>

    <!-- Queue List -->
    <div class="queue-list" role="log" aria-label="Play queue list">
      <TransitionGroup name="slide-up" tag="div" class="queue-inner">
        <!-- Empty state -->
        <div v-if="queue.length === 0" key="empty" class="queue-empty">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round">
            <rect width="18" height="18" x="3" y="3" rx="2"/>
            <path d="M12 8v8"/>
            <path d="M8 12h8"/>
          </svg>
          <p class="queue-empty-title">Queue is empty</p>
          <small>Add video URLs below to build a playlist — they'll play in order automatically</small>
        </div>

        <!-- Queue Items -->
        <div
          v-for="(item, index) in queue"
          :key="item.id"
          class="queue-item"
          :class="{ 'queue-item--first': index === 0 }"
        >
          <div class="queue-item-card">
            <!-- Thumbnail cover -->
            <div class="queue-thumbnail-wrapper">
              <img 
                v-if="item.thumbnail" 
                :src="item.thumbnail" 
                alt="Cover"
                class="queue-thumbnail"
                referrerpolicy="no-referrer"
              />
              <div v-else class="queue-thumbnail-fallback">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polygon points="5 3 19 12 5 21 5 3"/>
                </svg>
              </div>
              <span class="queue-index">#{{ index + 1 }}</span>
            </div>

            <!-- Info -->
            <div class="queue-info">
              <h4 class="queue-title-text" :title="item.title || 'Loading...'">
                {{ item.title || 'Resolving video details...' }}
              </h4>
              <p class="queue-subtitle-text" :title="item.episode || item.url">
                {{ item.episode || getDomain(item.url) }}
              </p>
            </div>

            <!-- Remove Button (Host Only) -->
            <button
              v-if="isHost"
              class="btn-remove"
              @click="emit('remove', item.id)"
              title="Remove from queue"
              aria-label="Remove item"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
        </div>
      </TransitionGroup>
    </div>

    <!-- Skip Host Controls -->
    <div v-if="isHost && queue.length > 0" class="queue-controls">
      <button @click="emit('skip')" class="btn btn-primary skip-btn">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="5 4 15 12 5 20 5 4"/>
          <line x1="19" y1="5" x2="19" y2="19"/>
        </svg>
        Skip to Next Video
      </button>
    </div>

    <!-- Input area -->
    <div class="queue-input-area">
      <input
        id="queue-input"
        v-model="newVideoUrl"
        type="text"
        class="input queue-input"
        placeholder="Paste video or anime URL..."
        @keydown.enter="handleAddToQueue"
        aria-label="Queue video input"
      />
      <button
        id="add-to-queue-btn"
        class="btn btn-primary add-btn"
        :disabled="!newVideoUrl.trim()"
        @click="handleAddToQueue"
        aria-label="Add to queue"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
      </button>
    </div>
  </aside>
</template>

<script setup>
import { ref } from 'vue'
import { isValidVideoInput } from '../utils/videoInput.js'

const props = defineProps({
  queue:  { type: Array,   default: () => [] },
  isHost: { type: Boolean, default: false },
})

const emit = defineEmits(['add', 'remove', 'skip', 'clear'])

const newVideoUrl = ref('')

function handleAddToQueue() {
  const url = newVideoUrl.value.trim()
  if (!url) return
  if (!isValidVideoInput(url)) return
  emit('add', url)
  newVideoUrl.value = ''
}

function getDomain(url) {
  try {
    return new URL(url).hostname
  } catch (e) {
    return url
  }
}
</script>

<style scoped>
.queue-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  border-radius: var(--radius-lg);
  overflow: hidden;
}

/* Header */
.queue-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.queue-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 0.875rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-secondary);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.item-count {
  font-size: 0.75rem;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.05);
  border-radius: var(--radius-full);
  padding: 1px 8px;
}

.btn-clear-all {
  background: transparent;
  color: var(--color-accent-red);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: var(--radius-sm);
  padding: 2px 8px;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-clear-all:hover {
  background: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.4);
}

/* Queue list */
.queue-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4) var(--space-4) 0;
  min-height: 0;
}

.queue-inner {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-bottom: var(--space-4);
}

.queue-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-10) var(--space-4);
  color: var(--text-muted);
  text-align: center;
  font-size: 0.875rem;
}

.queue-empty-title {
  font-weight: 600;
  color: var(--text-secondary);
}

.queue-empty small {
  color: var(--text-muted);
  opacity: 0.7;
  max-width: 260px;
  line-height: 1.4;
}

/* Queue Item Card */
.queue-item {
  animation: fadeIn 0.25s ease both;
}

.queue-item-card {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: var(--radius-md);
  padding: var(--space-2) var(--space-3);
  transition: all var(--transition-base);
}

.queue-item-card:hover {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.08);
}

.queue-item--first .queue-item-card {
  border-color: rgba(255,255,255,0.1);
  background: rgba(255,255,255,0.04);
}

.queue-item--first .queue-item-card:hover {
  background: rgba(255,255,255,0.06);
}

/* Thumbnail */
.queue-thumbnail-wrapper {
  position: relative;
  width: 52px;
  height: 68px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--color-border);
  background: #000;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.queue-thumbnail {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.queue-thumbnail-fallback {
  color: var(--text-muted);
}

.queue-index {
  position: absolute;
  top: 2px;
  left: 2px;
  background: rgba(0, 0, 0, 0.7);
  color: var(--text-primary);
  font-size: 0.65rem;
  font-weight: 700;
  padding: 1px 4px;
  border-radius: 3px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

/* Info */
.queue-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.queue-title-text {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.queue-subtitle-text {
  font-size: 0.75rem;
  color: var(--text-secondary);
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Remove button */
.btn-remove {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.btn-remove:hover {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-accent-red);
}

/* Host controls */
.queue-controls {
  padding: var(--space-3) var(--space-4) 0;
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}

.skip-btn {
  width: 100%;
  font-size: 0.875rem;
  padding: var(--space-3);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
}

/* Input area */
.queue-input-area {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}

.queue-input {
  flex: 1;
  border-radius: var(--radius-md);
  padding: var(--space-2) var(--space-3);
  font-size: 0.875rem;
}

.add-btn {
  width: 40px;
  height: 40px;
  padding: 0;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}
</style>
