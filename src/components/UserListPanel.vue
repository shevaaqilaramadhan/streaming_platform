<template>
  <Transition name="fade">
    <div v-if="visible" class="user-list-overlay" @click.self="$emit('close')">
      <div class="user-list-panel glass animate-fade-in-scale">
        <!-- Header -->
        <div class="panel-header">
          <h3 class="panel-title">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
              <circle cx="9" cy="7" r="4"/>
              <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
              <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>
            Participants
            <span class="count-badge">{{ participants.length }}</span>
          </h3>
          <button class="close-btn" @click="$emit('close')" aria-label="Close">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>

        <!-- List -->
        <ul class="user-list" role="list">
          <li
            v-for="user in participants"
            :key="user.userId"
            class="user-item"
            :class="{ 'user-item--self': user.userId === localUserId }"
          >
            <div class="user-info">
              <div class="user-avatar" :style="{ background: getUserColor(user.username) }">
                {{ (user.username || '?')[0].toUpperCase() }}
              </div>
              <span class="user-name">
                {{ user.username }}
                <span v-if="user.userId === localUserId" class="you-tag">(you)</span>
              </span>
              <span v-if="user.userId === hostId" class="host-badge">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                </svg>
                Host
              </span>
            </div>

            <!-- Actions: host-only, not self, not current host target for kick is OK; transfer to anyone else -->
            <div v-if="isHost && user.userId && user.userId !== localUserId" class="user-actions">
              <button
                type="button"
                class="action-btn action-btn--transfer"
                title="Make host"
                aria-label="Transfer host to this user"
                @click.stop="$emit('transfer-host', user.userId)"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                </svg>
                <span class="action-label">Host</span>
              </button>
              <button
                v-if="user.userId !== hostId"
                type="button"
                class="action-btn action-btn--kick"
                title="Kick user"
                aria-label="Kick this user"
                @click.stop="$emit('kick-user', user.userId)"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M18 6L6 18M6 6l12 12"/>
                </svg>
                <span class="action-label">Kick</span>
              </button>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </Transition>
</template>

<script setup>
defineProps({
  visible:      { type: Boolean, default: false },
  participants: { type: Array,   default: () => [] },
  hostId:       { type: String,  default: '' },
  localUserId:  { type: String,  default: '' },
  isHost:       { type: Boolean, default: false },
})

defineEmits(['close', 'transfer-host', 'kick-user'])

const USER_COLORS = [
  'hsl(330,70%,72%)', 'hsl(200,80%,65%)', 'hsl(160,60%,60%)',
  'hsl(30,90%,65%)',  'hsl(340,70%,68%)', 'hsl(50,90%,62%)',
  'hsl(290,65%,70%)', 'hsl(180,65%,60%)',
]

function getUserColor(username) {
  let hash = 0
  for (const ch of (username || '?')) hash = (hash * 31 + ch.charCodeAt(0)) | 0
  return USER_COLORS[Math.abs(hash) % USER_COLORS.length]
}
</script>

<style scoped>
.user-list-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  background: rgba(7, 7, 15, 0.4);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  padding: 60px var(--space-4) var(--space-4);
}

.user-list-panel {
  width: 320px;
  max-height: 400px;
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6), var(--shadow-glow-sm);
}

/* Header */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-glass-border);
  flex-shrink: 0;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 0.85rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-secondary);
  margin: 0;
}

.count-badge {
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.05);
  border-radius: var(--radius-full);
  padding: 1px 6px;
}

.close-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
}
.close-btn:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.05);
}

/* User list */
.user-list {
  list-style: none;
  padding: var(--space-2) 0;
  margin: 0;
  overflow-y: auto;
  flex: 1;
}

.user-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--space-4);
  transition: background var(--transition-fast);
}
.user-item:hover {
  background: rgba(255, 255, 255, 0.03);
}
.user-item--self {
  background: rgba(255, 255, 255, 0.02);
}

.user-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.user-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 700;
  color: white;
  flex-shrink: 0;
}

.user-name {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.you-tag {
  font-weight: 400;
  color: var(--text-muted);
  font-size: 0.75rem;
  margin-left: 2px;
}

.host-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 6px;
  border-radius: var(--radius-full);
  font-size: 0.65rem;
  font-weight: 700;
  color: var(--color-primary);
  background: var(--grad-brand-subtle);
  border: 1px solid hsla(195, 100%, 45%, 0.2);
  flex-shrink: 0;
}

/* Actions */
.user-actions {
  display: flex;
  gap: var(--space-1);
  flex-shrink: 0;
}

.action-btn {
  min-width: 28px;
  height: 28px;
  padding: 0 8px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: 1px solid var(--color-glass-border);
  color: var(--text-muted);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: all var(--transition-fast);
  font-size: 0.7rem;
  font-weight: 600;
}
.action-label {
  line-height: 1;
}
.action-btn:hover {
  background: var(--color-glass-hover);
  border-color: var(--color-glass-border);
  color: var(--text-primary);
}

.action-btn--transfer:hover {
  color: var(--color-primary);
  border-color: hsla(195, 100%, 45%, 0.3);
}

.action-btn--kick:hover {
  color: var(--color-accent-red);
  border-color: hsla(0, 75%, 55%, 0.3);
  background: rgba(239, 68, 68, 0.08);
}

/* Mobile */
@media (max-width: 480px) {
  .user-list-overlay {
    padding: 52px var(--space-3) var(--space-3);
    justify-content: center;
  }
  .user-list-panel {
    width: 100%;
    max-width: 340px;
  }
}
</style>
