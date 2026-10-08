<template>
  <aside class="chat-panel glass">
    <!-- Panel Header -->
    <div class="chat-header">
      <h2 class="chat-title">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
        Live Chat
      </h2>
      <span class="message-count">{{ messages.length }}</span>
    </div>

    <!-- Message List -->
    <div class="messages-list" ref="messagesEl" role="log" aria-live="polite" aria-label="Chat messages">
      <TransitionGroup name="slide-up" tag="div" class="messages-inner">
        <!-- Empty state -->
        <div v-if="messages.length === 0" key="empty" class="chat-empty">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          <p>No messages yet. Say hi!</p>
        </div>

        <!-- Messages -->
        <div
          v-for="msg in messages"
          :key="msg.id"
          class="message"
          :class="{
            'message--own':    msg.isOwn,
            'message--system': msg.type === 'system',
          }"
        >
          <!-- System messages -->
          <template v-if="msg.type === 'system'">
            <div class="system-msg">
              <span class="system-icon">—</span>
              <span>{{ msg.text }}</span>
            </div>
          </template>

          <!-- User messages -->
          <template v-else>
            <div class="message-bubble">
              <span class="message-author" :style="{ color: getUserColor(msg.username) }">
                {{ msg.username }}
                <span v-if="msg.isOwn" class="you-label">(you)</span>
              </span>
              <p class="message-text">{{ msg.text }}</p>
            </div>
            <span class="message-time">{{ formatTime(msg.sentAt) }}</span>
          </template>
        </div>
      </TransitionGroup>
    </div>

    <!-- Input area -->
    <div class="typing-indicator" v-if="typingDisplay">
      <span class="typing-dot"></span>
      <span class="typing-text">{{ typingDisplay }}</span>
    </div>

    <div class="chat-input-area" :class="{ 'chat-input-area--disabled': !isConnected }">
      <!-- Reaction picker -->
      <div class="react-popover-wrap">
        <button
          type="button"
          class="btn btn-ghost react-toggle-btn"
          :class="{ 'react-toggle-btn--open': showEmojiPicker }"
          :disabled="!isConnected"
          @click="showEmojiPicker = !showEmojiPicker"
          title="React with emoji"
          aria-label="Send emoji reaction"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
            <line x1="9" y1="9" x2="9.01" y2="9"/>
            <line x1="15" y1="9" x2="15.01" y2="9"/>
          </svg>
        </button>

        <Transition name="fade">
          <div v-if="showEmojiPicker" class="chat-emoji-popover glass-strong">
            <button
              v-for="emoji in EMOJIS"
              :key="emoji"
              type="button"
              class="emoji-popover-item"
              @click="handleSelectEmoji(emoji)"
              :aria-label="'React ' + emoji"
            >
              {{ emoji }}
            </button>
          </div>
        </Transition>
      </div>

      <input
        id="chat-input"
        ref="inputRef"
        v-model="inputText"
        type="text"
        class="input chat-input"
        placeholder="Send a message…"
        maxlength="300"
        :disabled="!isConnected"
        @keydown.enter="sendMessage"
        @input="onInput"
        aria-label="Chat message input"
      />
      <button
        id="send-chat-btn"
        class="btn btn-primary send-btn"
        :disabled="!inputText.trim() || !isConnected"
        @click="sendMessage"
        aria-label="Send message"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="22" y1="2" x2="11" y2="13"/>
          <polygon points="22 2 15 22 11 13 2 9 22 2"/>
        </svg>
      </button>
    </div>
  </aside>
</template>

<script setup>
import { ref, watch, nextTick, computed } from 'vue'

const EMOJIS = ['😂', '🔥', '👏', '❤️', '😮', '🎉']

const props = defineProps({
  messages:     { type: Array,   default: () => [] },
  isConnected:  { type: Boolean, default: false },
  nickname:     { type: String,  required: true },
  typingUsers:  { type: Array,   default: () => [] },
})

const emit = defineEmits(['send', 'typing', 'react'])

const inputText       = ref('')
const messagesEl      = ref(null)
const inputRef        = ref(null)
const showEmojiPicker = ref(false)
let typingDebounce    = null

function handleSelectEmoji(emoji) {
  emit('react', emoji)
  showEmojiPicker.value = false
}

function isNearBottom(el, threshold = 120) {
  return el.scrollHeight - el.scrollTop - el.clientHeight < threshold
}

/* Auto-scroll: check near-bottom before DOM grows, then scroll after nextTick */
watch(
  () => props.messages.length,
  async () => {
    const el = messagesEl.value
    if (!el) return
    const wasNearBottom = isNearBottom(el)
    await nextTick()
    if (wasNearBottom) {
      el.scrollTop = el.scrollHeight
    }
  }
)

/* Debounce typing event */
function onInput() {
  clearTimeout(typingDebounce)
  typingDebounce = setTimeout(() => {
    emit('typing')
  }, 300)
}

function sendMessage() {
  const text = inputText.value.trim()
  if (!text || !props.isConnected) return
  emit('send', text)
  inputText.value = ''
  inputRef.value?.focus()
  clearTimeout(typingDebounce)
  /* Always scroll to bottom when user sends a message */
  nextTick(() => {
    const el = messagesEl.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

/* Deterministic colour per username */
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

function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

/* Typing indicator display */
const typingDisplay = computed(() => {
  const users = props.typingUsers
  if (!users || users.length === 0) return ''
  if (users.length === 1) return `${users[0].username} is typing…`
  if (users.length === 2) return `${users[0].username} and ${users[1].username} are typing…`
  return `${users[0].username} and ${users.length - 1} others are typing…`
})
</script>

<style scoped>
.chat-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  border-radius: var(--radius-lg);
  overflow: hidden;
}

/* Header */
.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.chat-title {
  display: flex; align-items: center; gap: var(--space-2);
  font-size: 0.875rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-secondary);
}
.message-count {
  font-size: 0.75rem;
  color: var(--text-muted);
  background: rgba(255,255,255,0.05);
  border-radius: var(--radius-full);
  padding: 1px 8px;
}

/* Messages list */
.messages-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4) var(--space-4) 0;
  min-height: 0;
}
.messages-inner {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-bottom: var(--space-4);
}

.chat-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-10) var(--space-4);
  color: var(--text-muted);
  text-align: center;
  font-size: 0.875rem;
}

/* Message */
.message {
  display: flex;
  flex-direction: column;
  gap: 3px;
  animation: fadeIn 0.25s ease both;
}

.message-bubble {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 14px 14px 14px 4px;
  padding: var(--space-2) var(--space-3);
  max-width: 85%;
  transition: all var(--transition-base);
}
.message--own .message-bubble {
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(255, 255, 255, 0.08);
  border-radius: 14px 14px 4px 14px;
  align-self: flex-end;
}
.message--own { align-items: flex-end; }

.message-author {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.02em;
}
.you-label {
  font-weight: 400;
  color: var(--text-muted);
  margin-left: 3px;
}
.message-text {
  font-size: 0.875rem;
  color: var(--text-primary);
  word-break: break-word;
  line-height: 1.5;
  margin-top: 2px;
}
.message-time {
  font-size: 0.6875rem;
  color: var(--text-muted);
  padding: 0 var(--space-1);
}

/* System messages */
.system-msg {
  display: flex; align-items: center; gap: var(--space-2);
  color: var(--text-muted);
  font-size: 0.75rem;
  font-style: italic;
  text-align: center;
  justify-content: center;
}
.system-icon { color: var(--text-muted); opacity: 0.5; }

/* Input */
.typing-indicator {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-4);
  font-size: 0.75rem;
  color: var(--text-muted);
  font-style: italic;
  min-height: 22px;
}

.typing-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-muted);
  animation: typing-pulse 1.2s ease-in-out infinite;
}

.typing-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

@keyframes typing-pulse {
  0%, 100% { opacity: 0.4; transform: scale(0.8); }
  50% { opacity: 1; transform: scale(1); }
}

.chat-input-area {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}
.chat-input {
  flex: 1;
  border-radius: var(--radius-md);
  padding: var(--space-2) var(--space-3);
  font-size: 0.875rem;
}
.chat-input-area--disabled .chat-input {
  opacity: 0.4;
  cursor: not-allowed;
}

.send-btn {
  width: 40px; height: 40px;
  padding: 0;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}

.react-popover-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.react-toggle-btn {
  width: 38px;
  height: 38px;
  padding: 0;
  border-radius: var(--radius-md);
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}
.react-toggle-btn:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.08);
}
.react-toggle-btn--open {
  color: var(--color-primary);
  background: rgba(255, 255, 255, 0.1);
}

.chat-emoji-popover {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 0;
  display: flex;
  gap: 4px;
  padding: 6px 8px;
  border-radius: var(--radius-lg);
  background: rgba(18, 18, 20, 0.95);
  border: 1px solid var(--color-border);
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.5);
  z-index: 50;
}

.emoji-popover-item {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  font-size: 1.2rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform var(--transition-fast), background var(--transition-fast);
}
.emoji-popover-item:hover {
  background: rgba(255, 255, 255, 0.1);
  transform: scale(1.2);
}
</style>
