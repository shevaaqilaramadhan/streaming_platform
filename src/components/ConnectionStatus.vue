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
  background: rgba(92, 184, 92, 0.1);
  border-color: rgba(92, 184, 92, 0.2);
  color: hsl(120, 40%, 60%);
}
.status--connected .dot {
  background: hsl(120, 40%, 60%);
}

/* Connecting / Reconnecting */
.status--connecting {
  background: rgba(240, 173, 78, 0.08);
  border-color: rgba(240, 173, 78, 0.15);
  color: hsl(38, 70%, 60%);
}
.status--connecting .dot {
  background: hsl(38, 70%, 60%);
  animation: blink 1s step-start infinite;
}

/* Disconnected */
.status--disconnected {
  background: rgba(217, 83, 79, 0.08);
  border-color: rgba(217, 83, 79, 0.15);
  color: hsl(0, 60%, 65%);
}
.status--disconnected .dot {
  background: hsl(0, 60%, 65%);
}

@keyframes blink {
  50% { opacity: 0; }
}
</style>
