<template>
  <main class="status-page">
    <!-- Nav -->
    <nav class="home-nav">
      <router-link to="/" class="nav-logo">
        <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
          <circle cx="14" cy="14" r="14" fill="url(#status-logo-grad)"/>
          <polygon points="11,9 21,14 11,19" fill="white"/>
          <defs>
            <linearGradient id="status-logo-grad" x1="0" y1="0" x2="28" y2="28">
              <stop offset="0%" stop-color="hsl(195,100%,45%)"/>
              <stop offset="100%" stop-color="hsl(215,90%,50%)"/>
            </linearGradient>
          </defs>
        </svg>
        <span class="nav-brand">WatchParty</span>
      </router-link>
      <button
        class="hamburger-btn"
        aria-label="Toggle navigation menu"
        :aria-expanded="mobileMenuOpen"
        @click="mobileMenuOpen = !mobileMenuOpen"
      >
        <svg v-if="!mobileMenuOpen" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/>
        </svg>
        <svg v-else width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
      <nav class="nav-links" :class="{ 'nav-links--open': mobileMenuOpen }" aria-label="Main">
        <router-link to="/docs" class="nav-link" @click="mobileMenuOpen = false">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z"/>
            <path d="M14 2v5a1 1 0 0 0 1 1h5"/>
            <path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/>
          </svg>
          Docs
        </router-link>
        <router-link to="/faq" class="nav-link" @click="mobileMenuOpen = false">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/>
          </svg>
          FAQ
        </router-link>
        <router-link to="/status" class="nav-link active">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
          </svg>
          Status
        </router-link>
        <a href="https://github.com/litcq/streaming_platform" target="_blank" rel="noopener noreferrer" class="nav-link" @click="mobileMenuOpen = false">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/>
          </svg>
          GitHub
        </a>
      </nav>
    </nav>

    <!-- Hero -->
    <header class="status-hero">
      <div class="status-hero-inner">
        <span class="status-hero-badge" :class="overallStatusClass">
          <span class="status-dot" :class="overallStatusClass"></span>
          {{ overallStatusText }}
        </span>
        <h1 class="status-hero-title">
          Server <span class="gradient-text">Status</span>
        </h1>
        <p class="status-hero-subtitle">
          Real-time health monitoring of all WatchParty services.
          Last checked: {{ lastChecked }}
        </p>
        <p class="status-disclaimer">
          Note: This page currently displays simulated data for demonstration purposes.
        </p>
      </div>
    </header>

    <!-- Status Content -->
    <div class="status-content">
      <!-- Overall Summary Cards -->
      <div class="summary-grid">
        <div class="summary-card glass" v-for="s in summaryCards" :key="s.label">
          <div class="summary-icon" :class="s.status">
            <component :is="s.iconComponent" />
          </div>
          <div class="summary-info">
            <span class="summary-value">{{ s.value }}</span>
            <span class="summary-label">{{ s.label }}</span>
          </div>
        </div>
      </div>

      <!-- Services -->
      <div class="services-section">
        <h2 class="section-title">Service Health</h2>
        <div class="services-list">
          <div
            v-for="svc in services"
            :key="svc.name"
            class="service-card glass"
          >
            <div class="service-header">
              <div class="service-icon-box" :class="svc.status">
                <span v-html="svc.icon"></span>
              </div>
              <div class="service-info">
                <h3>{{ svc.name }}</h3>
                <p>{{ svc.description }}</p>
              </div>
              <div class="service-status-badge" :class="svc.status">
                <span class="status-indicator"></span>
                {{ statusText(svc.status) }}
              </div>
            </div>
            <div class="service-metrics">
              <div class="metric">
                <span class="metric-label">Uptime</span>
                <span class="metric-value">{{ svc.uptime }}</span>
              </div>
              <div class="metric">
                <span class="metric-label">Response</span>
                <span class="metric-value">{{ svc.responseTime }}</span>
              </div>
              <div class="metric">
                <span class="metric-label">Last Incident</span>
                <span class="metric-value">{{ svc.lastIncident }}</span>
              </div>
            </div>
            <!-- Mini uptime bar -->
            <div class="uptime-bar-wrapper">
              <div class="uptime-bar-label">90-day uptime</div>
              <div class="uptime-bar">
                <div
                  v-for="(day, i) in svc.uptimeHistory"
                  :key="i"
                  class="uptime-segment"
                  :class="day"
                  :title="'Day ' + (i + 1) + ': ' + day"
                ></div>
              </div>
              <div class="uptime-bar-legend">
                <span>90 days ago</span>
                <span>Today</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Incident History -->
      <div class="incidents-section">
        <h2 class="section-title">Recent Incidents</h2>
        <div v-if="incidents.length === 0" class="no-incidents glass">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
          <p>No incidents reported in the last 30 days.</p>
        </div>
        <div v-else class="incidents-list">
          <div v-for="inc in incidents" :key="inc.id" class="incident-card glass">
            <div class="incident-header">
              <span class="incident-severity" :class="inc.severity">{{ inc.severity }}</span>
              <span class="incident-date">{{ inc.date }}</span>
            </div>
            <h3>{{ inc.title }}</h3>
            <p>{{ inc.description }}</p>
            <div class="incident-duration">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
              </svg>
              Duration: {{ inc.duration }}
            </div>
          </div>
        </div>
      </div>

      <!-- Response Time Chart (CSS only) -->
      <div class="chart-section">
        <h2 class="section-title">Response Time (Last 24h)</h2>
        <div class="chart-card glass">
          <div class="chart-bars">
            <div
              v-for="(point, i) in responseTimeData"
              :key="i"
              class="chart-bar-wrapper"
            >
              <div class="chart-bar" :style="{ height: point.height + '%' }" :class="point.status">
                <span class="chart-tooltip">{{ point.value }}ms</span>
              </div>
              <span class="chart-label">{{ point.hour }}</span>
            </div>
          </div>
          <div class="chart-y-axis">
            <span>200ms</span>
            <span>100ms</span>
            <span>0ms</span>
          </div>
        </div>
      </div>

      <!-- Subscribe -->
      <div class="subscribe-section glass">
        <div class="subscribe-content">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/>
            <path d="M13.73 21a2 2 0 0 1-3.46 0"/>
          </svg>
          <div>
            <h3>Subscribe to Updates</h3>
            <p>Get notified when services experience downtime or degraded performance.</p>
          </div>
        </div>
        <button class="btn btn-primary" @click="subscribed = !subscribed">
          {{ subscribed ? '✓ Subscribed' : 'Subscribe' }}
        </button>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref, computed, h } from 'vue'

const subscribed = ref(false)
const mobileMenuOpen = ref(false)
const lastChecked = ref(new Date().toLocaleTimeString())

// SVG icon components
const ServerIcon = {
  render: () => h('svg', { width: 20, height: 20, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }, [
    h('rect', { x: 2, y: 2, width: 20, height: 8, rx: 2, ry: 2 }),
    h('rect', { x: 2, y: 14, width: 20, height: 8, rx: 2, ry: 2 }),
    h('line', { x1: 6, y1: 6, x2: '6.01', y2: 6 }),
    h('line', { x1: 6, y1: 18, x2: '6.01', y2: 18 }),
  ])
}
const UsersIcon = {
  render: () => h('svg', { width: 20, height: 20, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2 }, [
    h('path', { d: 'M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2' }),
    h('circle', { cx: 9, cy: 7, r: 4 }),
    h('path', { d: 'M23 21v-2a4 4 0 0 0-3-3.87' }),
    h('path', { d: 'M16 3.13a4 4 0 0 1 0 7.75' }),
  ])
}
const WifiIcon = {
  render: () => h('svg', { width: 20, height: 20, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2 }, [
    h('path', { d: 'M5 12.55a11 11 0 0 1 14.08 0' }),
    h('path', { d: 'M1.42 9a16 16 0 0 1 21.16 0' }),
    h('path', { d: 'M8.53 16.11a6 6 0 0 1 6.95 0' }),
    h('line', { x1: 12, y1: 20, x2: '12.01', y2: 20 }),
  ])
}
const ClockIcon = {
  render: () => h('svg', { width: 20, height: 20, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2 }, [
    h('circle', { cx: 12, cy: 12, r: 10 }),
    h('polyline', { points: '12 6 12 12 16 14' }),
  ])
}

const services = ref([
  {
    name: 'Web Application',
    description: 'Frontend UI and static assets',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>',
    status: 'operational',
    uptime: '99.98%',
    responseTime: '45ms',
    lastIncident: 'None',
    uptimeHistory: generateUptimeHistory(0.98),
  },
  {
    name: 'WebSocket Server',
    description: 'Real-time sync and chat',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>',
    status: 'operational',
    uptime: '99.95%',
    responseTime: '12ms',
    lastIncident: '3 days ago',
    uptimeHistory: generateUptimeHistory(0.95),
  },
  {
    name: 'API Server',
    description: 'REST API for room management',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>',
    status: 'operational',
    uptime: '99.99%',
    responseTime: '38ms',
    lastIncident: 'None',
    uptimeHistory: generateUptimeHistory(0.99),
  },
  {
    name: 'Video Proxy',
    description: 'HLS stream proxying service',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>',
    status: 'degraded',
    uptime: '98.72%',
    responseTime: '120ms',
    lastIncident: '2 hours ago',
    uptimeHistory: generateUptimeHistory(0.88),
  },
  {
    name: 'CDN / Static Assets',
    description: 'Global content delivery',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',
    status: 'operational',
    uptime: '99.99%',
    responseTime: '8ms',
    lastIncident: 'None',
    uptimeHistory: generateUptimeHistory(0.99),
  },
])

const incidents = ref([
  {
    id: 1,
    severity: 'minor',
    date: 'Jul 15, 2026',
    title: 'Elevated latency on Video Proxy',
    description: 'The HLS proxy service experienced increased response times due to upstream provider issues. Streams continued to work but with higher initial buffering times.',
    duration: '45 minutes',
  },
])

const summaryCards = computed(() => [
  {
    label: 'Services Online',
    value: services.value.filter(s => s.status === 'operational').length + '/' + services.value.length,
    status: 'operational',
    iconComponent: ServerIcon,
  },
  {
    label: 'Active Rooms',
    value: '24',
    status: 'operational',
    iconComponent: UsersIcon,
  },
  {
    label: 'Avg Response',
    value: '44ms',
    status: 'operational',
    iconComponent: WifiIcon,
  },
  {
    label: 'Overall Uptime',
    value: '99.93%',
    status: 'operational',
    iconComponent: ClockIcon,
  },
])

const responseTimeData = ref(
  Array.from({ length: 24 }, (_, i) => {
    const base = 35 + Math.random() * 30
    const spike = i === 14 ? 80 : 0
    const value = Math.round(base + spike)
    return {
      hour: String(i).padStart(2, '0'),
      value,
      height: Math.min((value / 200) * 100, 100),
      status: value > 100 ? 'warning' : value > 60 ? 'elevated' : 'good',
    }
  })
)

const overallStatusClass = computed(() => {
  const hasDown = services.value.some(s => s.status === 'down')
  const hasDegraded = services.value.some(s => s.status === 'degraded')
  if (hasDown) return 'down'
  if (hasDegraded) return 'degraded'
  return 'operational'
})

const overallStatusText = computed(() => {
  const cls = overallStatusClass.value
  if (cls === 'down') return 'Major Outage'
  if (cls === 'degraded') return 'Partial System Degradation'
  return 'All Systems Operational'
})

function statusText(status) {
  if (status === 'operational') return 'Operational'
  if (status === 'degraded') return 'Degraded'
  return 'Down'
}

function generateUptimeHistory(ratio) {
  const statuses = []
  for (let i = 0; i < 90; i++) {
    const rand = Math.random()
    if (rand < ratio) statuses.push('up')
    else if (rand < ratio + 0.02) statuses.push('degraded')
    else statuses.push('down')
  }
  return statuses
}
</script>

<style scoped>
.status-page {
  min-height: 100vh;
  position: relative;
  z-index: 1;
}

/* Nav */
.home-nav {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 var(--space-8);
  height: 56px;
  position: sticky; top: 0; z-index: var(--z-overlay);
  background: rgba(10, 11, 16, 0.8);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-bottom: 1px solid var(--color-glass-border);
}
.nav-logo {
  display: flex; align-items: center; gap: var(--space-2);
  text-decoration: none;
}
.nav-brand {
  font-size: 1.05rem; font-weight: 700; letter-spacing: -0.02em;
  color: var(--text-primary);
}
.nav-links { display: flex; align-items: center; gap: var(--space-1); }
.hamburger-btn {
  display: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: 1px solid var(--color-glass-border);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.hamburger-btn:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.05);
}
.nav-link {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 6px 12px; border-radius: var(--radius-sm);
  font-size: 0.875rem; font-weight: 500;
  color: var(--text-secondary); text-decoration: none;
  transition: all var(--transition-fast);
}
.nav-link:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.05);
}
.nav-link.active {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.08);
}
.nav-link svg { opacity: 0.7; }
.nav-link:hover svg, .nav-link.active svg { opacity: 1; }
.nav-right { display: flex; align-items: center; gap: var(--space-6); }

/* Hero */
.status-hero {
  padding: var(--space-12) var(--space-6) var(--space-8);
  text-align: center;
  max-width: 700px;
  margin: 0 auto;
}
.status-hero-badge {
  display: inline-flex; align-items: center; gap: 8px;
  padding: 6px 16px; border-radius: var(--radius-full);
  font-size: 0.85rem; font-weight: 600;
  margin-bottom: var(--space-4);
}
.status-hero-badge.operational {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: var(--color-accent-green);
}
.status-hero-badge.degraded {
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.3);
  color: var(--color-accent-amber);
}
.status-hero-badge.down {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: var(--color-accent-red);
}
.status-dot {
  width: 8px; height: 8px; border-radius: 50%;
}
.status-dot.operational {
  background: var(--color-accent-green);
  box-shadow: 0 0 8px var(--color-accent-green);
  animation: pulse-green 2s infinite;
}
.status-dot.degraded {
  background: var(--color-accent-amber);
  box-shadow: 0 0 8px var(--color-accent-amber);
  animation: pulse-amber 2s infinite;
}
.status-dot.down {
  background: var(--color-accent-red);
  box-shadow: 0 0 8px var(--color-accent-red);
  animation: pulse-red 2s infinite;
}

@keyframes pulse-green { 0%, 100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4); } 50% { box-shadow: 0 0 0 6px rgba(16, 185, 129, 0); } }
@keyframes pulse-amber { 0%, 100% { box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.4); } 50% { box-shadow: 0 0 0 6px rgba(245, 158, 11, 0); } }
@keyframes pulse-red { 0%, 100% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4); } 50% { box-shadow: 0 0 0 6px rgba(239, 68, 68, 0); } }

.status-hero-title {
  font-size: clamp(2rem, 4vw, 3rem);
  font-weight: 800; letter-spacing: -0.04em;
  line-height: 1.15; margin-bottom: var(--space-4);
}
.status-hero-subtitle {
  font-size: 1.05rem; color: var(--text-secondary);
  line-height: 1.7; max-width: 500px; margin: 0 auto;
}
.status-disclaimer {
  font-size: 0.8rem; color: var(--text-muted);
  font-style: italic; margin-top: var(--space-2);
}

/* Content */
.status-content {
  max-width: 900px;
  margin: 0 auto;
  padding: 0 var(--space-6) var(--space-16);
}

/* Summary */
.summary-grid {
  display: grid; grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
  margin-bottom: var(--space-10);
}
.summary-card {
  padding: var(--space-5);
  border-radius: var(--radius-lg);
  display: flex; align-items: center; gap: var(--space-4);
  transition: all var(--transition-base);
}
.summary-card:hover {
  background: var(--color-glass-hover);
  border-color: rgba(255, 255, 255, 0.12);
  transform: translateY(-2px);
}
.summary-icon {
  width: 44px; height: 44px; border-radius: var(--radius-md);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.summary-icon.operational {
  background: rgba(16, 185, 129, 0.12);
  color: var(--color-accent-green);
}
.summary-icon.degraded {
  background: rgba(245, 158, 11, 0.12);
  color: var(--color-accent-amber);
}
.summary-info { display: flex; flex-direction: column; }
.summary-value {
  font-size: 1.4rem; font-weight: 800;
  color: var(--text-primary); letter-spacing: -0.02em;
}
.summary-label {
  font-size: 0.8rem; color: var(--text-muted);
  font-weight: 500;
}

/* Section */
.section-title {
  font-size: 1.3rem; font-weight: 800;
  letter-spacing: -0.02em;
  margin-bottom: var(--space-5);
}

/* Services */
.services-section { margin-bottom: var(--space-10); }
.services-list { display: flex; flex-direction: column; gap: var(--space-4); }

.service-card {
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  transition: all var(--transition-base);
}
.service-card:hover {
  background: var(--color-glass-hover);
  border-color: rgba(255, 255, 255, 0.12);
}
.service-header {
  display: flex; align-items: center; gap: var(--space-4);
  margin-bottom: var(--space-4);
}
.service-icon-box {
  width: 44px; height: 44px; border-radius: var(--radius-md);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.service-icon-box.operational {
  background: rgba(16, 185, 129, 0.12);
  color: var(--color-accent-green);
}
.service-icon-box.degraded {
  background: rgba(245, 158, 11, 0.12);
  color: var(--color-accent-amber);
}
.service-icon-box.down {
  background: rgba(239, 68, 68, 0.12);
  color: var(--color-accent-red);
}
.service-info { flex: 1; }
.service-info h3 {
  font-size: 1rem; font-weight: 700; margin: 0 0 2px;
}
.service-info p {
  font-size: 0.85rem; color: var(--text-muted); margin: 0;
}
.service-status-badge {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 12px; border-radius: var(--radius-full);
  font-size: 0.8rem; font-weight: 600;
  flex-shrink: 0;
}
.service-status-badge.operational {
  background: rgba(16, 185, 129, 0.1);
  color: var(--color-accent-green);
}
.service-status-badge.degraded {
  background: rgba(245, 158, 11, 0.1);
  color: var(--color-accent-amber);
}
.service-status-badge.down {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-accent-red);
}
.status-indicator {
  width: 6px; height: 6px; border-radius: 50%;
}
.service-status-badge.operational .status-indicator { background: var(--color-accent-green); }
.service-status-badge.degraded .status-indicator { background: var(--color-accent-amber); }
.service-status-badge.down .status-indicator { background: var(--color-accent-red); }

.service-metrics {
  display: flex; gap: var(--space-6);
  padding: var(--space-3) 0;
  border-top: 1px solid var(--color-glass-border);
  border-bottom: 1px solid var(--color-glass-border);
  margin-bottom: var(--space-4);
}
.metric { display: flex; flex-direction: column; gap: 2px; }
.metric-label {
  font-size: 0.75rem; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.05em;
  color: var(--text-muted);
}
.metric-value {
  font-size: 0.9rem; font-weight: 700;
  color: var(--text-primary);
}

/* Uptime Bar */
.uptime-bar-wrapper { margin-top: var(--space-2); }
.uptime-bar-label {
  font-size: 0.75rem; color: var(--text-muted);
  margin-bottom: var(--space-2);
}
.uptime-bar {
  display: flex; gap: 2px;
  height: 20px;
}
.uptime-segment {
  flex: 1; border-radius: 2px;
  transition: opacity var(--transition-fast);
}
.uptime-segment:hover { opacity: 0.8; }
.uptime-segment.up { background: var(--color-accent-green); opacity: 0.7; }
.uptime-segment.degraded { background: var(--color-accent-amber); opacity: 0.8; }
.uptime-segment.down { background: var(--color-accent-red); opacity: 0.8; }
.uptime-bar-legend {
  display: flex; justify-content: space-between;
  font-size: 0.7rem; color: var(--text-muted);
  margin-top: var(--space-1);
}

/* Incidents */
.incidents-section { margin-bottom: var(--space-10); }
.no-incidents {
  display: flex; flex-direction: column;
  align-items: center; gap: var(--space-3);
  padding: var(--space-8); text-align: center;
  border-radius: var(--radius-lg);
  color: var(--color-accent-green);
}
.no-incidents p { color: var(--text-secondary); margin: 0; }
.incidents-list { display: flex; flex-direction: column; gap: var(--space-3); }
.incident-card {
  padding: var(--space-5);
  border-radius: var(--radius-lg);
}
.incident-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: var(--space-2);
}
.incident-severity {
  padding: 2px 10px; border-radius: var(--radius-full);
  font-size: 0.75rem; font-weight: 700;
  text-transform: uppercase; letter-spacing: 0.05em;
}
.incident-severity.minor {
  background: rgba(245, 158, 11, 0.1);
  color: var(--color-accent-amber);
}
.incident-severity.major {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-accent-red);
}
.incident-date {
  font-size: 0.8rem; color: var(--text-muted);
}
.incident-card h3 {
  font-size: 1rem; font-weight: 700; margin: 0 0 var(--space-2);
}
.incident-card p {
  font-size: 0.9rem; color: var(--text-secondary);
  line-height: 1.6; margin: 0 0 var(--space-3);
}
.incident-duration {
  display: flex; align-items: center; gap: 6px;
  font-size: 0.8rem; color: var(--text-muted);
}
.incident-duration svg { color: var(--color-primary); }

/* Chart */
.chart-section { margin-bottom: var(--space-10); }
.chart-card {
  padding: var(--space-5);
  border-radius: var(--radius-lg);
  position: relative;
}
.chart-bars {
  display: flex; align-items: flex-end; gap: 4px;
  height: 160px;
  padding-bottom: var(--space-5);
}
.chart-bar-wrapper {
  flex: 1; display: flex; flex-direction: column;
  align-items: center; height: 100%;
  justify-content: flex-end;
  position: relative;
}
.chart-bar {
  width: 100%; border-radius: 3px 3px 0 0;
  min-height: 4px;
  transition: all var(--transition-base);
  position: relative;
  cursor: pointer;
}
.chart-bar.good { background: var(--color-accent-green); opacity: 0.7; }
.chart-bar.elevated { background: var(--color-accent-amber); opacity: 0.8; }
.chart-bar.warning { background: var(--color-accent-red); opacity: 0.8; }
.chart-bar:hover { opacity: 1; }
.chart-tooltip {
  position: absolute; bottom: calc(100% + 6px);
  left: 50%; transform: translateX(-50%);
  background: rgba(14, 14, 28, 0.95);
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-sm);
  padding: 2px 8px;
  font-size: 0.7rem; font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  opacity: 0; pointer-events: none;
  transition: opacity var(--transition-fast);
}
.chart-bar:hover .chart-tooltip { opacity: 1; }
.chart-label {
  font-size: 0.65rem; color: var(--text-muted);
  margin-top: 4px;
}
.chart-y-axis {
  position: absolute; top: var(--space-5);
  right: var(--space-4);
  display: flex; flex-direction: column;
  justify-content: space-between;
  height: 160px;
  font-size: 0.7rem; color: var(--text-muted);
}

/* Subscribe */
.subscribe-section {
  display: flex; align-items: center; justify-content: space-between;
  padding: var(--space-6) var(--space-8);
  border-radius: var(--radius-lg);
  gap: var(--space-6);
}
.subscribe-content {
  display: flex; align-items: center; gap: var(--space-4);
}
.subscribe-content svg { color: var(--color-primary); flex-shrink: 0; }
.subscribe-content h3 {
  font-size: 1rem; font-weight: 700; margin: 0 0 2px;
}
.subscribe-content p {
  font-size: 0.85rem; color: var(--text-secondary); margin: 0;
}

@media (max-width: 768px) {
  .summary-grid { grid-template-columns: repeat(2, 1fr); }
  .service-metrics { flex-wrap: wrap; gap: var(--space-3); }
  .chart-bars { height: 120px; }
}

@media (max-width: 640px) {
  .hamburger-btn { display: flex; }
  .nav-links {
    display: none;
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    flex-direction: column;
    align-items: stretch;
    gap: 0;
    background: rgba(10, 11, 16, 0.95);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border-bottom: 1px solid var(--color-glass-border);
    padding: var(--space-2) var(--space-4);
    z-index: var(--z-overlay);
  }
  .nav-links--open { display: flex; }
  .nav-link { padding: var(--space-3) var(--space-2); }
  .home-nav { padding: var(--space-3) var(--space-4); position: relative; }
  .status-hero { padding: var(--space-8) var(--space-4) var(--space-6); }
  .summary-grid { grid-template-columns: 1fr 1fr; }
  .subscribe-section {
    flex-direction: column; text-align: center;
    padding: var(--space-5);
  }
  .subscribe-content { flex-direction: column; }
  .service-header { flex-wrap: wrap; }
}
</style>
