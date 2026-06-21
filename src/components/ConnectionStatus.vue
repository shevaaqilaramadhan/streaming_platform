<template>
  <div class="connection-status" :class="statusClass" :data-tooltip="statusLabel">
    <span class="dot"></span>
    <span class="label">{{ statusLabel }}</span>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: {
    type: String,
    required: true,
  },
})

const statusClass = computed(() => ({
  'status--connected':    props.status === 'connected',
  'status--connecting':   props.status === 'connecting' || props.status === 'reconnecting',
  'status--disconnected': props.status === 'disconnected',
}))

const statusLabel = computed(() => {
  switch (props.status) {
    case 'connected':    return 'Live'
    case 'connecting':   return 'Connecting…'
    case 'reconnecting': return 'Reconnecting…'
    default:             return 'Offline'
  }
})
</script>

<style scoped>
.connection-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 600;
  border: 1px solid transparent;
  transition: all var(--transition-base);
}

.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.label {
  letter-spacing: 0.04em;
}

/* Connected */
.status--connected {
  background: rgba(52, 211, 153, 0.12);
  border-color: rgba(52, 211, 153, 0.3);
  color: hsl(158, 64%, 62%);
}
.status--connected .dot {
  background: hsl(158, 64%, 62%);
  animation: pulse-glow-green 2s infinite;
}

/* Connecting / Reconnecting */
.status--connecting {
  background: rgba(251, 191, 36, 0.1);
  border-color: rgba(251, 191, 36, 0.25);
  color: hsl(38, 95%, 58%);
}
.status--connecting .dot {
  background: hsl(38, 95%, 58%);
  animation: blink 1s step-start infinite;
}

/* Disconnected */
.status--disconnected {
  background: rgba(248, 113, 113, 0.1);
  border-color: rgba(248, 113, 113, 0.2);
  color: hsl(0, 75%, 65%);
}
.status--disconnected .dot {
  background: hsl(0, 75%, 65%);
}

@keyframes pulse-glow-green {
  0%, 100% { box-shadow: 0 0 0 0 rgba(52,211,153,0.5); }
  50%       { box-shadow: 0 0 0 5px rgba(52,211,153,0); }
}
@keyframes blink {
  50% { opacity: 0; }
}
</style>
