<template>
  <main class="docs-page">
    <!-- Nav -->
    <nav class="home-nav">
      <router-link to="/" class="nav-logo">
        <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
          <circle cx="14" cy="14" r="14" fill="url(#docs-logo-grad)"/>
          <polygon points="11,9 21,14 11,19" fill="white"/>
          <defs>
            <linearGradient id="docs-logo-grad" x1="0" y1="0" x2="28" y2="28">
              <stop offset="0%" stop-color="hsl(195,100%,45%)"/>
              <stop offset="100%" stop-color="hsl(215,90%,50%)"/>
            </linearGradient>
          </defs>
        </svg>
        <span class="nav-brand">WatchParty</span>
      </router-link>
      <nav class="nav-links" :class="{ 'nav-links--open': mobileMenuOpen }" aria-label="Main">
        <router-link to="/docs" class="nav-link active" @click="mobileMenuOpen = false">
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
        <router-link to="/status" class="nav-link" @click="mobileMenuOpen = false">
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
      <div class="nav-right-controls">
        <ThemeToggle />
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
      </div>
    </nav>

    <!-- Content Layout -->
    <div class="docs-layout">
      <!-- Sidebar / Table of Contents -->
      <aside class="docs-sidebar">
        <div class="sidebar-sticky">
          <div class="sidebar-brand">
            <svg width="20" height="20" viewBox="0 0 28 28" fill="none">
              <circle cx="14" cy="14" r="14" fill="url(#sidebar-logo-grad)"/>
              <polygon points="11,9 21,14 11,19" fill="white"/>
              <defs>
                <linearGradient id="sidebar-logo-grad" x1="0" y1="0" x2="28" y2="28">
                  <stop offset="0%" stop-color="hsl(195,100%,45%)"/>
                  <stop offset="100%" stop-color="hsl(215,90%,50%)"/>
                </linearGradient>
              </defs>
            </svg>
            <span class="gradient-text sidebar-brand-text">Docs</span>
          </div>
          <h4 class="sidebar-title">Getting Started</h4>
          <nav class="sidebar-nav">
            <a
              v-for="section in sections"
              :key="section.id"
              :href="'#' + section.id"
              class="sidebar-link"
              :class="{ active: activeSection === section.id }"
              @click.prevent="scrollToSection(section.id)"
            >
              <span class="sidebar-link-icon">{{ section.icon }}</span>
              {{ section.title }}
            </a>
          </nav>
          <div class="sidebar-divider"></div>
          <h4 class="sidebar-title">Resources</h4>
          <nav class="sidebar-nav">
            <router-link to="/faq" class="sidebar-link">
              <span class="sidebar-link-icon">❓</span>
              FAQ
            </router-link>
            <router-link to="/status" class="sidebar-link">
              <span class="sidebar-link-icon">📊</span>
              Server Status
            </router-link>
            <a href="https://github.com/litcq/streaming_platform" target="_blank" rel="noopener noreferrer" class="sidebar-link">
              <span class="sidebar-link-icon">🐙</span>
              GitHub
            </a>
          </nav>
        </div>
      </aside>

      <!-- Main Content -->
      <div class="docs-content">
        <!-- Hero -->
        <header class="docs-hero">
          <span class="docs-hero-badge">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/>
            </svg>
            Documentation
          </span>
          <h1 class="docs-hero-title">
            <span class="gradient-text">Getting Started</span> with WatchParty
          </h1>
          <p class="docs-hero-subtitle">
            Everything you need to know about creating and joining watch parties.
            Start watching videos together in perfect sync.
          </p>
        </header>

        <!-- Quick Start -->
        <section id="quick-start" class="docs-section">
          <div class="section-header">
            <span class="section-icon">🚀</span>
            <h2>Quick Start</h2>
          </div>
          <p>Get started with WatchParty in three simple steps:</p>

          <div class="steps-grid">
            <div class="step-card glass">
              <div class="step-number">1</div>
              <h3>Create a Room</h3>
              <p>Click the <strong>"Create Watch Room"</strong> button on the homepage. A unique room link will be generated for you instantly.</p>
              <div class="step-code">
                <code>https://watchparty.live/room/<span class="code-highlight">abc123</span></code>
              </div>
            </div>
            <div class="step-card glass">
              <div class="step-number">2</div>
              <h3>Share the Link</h3>
              <p>Copy the room URL and share it with friends. Anyone with the link can join your watch party — no account needed.</p>
              <div class="step-tip">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>
                </svg>
                <span>Tip: You can also share via the in-room share button</span>
              </div>
            </div>
            <div class="step-card glass">
              <div class="step-number">3</div>
              <h3>Watch Together</h3>
              <p>Paste a video URL (MP4 or HLS stream) and hit play. All participants will be perfectly synchronized in real-time.</p>
              <div class="step-formats">
                <span class="format-badge">MP4</span>
                <span class="format-badge">HLS (.m3u8)</span>
                <span class="format-badge">YouTube</span>
              </div>
            </div>
          </div>
        </section>

        <!-- Room Controls -->
        <section id="room-controls" class="docs-section">
          <div class="section-header">
            <span class="section-icon">🎮</span>
            <h2>Room Controls</h2>
          </div>
          <p>As a <strong>host</strong>, you have full control over the playback experience. Guests can watch and chat, but only the host controls the video.</p>

          <div class="controls-table glass">
            <div class="table-header">
              <span>Action</span>
              <span>Host</span>
              <span>Guest</span>
            </div>
            <div class="table-row" v-for="ctrl in controls" :key="ctrl.action">
              <span class="ctrl-action">
                <span class="ctrl-icon">{{ ctrl.icon }}</span>
                {{ ctrl.action }}
              </span>
              <span :class="ctrl.host ? 'perm-yes' : 'perm-no'">
                {{ ctrl.host ? '✓' : '✗' }}
              </span>
              <span :class="ctrl.guest ? 'perm-yes' : 'perm-no'">
                {{ ctrl.guest ? '✓' : '✗' }}
              </span>
            </div>
          </div>
        </section>

        <!-- Video Sources -->
        <section id="video-sources" class="docs-section">
          <div class="section-header">
            <span class="section-icon">🎬</span>
            <h2>Supported Video Sources</h2>
          </div>
          <p>WatchParty supports multiple video source formats for maximum flexibility:</p>

          <div class="sources-grid">
            <div class="source-card glass">
              <div class="source-header">
                <div class="source-icon-box mp4">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polygon points="5 3 19 12 5 21 5 3"/>
                  </svg>
                </div>
                <h3>Direct MP4 Links</h3>
              </div>
              <p>Any publicly accessible .mp4 URL. The video is played using the native HTML5 video element.</p>
              <div class="source-example">
                <code>https://example.com/video.mp4</code>
              </div>
            </div>

            <div class="source-card glass">
              <div class="source-header">
                <div class="source-icon-box hls">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
                  </svg>
                </div>
                <h3>HLS Streams</h3>
              </div>
              <p>HTTP Live Streaming (.m3u8) URLs are supported via hls.js. Great for adaptive bitrate streaming.</p>
              <div class="source-example">
                <code>https://example.com/stream/index.m3u8</code>
              </div>
            </div>

            <div class="source-card glass">
              <div class="source-header">
                <div class="source-icon-box youtube">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="2" y="2" width="20" height="20" rx="5" ry="5"/>
                    <polygon points="10 8 16 12 10 16 10 8" fill="currentColor"/>
                  </svg>
                </div>
                <h3>YouTube Videos</h3>
              </div>
              <p>Paste any YouTube video URL. The platform extracts the video ID and syncs playback across all participants.</p>
              <div class="source-example">
                <code>https://youtube.com/watch?v=...</code>
              </div>
            </div>
          </div>

          <!-- Anime/Movie Sites -->
          <h3 class="sub-section-title">Anime & Movie Page URLs</h3>
          <p>Paste an episode or movie page URL directly — the backend automatically resolves the best available stream. No need to find the raw video link yourself.</p>

          <div class="sites-table glass">
            <div class="sites-table-header">
              <span>Platform</span>
              <span>Method</span>
              <span>Speed</span>
            </div>
            <div class="sites-table-row" v-for="site in scrapableSites" :key="site.name">
              <span class="site-name">
                <span class="site-dot" :class="site.speed"></span>
                {{ site.name }}
              </span>
              <span class="site-method">{{ site.method }}</span>
              <span class="site-speed" :class="site.speed">{{ site.speedLabel }}</span>
            </div>
          </div>

          <div class="site-note glass">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>
            </svg>
            <span>Any anime/movie site not listed above also works via our generic multi-layer scraper (HTML → scripts → iframes → headless browser).</span>
          </div>
        </section>

        <!-- Chat & Interaction -->
        <section id="chat" class="docs-section">
          <div class="section-header">
            <span class="section-icon">💬</span>
            <h2>Chat & Interaction</h2>
          </div>
          <p>Every room comes with a built-in real-time chat panel powered by WebSockets.</p>

          <div class="feature-list">
            <div class="feature-item glass">
              <div class="feature-check">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
              </div>
              <div>
                <h4>Instant Messaging</h4>
                <p>Messages are delivered in real-time to all room participants with sub-second latency.</p>
              </div>
            </div>
            <div class="feature-item glass">
              <div class="feature-check">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
              </div>
              <div>
                <h4>Nickname System</h4>
                <p>Choose a display name when joining. No registration required — just pick a name and go.</p>
              </div>
            </div>
            <div class="feature-item glass">
              <div class="feature-check">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
              </div>
              <div>
                <h4>Participant Count</h4>
                <p>See how many people are watching in real-time. The count updates automatically as users join and leave.</p>
              </div>
            </div>
          </div>
        </section>

        <!-- Queue System -->
        <section id="queue" class="docs-section">
          <div class="section-header">
            <span class="section-icon">📋</span>
            <h2>Queue System</h2>
          </div>
          <p>Line up multiple videos for a seamless binge-watching experience.</p>

          <div class="queue-demo glass">
            <div class="queue-header">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/>
                <line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/>
              </svg>
              <span>Video Queue</span>
            </div>
            <div class="queue-list">
              <div class="queue-item now-playing">
                <span class="queue-status">▶ Now Playing</span>
                <span class="queue-title">Episode 1 - Pilot</span>
                <span class="queue-duration">22:14</span>
              </div>
              <div class="queue-item">
                <span class="queue-pos">2</span>
                <span class="queue-title">Episode 2 - The Storm</span>
                <span class="queue-duration">21:48</span>
              </div>
              <div class="queue-item">
                <span class="queue-pos">3</span>
                <span class="queue-title">Episode 3 - Revelations</span>
                <span class="queue-duration">23:01</span>
              </div>
            </div>
            <p class="queue-note">Videos automatically play in order. The host can reorder or remove items from the queue at any time.</p>
          </div>
        </section>

        <!-- Keyboard Shortcuts -->
        <section id="shortcuts" class="docs-section">
          <div class="section-header">
            <span class="section-icon">⌨️</span>
            <h2>Keyboard Shortcuts</h2>
          </div>
          <p>Control playback faster with these keyboard shortcuts:</p>

          <div class="shortcuts-grid">
            <div class="shortcut-item" v-for="sc in shortcuts" :key="sc.key">
              <kbd class="shortcut-key">{{ sc.key }}</kbd>
              <span class="shortcut-desc">{{ sc.desc }}</span>
            </div>
          </div>
        </section>

        <!-- API Reference -->
        <section id="api" class="docs-section">
          <div class="section-header">
            <span class="section-icon">🔌</span>
            <h2>WebSocket API</h2>
          </div>
          <p>For developers who want to build custom integrations, WatchParty exposes a WebSocket API:</p>

          <div class="api-block glass">
            <div class="api-header">
              <span class="api-method ws">WS</span>
              <code>/ws/room/{roomId}</code>
            </div>
            <div class="api-body">
              <p class="api-desc">Connect to a room's WebSocket endpoint for real-time events.</p>
              <div class="api-events">
                <h5>Events:</h5>
                <div class="api-event">
                  <code class="event-name">JOIN_EVENT</code>
                  <span class="event-desc">Join a room with username</span>
                </div>
                <div class="api-event">
                  <code class="event-name">SYNC_EVENT</code>
                  <span class="event-desc">Sync video playback (play/pause/seek) across all clients</span>
                </div>
                <div class="api-event">
                  <code class="event-name">CHAT_EVENT</code>
                  <span class="event-desc">Send and receive chat messages</span>
                </div>
                <div class="api-event">
                  <code class="event-name">SET_VIDEO</code>
                  <span class="event-desc">Change the current video URL (host only)</span>
                </div>
                <div class="api-event">
                  <code class="event-name">ROOM_INIT</code>
                  <span class="event-desc">Initial room state sent to new joiners</span>
                </div>
                <div class="api-event">
                  <code class="event-name">USER_LEFT</code>
                  <span class="event-desc">Notify when a participant leaves the room</span>
                </div>
                <div class="api-event">
                  <code class="event-name">SCRAPE_ERROR</code>
                  <span class="event-desc">Notify stream extraction failure for anime/movie URLs</span>
                </div>
                <div class="api-event">
                  <code class="event-name">ADD_TO_QUEUE</code>
                  <span class="event-desc">Add a video URL to the play queue</span>
                </div>
                <div class="api-event">
                  <code class="event-name">QUEUE_UPDATE</code>
                  <span class="event-desc">Notify queue changes to all clients</span>
                </div>
                <div class="api-event">
                  <code class="event-name">SKIP_TO_NEXT</code>
                  <span class="event-desc">Skip to the next video in queue (host only)</span>
                </div>
                <div class="api-event">
                  <code class="event-name">TOGGLE_PUBLIC</code>
                  <span class="event-desc">Toggle room public/private visibility (host only)</span>
                </div>
                <div class="api-event">
                  <code class="event-name">HOST_CHANGED</code>
                  <span class="event-desc">Notify when the host role is reassigned</span>
                </div>
                <div class="api-event">
                  <code class="event-name">TRANSFER_HOST</code>
                  <span class="event-desc">Host promotes another participant (host only)</span>
                </div>
                <div class="api-event">
                  <code class="event-name">KICK_USER</code>
                  <span class="event-desc">Host removes a participant (host only)</span>
                </div>
                <div class="api-event">
                  <code class="event-name">KICKED</code>
                  <span class="event-desc">Sent to the kicked client before disconnect</span>
                </div>
                <div class="api-event">
                  <code class="event-name">SET_ROOM_NAME</code>
                  <span class="event-desc">Host renames the room (host only)</span>
                </div>
                <div class="api-event">
                  <code class="event-name">ROOM_NAME_CHANGED</code>
                  <span class="event-desc">Broadcast when room name is updated</span>
                </div>
                <div class="api-event">
                  <code class="event-name">SCRAPE_STARTED</code>
                  <span class="event-desc">Stream extraction began (e.g. IDLIX may take 30–90s)</span>
                </div>
                <div class="api-event">
                  <code class="event-name">SCRAPE_FINISHED</code>
                  <span class="event-desc">Stream extraction finished (success path)</span>
                </div>
                <div class="api-event">
                  <code class="event-name">REACTION</code>
                  <span class="event-desc">Floating emoji reaction in the room</span>
                </div>
                <div class="api-event">
                  <code class="event-name">TYPING</code>
                  <span class="event-desc">Chat typing indicator (not echoed to sender)</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- Back to top -->
        <div class="docs-footer-nav">
          <router-link to="/" class="btn btn-ghost">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 12H5"/><polyline points="12 19 5 12 12 5"/>
            </svg>
            Back to Home
          </router-link>
          <router-link to="/faq" class="btn btn-primary">
            Read FAQ
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M5 12h14"/><polyline points="12 5 19 12 12 19"/>
            </svg>
          </router-link>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import ThemeToggle from '../components/ThemeToggle.vue'

const activeSection = ref('quick-start')
const mobileMenuOpen = ref(false)

const sections = [
  { id: 'quick-start', title: 'Quick Start', icon: '🚀' },
  { id: 'room-controls', title: 'Room Controls', icon: '🎮' },
  { id: 'video-sources', title: 'Video Sources', icon: '🎬' },
  { id: 'chat', title: 'Chat & Interaction', icon: '💬' },
  { id: 'queue', title: 'Queue System', icon: '📋' },
  { id: 'shortcuts', title: 'Keyboard Shortcuts', icon: '⌨️' },
  { id: 'api', title: 'WebSocket API', icon: '🔌' },
]

const controls = [
  { icon: '▶️', action: 'Play / Pause', host: true, guest: false },
  { icon: '⏩', action: 'Seek', host: true, guest: false },
  { icon: '🔊', action: 'Volume Control', host: true, guest: true },
  { icon: '📺', action: 'Load Video URL', host: true, guest: false },
  { icon: '📋', action: 'Manage Queue', host: true, guest: false },
  { icon: '💬', action: 'Send Chat', host: true, guest: true },
  { icon: '👥', action: 'Kick User', host: true, guest: false },
]

const shortcuts = [
  { key: 'Space', desc: 'Play / Pause video' },
  { key: 'F', desc: 'Toggle fullscreen' },
  { key: 'M', desc: 'Mute / Unmute' },
  { key: '→', desc: 'Seek forward 10s' },
  { key: '←', desc: 'Seek backward 10s' },
  { key: 'Enter', desc: 'Send chat message' },
]

const scrapableSites = [
  { name: 'Samehadaku', method: 'WordPress AJAX player API', speed: 'fast', speedLabel: '~1–3s' },
  { name: 'Otakudesu', method: 'WordPress AJAX + Packer unpack', speed: 'fast', speedLabel: '~2–4s' },
  { name: 'Anoboy', method: 'Iframe + base64 + Blogger', speed: 'fast', speedLabel: '~2–5s' },
  { name: 'Animasu', method: 'Generic multi-layer scraper', speed: 'medium', speedLabel: '~3–10s' },
  { name: 'Kuronime', method: 'Generic multi-layer scraper', speed: 'medium', speedLabel: '~3–10s' },
  { name: 'Nanime', method: 'Generic multi-layer scraper', speed: 'medium', speedLabel: '~3–10s' },
  { name: 'IDLIX', method: 'API gate/claim chain', speed: 'slow', speedLabel: '~15–25s' },
]

function scrollToSection(id) {
  const el = document.getElementById(id)
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    activeSection.value = id
  }
}

let scrollTicking = false
function handleScroll() {
  if (scrollTicking) return
  scrollTicking = true
  requestAnimationFrame(() => {
    const scrollPos = window.scrollY + 120
    for (let i = sections.length - 1; i >= 0; i--) {
      const el = document.getElementById(sections[i].id)
      if (el && el.offsetTop <= scrollPos) {
        activeSection.value = sections[i].id
        break
      }
    }
    scrollTicking = false
  })
}

onMounted(() => window.addEventListener('scroll', handleScroll))
onUnmounted(() => window.removeEventListener('scroll', handleScroll))
</script>

<style scoped>
.docs-page {
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
  background: color-mix(in srgb, var(--color-bg-base) 82%, transparent);
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
.nav-links { display: flex; align-items: center; gap: var(--space-1); margin-left: auto; }
.nav-right-controls {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
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
  background: var(--color-glass-hover);
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
  background: var(--color-glass-hover);
}
.nav-link.active {
  color: var(--text-primary);
  background: var(--color-glass-hover);
}
.nav-link svg { opacity: 0.7; }
.nav-link:hover svg, .nav-link.active svg { opacity: 1; }

/* Hero */
.docs-hero {
  padding: var(--space-10) 0 var(--space-8);
  max-width: 800px;
}
.docs-hero-badge {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 4px 14px; border-radius: var(--radius-full);
  font-size: 0.8rem; font-weight: 600;
  background: var(--grad-brand-subtle);
  border: 1px solid hsla(195, 100%, 45%, 0.2);
  color: var(--color-primary);
  margin-bottom: var(--space-4);
}
.docs-hero-title {
  font-size: clamp(2rem, 4vw, 3rem);
  font-weight: 800; letter-spacing: -0.04em;
  line-height: 1.15; margin-bottom: var(--space-4);
}
.docs-hero-subtitle {
  font-size: 1.1rem; color: var(--text-secondary);
  line-height: 1.7; max-width: 600px; margin: 0 auto;
}

/* Layout */
.docs-layout {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  min-height: calc(100vh - 56px);
}

/* Sidebar */
.docs-sidebar {
  position: relative;
  border-right: 1px solid var(--color-glass-border);
  background: color-mix(in srgb, var(--color-bg-surface) 55%, transparent);
}
.sidebar-sticky {
  position: sticky; top: 56px;
  padding: var(--space-6) var(--space-5);
  height: calc(100vh - 56px);
  overflow-y: auto;
}
.sidebar-brand {
  display: flex; align-items: center; gap: var(--space-2);
  margin-bottom: var(--space-6);
  padding: 0 var(--space-2);
}
.sidebar-brand-text {
  font-size: 0.85rem; font-weight: 700; letter-spacing: 0.05em;
  text-transform: uppercase;
}
.sidebar-title {
  font-size: 0.7rem; font-weight: 700;
  text-transform: uppercase; letter-spacing: 0.1em;
  color: var(--text-muted); margin-bottom: var(--space-2);
  padding: 0 var(--space-2);
}
.sidebar-nav { display: flex; flex-direction: column; gap: 1px; margin-bottom: var(--space-4); }
.sidebar-link {
  display: flex; align-items: center; gap: var(--space-2);
  padding: 7px 10px; border-radius: var(--radius-sm);
  font-size: 0.85rem; font-weight: 500;
  color: var(--text-secondary); text-decoration: none;
  transition: all var(--transition-fast);
}
.sidebar-link:hover { color: var(--text-primary); background: var(--color-glass-hover); }
.sidebar-link.active {
  color: var(--color-primary);
  background: var(--grad-brand-subtle);
  font-weight: 600;
}
.sidebar-link-icon { font-size: 0.85rem; width: 20px; text-align: center; flex-shrink: 0; }
.sidebar-divider {
  height: 1px;
  background: var(--color-glass-border);
  margin: var(--space-4) 0;
}

/* Content Area */
.docs-content {
  padding: 0 var(--space-8) var(--space-16);
  max-width: 900px;
  min-width: 0;
}

/* Sections */
.docs-section {
  margin-bottom: var(--space-12);
  scroll-margin-top: 80px;
}
.section-header {
  display: flex; align-items: center; gap: var(--space-3);
  margin-bottom: var(--space-3);
}
.section-icon { font-size: 1.5rem; }
.docs-section h2 {
  font-size: 1.5rem; font-weight: 800;
  letter-spacing: -0.02em; margin: 0;
}
.docs-section > p {
  font-size: 1rem; color: var(--text-secondary);
  line-height: 1.7; margin-bottom: var(--space-6);
}

/* Steps */
.steps-grid {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: var(--space-5);
}
.step-card {
  padding: var(--space-6); border-radius: var(--radius-lg);
  display: flex; flex-direction: column; gap: var(--space-3);
  transition: all var(--transition-base);
}
.step-card:hover {
  background: var(--color-glass-hover);
  border-color: rgba(255, 255, 255, 0.12);
  transform: translateY(-2px);
}
.step-number {
  width: 32px; height: 32px;
  border-radius: 50%;
  background: var(--grad-brand);
  color: white; font-weight: 800; font-size: 0.9rem;
  display: flex; align-items: center; justify-content: center;
  box-shadow: var(--shadow-glow-sm);
}
.step-card h3 { font-size: 1rem; font-weight: 700; margin: 0; }
.step-card p { font-size: 0.9rem; color: var(--text-secondary); line-height: 1.6; margin: 0; }
.step-code {
  background: color-mix(in srgb, var(--color-bg-elevated) 90%, transparent);
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-sm);
  padding: 6px 10px;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--text-secondary);
  overflow-x: auto;
}
.code-highlight { color: var(--color-primary); font-weight: 600; }
.step-tip {
  display: flex; align-items: flex-start; gap: 6px;
  font-size: 0.8rem; color: var(--text-muted);
  padding: 6px 8px; border-radius: var(--radius-sm);
  background: var(--color-glass);
  border: 1px solid var(--color-glass-border);
}
.step-tip svg { flex-shrink: 0; margin-top: 1px; color: var(--color-primary); }
.step-formats { display: flex; gap: var(--space-2); flex-wrap: wrap; }
.format-badge {
  padding: 3px 10px; border-radius: var(--radius-full);
  font-size: 0.75rem; font-weight: 600;
  background: var(--color-glass-hover);
  border: 1px solid var(--color-glass-border);
  color: var(--text-secondary);
}

/* Controls Table */
.controls-table {
  border-radius: var(--radius-lg);
  overflow: hidden;
}
.table-header, .table-row {
  display: grid;
  grid-template-columns: 1fr 80px 80px;
  padding: var(--space-3) var(--space-4);
  align-items: center;
}
.table-header {
  background: var(--color-glass);
  font-size: 0.75rem; font-weight: 700;
  text-transform: uppercase; letter-spacing: 0.06em;
  color: var(--text-muted);
  border-bottom: 1px solid var(--color-glass-border);
}
.table-row {
  border-bottom: 1px solid var(--color-glass-border);
  font-size: 0.9rem;
}
.table-row:last-child { border-bottom: none; }
.ctrl-action {
  display: flex; align-items: center; gap: var(--space-2);
  color: var(--text-primary); font-weight: 500;
}
.ctrl-icon { font-size: 1rem; width: 24px; text-align: center; }
.perm-yes { color: var(--color-accent-green); font-weight: 700; text-align: center; }
.perm-no { color: var(--color-accent-red); opacity: 0.5; font-weight: 700; text-align: center; }

/* Sources */
.sources-grid {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: var(--space-5);
}
.source-card {
  padding: var(--space-6); border-radius: var(--radius-lg);
  display: flex; flex-direction: column; gap: var(--space-3);
  transition: all var(--transition-base);
}
.source-card:hover {
  background: var(--color-glass-hover);
  border-color: rgba(255, 255, 255, 0.12);
  transform: translateY(-2px);
}
.source-header { display: flex; align-items: center; gap: var(--space-3); }
.source-icon-box {
  width: 40px; height: 40px; border-radius: var(--radius-md);
  display: flex; align-items: center; justify-content: center;
}
.source-icon-box.mp4 { background: rgba(16, 185, 129, 0.15); color: var(--color-accent-green); }
.source-icon-box.hls { background: hsla(195, 100%, 45%, 0.15); color: var(--color-primary); }
.source-icon-box.youtube { background: rgba(239, 68, 68, 0.15); color: var(--color-accent-red); }
.source-card h3 { font-size: 1rem; font-weight: 700; margin: 0; }
.source-card p { font-size: 0.9rem; color: var(--text-secondary); line-height: 1.6; margin: 0; }
.source-example {
  background: color-mix(in srgb, var(--color-bg-elevated) 90%, transparent);
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-sm);
  padding: 6px 10px;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--color-primary);
}

/* Sub Section Title */
.sub-section-title {
  font-size: 1.1rem; font-weight: 700;
  margin: var(--space-8) 0 var(--space-2);
  color: var(--text-primary);
}

/* Sites Table */
.sites-table {
  border-radius: var(--radius-lg);
  overflow: hidden;
  margin-top: var(--space-4);
}
.sites-table-header, .sites-table-row {
  display: grid;
  grid-template-columns: 1fr 1fr 100px;
  padding: var(--space-3) var(--space-4);
  align-items: center;
}
.sites-table-header {
  background: var(--color-glass);
  font-size: 0.75rem; font-weight: 700;
  text-transform: uppercase; letter-spacing: 0.06em;
  color: var(--text-muted);
  border-bottom: 1px solid var(--color-glass-border);
}
.sites-table-row {
  border-bottom: 1px solid var(--color-glass-border);
  font-size: 0.9rem;
}
.sites-table-row:last-child { border-bottom: none; }
.site-name {
  display: flex; align-items: center; gap: var(--space-2);
  color: var(--text-primary); font-weight: 600;
}
.site-dot {
  width: 8px; height: 8px; border-radius: 50%;
  flex-shrink: 0;
}
.site-dot.fast { background: var(--color-accent-green); box-shadow: 0 0 6px var(--color-accent-green); }
.site-dot.medium { background: var(--color-accent-amber); box-shadow: 0 0 6px var(--color-accent-amber); }
.site-dot.slow { background: var(--color-accent-red); box-shadow: 0 0 6px var(--color-accent-red); }
.site-method {
  font-size: 0.85rem; color: var(--text-secondary);
}
.site-speed {
  font-size: 0.85rem; font-weight: 600;
  text-align: right;
}
.site-speed.fast { color: var(--color-accent-green); }
.site-speed.medium { color: var(--color-accent-amber); }
.site-speed.slow { color: var(--color-accent-red); }

.site-note {
  display: flex; align-items: flex-start; gap: var(--space-3);
  padding: var(--space-4);
  border-radius: var(--radius-md);
  margin-top: var(--space-4);
  font-size: 0.85rem; color: var(--text-secondary);
  line-height: 1.5;
}
.site-note svg { flex-shrink: 0; margin-top: 2px; color: var(--color-primary); }

/* Feature List */
.feature-list { display: flex; flex-direction: column; gap: var(--space-3); }
.feature-item {
  padding: var(--space-4) var(--space-5);
  border-radius: var(--radius-md);
  display: flex; align-items: flex-start; gap: var(--space-4);
}
.feature-check {
  width: 28px; height: 28px; flex-shrink: 0;
  border-radius: 50%;
  background: rgba(16, 185, 129, 0.12);
  color: var(--color-accent-green);
  display: flex; align-items: center; justify-content: center;
  margin-top: 2px;
}
.feature-item h4 { font-size: 0.95rem; font-weight: 700; margin: 0 0 2px; }
.feature-item p { font-size: 0.875rem; color: var(--text-secondary); margin: 0; line-height: 1.5; }

/* Queue Demo */
.queue-demo { border-radius: var(--radius-lg); padding: var(--space-5); }
.queue-header {
  display: flex; align-items: center; gap: var(--space-2);
  font-size: 0.9rem; font-weight: 700; color: var(--text-primary);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--color-glass-border);
  margin-bottom: var(--space-3);
}
.queue-list { display: flex; flex-direction: column; gap: 2px; }
.queue-item {
  display: flex; align-items: center; gap: var(--space-3);
  padding: 8px 10px; border-radius: var(--radius-sm);
  font-size: 0.875rem;
  transition: background var(--transition-fast);
}
.queue-item:hover { background: var(--color-glass-hover); }
.queue-item.now-playing {
  background: var(--grad-brand-subtle);
  border-left: 2px solid var(--color-primary);
}
.queue-status {
  font-size: 0.75rem; font-weight: 700; color: var(--color-primary);
  min-width: 90px;
}
.queue-pos {
  width: 20px; height: 20px; border-radius: 50%;
  background: var(--color-glass-hover);
  font-size: 0.7rem; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  color: var(--text-muted);
}
.queue-title { flex: 1; color: var(--text-primary); font-weight: 500; }
.queue-duration {
  font-family: var(--font-mono);
  font-size: 0.8rem; color: var(--text-muted);
}
.queue-note {
  font-size: 0.85rem; color: var(--text-muted);
  margin-top: var(--space-3); padding-top: var(--space-3);
  border-top: 1px solid var(--color-glass-border);
}

/* Shortcuts */
.shortcuts-grid {
  display: grid; grid-template-columns: repeat(2, 1fr);
  gap: var(--space-3);
}
.shortcut-item {
  display: flex; align-items: center; gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: var(--color-glass);
  border: 1px solid var(--color-glass-border);
}
.shortcut-key {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 48px; height: 32px; padding: 0 10px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono); font-size: 0.8rem;
  font-weight: 600; color: var(--text-primary);
  box-shadow: 0 2px 0 color-mix(in srgb, var(--text-primary) 12%, transparent);
}
.shortcut-desc { font-size: 0.9rem; color: var(--text-secondary); }

/* API */
.api-block { border-radius: var(--radius-lg); overflow: hidden; }
.api-header {
  display: flex; align-items: center; gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  background: var(--color-glass);
  border-bottom: 1px solid var(--color-glass-border);
}
.api-method {
  padding: 3px 10px; border-radius: var(--radius-sm);
  font-size: 0.75rem; font-weight: 800;
  letter-spacing: 0.05em;
}
.api-method.ws {
  background: rgba(16, 185, 129, 0.15);
  color: var(--color-accent-green);
}
.api-header code {
  font-family: var(--font-mono); font-size: 0.9rem;
  color: var(--text-primary);
}
.api-body { padding: var(--space-5); }
.api-desc {
  font-size: 0.9rem; color: var(--text-secondary);
  margin: 0 0 var(--space-4);
}
.api-events h5 {
  font-size: 0.8rem; font-weight: 700;
  text-transform: uppercase; letter-spacing: 0.06em;
  color: var(--text-muted); margin: 0 0 var(--space-3);
}
.api-event {
  display: flex; align-items: center; gap: var(--space-3);
  padding: 6px 0;
  border-bottom: 1px solid var(--color-glass-border);
}
.api-event:last-child { border-bottom: none; }
.event-name {
  font-family: var(--font-mono); font-size: 0.85rem;
  color: var(--color-primary); font-weight: 600;
  min-width: 120px;
}
.event-desc { font-size: 0.85rem; color: var(--text-secondary); }

/* Footer Nav */
.docs-footer-nav {
  display: flex; justify-content: space-between;
  padding-top: var(--space-8);
  border-top: 1px solid var(--color-glass-border);
  margin-top: var(--space-8);
}

@media (max-width: 900px) {
  .docs-layout { grid-template-columns: 1fr; }
  .docs-sidebar {
    display: none;
  }
  .docs-content { padding: 0 var(--space-6) var(--space-16); }
  .steps-grid, .sources-grid { grid-template-columns: 1fr; }
  .shortcuts-grid { grid-template-columns: 1fr; }
  .table-header, .table-row { grid-template-columns: 1fr 60px 60px; }
  .sites-table-header, .sites-table-row { grid-template-columns: 1fr 1fr 80px; }
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
    background: color-mix(in srgb, var(--color-bg-base) 95%, transparent);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border-bottom: 1px solid var(--color-glass-border);
    padding: var(--space-2) var(--space-4);
    z-index: var(--z-overlay);
  }
  .nav-links--open { display: flex; }
  .nav-link { padding: var(--space-3) var(--space-2); }
  .home-nav { padding: var(--space-3) var(--space-4); position: relative; }
  .docs-hero { padding: var(--space-6) 0 var(--space-4); }
  .docs-content { padding: 0 var(--space-4) var(--space-12); }
  .docs-page { padding: 0; }
}
</style>
