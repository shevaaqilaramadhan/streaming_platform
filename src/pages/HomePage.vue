<template>
  <main class="home-page">
    <!-- Animated background orbs -->
    <div class="orbs" aria-hidden="true">
      <div class="orb orb-1"></div>
      <div class="orb orb-2"></div>
      <div class="orb orb-3"></div>
    </div>

    <!-- Navigation bar -->
    <nav class="home-nav">
      <router-link to="/" class="nav-logo">
        <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
          <circle cx="14" cy="14" r="14" fill="url(#nav-logo-grad)"/>
          <polygon points="11,9 21,14 11,19" fill="white"/>
          <defs>
            <linearGradient id="nav-logo-grad" x1="0" y1="0" x2="28" y2="28">
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
        <router-link to="/status" class="nav-link" @click="mobileMenuOpen = false">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
          </svg>
          Status
        </router-link>
        <a href="https://github.com/litcq/streaming_platform" target="_blank" rel="noopener noreferrer" class="nav-link">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/>
          </svg>
          GitHub
        </a>
      </nav>
    </nav>

    <!-- Hero Section -->
    <section class="hero-section" aria-labelledby="hero-heading">
      <div class="hero-container">
        <!-- Left Column: Content -->
        <div class="hero-content animate-fade-in">
          <h3 class="hero-tagline">
            <span class="tagline-dot"></span>
            Live Sync • No account needed
          </h3>

          <h1 id="hero-heading" class="hero-title">
            <span class="title-line">Watch Video</span>
            <span class="title-line">Together,</span>
            <span class="gradient-text title-line">in sync.</span>
          </h1>

          <p class="hero-subtitle">
            Create a room, share the link, and enjoy perfectly synchronized video
            playback with friends — anywhere in the world.
          </p>

          <!-- CTA actions -->
          <div class="hero-actions">
            <button
              id="create-room-btn"
              class="btn btn-primary btn-lg hero-cta"
              :class="{ 'is-loading': isCreating }"
              :disabled="isCreating"
              @click="createRoom"
            >
              <span v-if="isCreating" class="btn-spinner"></span>
              <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 5v14M5 12h14"/>
              </svg>
              {{ isCreating ? 'Creating room…' : 'Create Watch Room' }}
            </button>

            <span class="actions-divider">or</span>

            <div class="join-group">
              <input
                id="join-room-input"
                v-model="joinRoomId"
                type="text"
                class="join-input"
                placeholder="Room code (e.g. xyz123)"
                maxlength="20"
                spellcheck="false"
                @keydown.enter="joinRoom"
              />
              <button
                id="join-room-submit-btn"
                type="button"
                class="btn btn-ghost join-btn"
                :disabled="!joinRoomId.trim()"
                @click="joinRoom"
              >
                Join
              </button>
            </div>
          </div>

          <!-- Error message -->
          <Transition name="fade">
            <p v-if="errorMsg" class="error-msg" role="alert">{{ errorMsg }}</p>
          </Transition>
        </div>

        <!-- Right Column: Interactive Mock Room Preview -->
        <div class="hero-preview-wrapper animate-fade-in-scale">
          <div class="hero-preview glass">
            <!-- Browser Header -->
            <div class="preview-header">
              <div class="window-controls">
                <span class="dot dot-red"></span>
                <span class="dot dot-yellow"></span>
                <span class="dot dot-green"></span>
              </div>
              <div class="address-bar">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="lock-icon">
                  <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                  <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                </svg>
                <span>watchparty.live/room/anime-night</span>
              </div>
            </div>

            <!-- Browser Content (Mock Room Layout) -->
            <div class="preview-content">
              <!-- Mock Video Player -->
              <div class="mock-player-col">
                <div class="mock-now-watching">
                  <div class="mock-thumbnail"></div>
                  <div class="mock-meta-text">
                    <span class="mock-meta-label">NOW WATCHING</span>
                    <span class="mock-meta-title">Chainsaw Man - Episode 12</span>
                  </div>
                  <span class="mock-status-pill">LIVE SYNCED</span>
                </div>
                
                <div class="mock-video-area">
                  <!-- Abstract stylized video thumbnail representation -->
                  <div class="mock-video-poster">
                    <div class="play-trigger-btn">
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                        <polygon points="5 3 19 12 5 21 5 3"/>
                      </svg>
                    </div>
                    <!-- Stylized audio waves or visual bars inside poster -->
                    <div class="poster-visuals">
                      <span class="bar bar-1"></span>
                      <span class="bar bar-2"></span>
                      <span class="bar bar-3"></span>
                    </div>
                  </div>
                  
                  <!-- Mock Controls -->
                  <div class="mock-controls">
                    <button class="mock-ctrl-btn">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor">
                        <rect x="6" y="4" width="4" height="16"/>
                        <rect x="14" y="4" width="4" height="16"/>
                      </svg>
                    </button>
                    <div class="mock-timeline">
                      <div class="mock-timeline-progress" style="width: 65%;"></div>
                    </div>
                    <span class="mock-time">14:20 / 22:05</span>
                    <button class="mock-ctrl-btn">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                        <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
                        <path d="M19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07"/>
                      </svg>
                    </button>
                  </div>
                </div>
              </div>

              <!-- Mock Chat Panel -->
              <div class="mock-chat-col">
                <div class="mock-chat-header">
                  <span class="mock-chat-title">Room Chat</span>
                  <div class="mock-active-users">
                    <span class="eye-icon">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                        <circle cx="12" cy="12" r="3"/>
                      </svg>
                    </span>
                    <span>3 online</span>
                  </div>
                </div>

                <div class="mock-chat-messages">
                  <div class="mock-msg sender-other">
                    <span class="mock-avatar avatar-blue">A</span>
                    <div class="mock-msg-bubble">
                      <span class="mock-msg-user">Alice</span>
                      <p class="mock-msg-text">The stream quality is awesome! 🍿</p>
                    </div>
                  </div>
                  <div class="mock-msg sender-self">
                    <div class="mock-msg-bubble">
                      <span class="mock-msg-user">You (Host)</span>
                      <p class="mock-msg-text">Yeah, synced perfectly!</p>
                    </div>
                  </div>
                  <div class="mock-msg sender-other">
                    <span class="mock-avatar avatar-pink">B</span>
                    <div class="mock-msg-bubble">
                      <span class="mock-msg-user">Bob</span>
                      <p class="mock-msg-text">Ready for the next episode? 🙌</p>
                    </div>
                  </div>
                </div>

                <div class="mock-chat-input">
                  <input type="text" placeholder="Type a message..." disabled class="mock-input-field" />
                  <button class="mock-send-btn" disabled>
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                      <line x1="22" y1="2" x2="11" y2="13"/>
                      <polygon points="22 2 15 22 11 13 2 9 22 2"/>
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Features section -->
    <section id="features" class="features-section" aria-labelledby="features-heading">
      <h2 id="features-heading" class="section-title">Everything you need for the perfect watch party</h2>
      <div class="features-grid">
        <article
          v-for="feat in features"
          :key="feat.title"
          class="feature-card glass"
        >
          <div class="feature-icon">{{ feat.icon }}</div>
          <h3>{{ feat.title }}</h3>
          <p>{{ feat.desc }}</p>
        </article>
      </div>
    </section>

    <!-- Footer -->
    <footer class="home-footer">
      <div class="footer-container">
        <div class="footer-grid">
          <!-- Column 1: Brand -->
          <div class="footer-col brand-col">
            <div class="footer-logo">
              <svg width="32" height="32" viewBox="0 0 28 28" fill="none" aria-hidden="true">
                <circle cx="14" cy="14" r="14" fill="url(#footer-logo-grad)"/>
                <polygon points="11,9 21,14 11,19" fill="white"/>
                <defs>
                  <linearGradient id="footer-logo-grad" x1="0" y1="0" x2="28" y2="28">
                    <stop offset="0%" stop-color="hsl(195,100%,45%)"/>
                    <stop offset="100%" stop-color="hsl(215,90%,50%)"/>
                  </linearGradient>
                </defs>
              </svg>
              <span class="gradient-text footer-brand">WatchParty</span>
            </div>
            <p class="brand-desc">
              Synchronized video playback with friends anywhere in the world. No sign-up, no plugins, just instant sync.
            </p>
            <div class="footer-socials">
              <a href="https://github.com/litcq" target="_blank" rel="noopener noreferrer" class="social-link" aria-label="GitHub">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4"/>
                  <path d="M9 18c-4.51 2-5-2-7-2"/>
                </svg>
              </a>
              <a href="mailto:support@watchparty.live" class="social-link" aria-label="Email support">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect width="20" height="16" x="2" y="4" rx="2"/>
                  <path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/>
                </svg>
              </a>
            </div>
          </div>

          <!-- Column 2: Features -->
          <div class="footer-col">
            <h4 class="footer-title">Features</h4>
            <ul class="footer-links">
              <li><a href="#features" class="footer-link">Real-time Sync</a></li>
              <li><a href="#features" class="footer-link">Host Controls</a></li>
              <li><a href="#features" class="footer-link">Live Chat</a></li>
              <li><a href="#features" class="footer-link">HLS & YouTube Support</a></li>
            </ul>
          </div>

          <!-- Column 3: Support -->
          <div class="footer-col">
            <h4 class="footer-title">Support</h4>
            <ul class="footer-links">
              <li><router-link to="/docs" class="footer-link">Documentation</router-link></li>
              <li><router-link to="/faq" class="footer-link">FAQ</router-link></li>
              <li><a href="https://github.com/litcq/streaming_platform" target="_blank" rel="noopener noreferrer" class="footer-link">GitHub Repository</a></li>
              <li><router-link to="/status" class="footer-link">Server Status</router-link></li>
            </ul>
          </div>

          <!-- Column 4: Contact/Developer -->
          <div class="footer-col">
            <h4 class="footer-title">Developer</h4>
            <ul class="footer-details">
              <li>
                <span class="detail-label">Email:</span>
                <a href="mailto:support@watchparty.live" class="detail-value">support@watchparty.live</a>
              </li>
              <li>
                <span class="detail-label">GitHub:</span>
                <a href="https://github.com/litcq" target="_blank" rel="noopener noreferrer" class="detail-value">@litcq</a>
              </li>
              <li class="built-with">
                <span class="detail-label">Built with:</span>
                <span class="detail-value">Vite + Vue 3 + Go</span>
              </li>
            </ul>
          </div>
        </div>

        <div class="footer-bottom">
          <p class="copyright">© 2026 WatchParty. All rights reserved.</p>
          <div class="footer-legal">
            <router-link to="/docs" class="footer-link">Privacy Policy</router-link>
            <router-link to="/faq" class="footer-link">Terms of Service</router-link>
          </div>
        </div>
      </div>
    </footer>
  </main>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const isCreating = ref(false)
const joinRoomId = ref('')
const errorMsg   = ref('')
const mobileMenuOpen = ref(false)

/* ---- Create Room ----
 * Calls the Go backend to generate a unique room ID.
 * If the backend is unavailable, generate a local ID for demo.
 */
async function createRoom() {
  isCreating.value = true
  errorMsg.value   = ''

  try {
    const res = await fetch('/api/rooms', { method: 'POST' })
    if (!res.ok) throw new Error('Server error')
    const data = await res.json()
    if (data.hostToken && typeof sessionStorage !== 'undefined') {
      sessionStorage.setItem(`wp_host_${data.roomId}`, data.hostToken)
    }
    router.push({ name: 'room', params: { roomId: data.roomId } })
  } catch {
    const roomId = generateLocalId()
    errorMsg.value = 'Server unavailable — starting a local room. Some features may be limited.'
    setTimeout(() => {
      router.push({ name: 'room', params: { roomId } })
    }, 1500)
  } finally {
    isCreating.value = false
  }
}

function joinRoom() {
  const id = joinRoomId.value.trim()
  if (!id) { errorMsg.value = 'Please enter a room code.'; return }
  errorMsg.value = ''
  router.push({ name: 'room', params: { roomId: id } })
}

function generateLocalId() {
  return Math.random().toString(36).slice(2, 8)
}

/* ---- Static content ---- */
const features = [
  {
    icon: '⚡',
    title: 'Real-time sync',
    desc: 'Millisecond-precision video sync over WebSockets. Play, pause, and seek stay perfectly in step for everyone.',
  },
  {
    icon: '💬',
    title: 'Live chat',
    desc: 'React to every scene together with a built-in chat panel that streams messages instantly to the whole room.',
  },
  {
    icon: '🔒',
    title: 'Host controls',
    desc: 'The room creator is the host. Only they control playback — preventing chaotic desyncs from multiple cooks.',
  },
  {
    icon: '🎬',
    title: 'Any MP4 link',
    desc: 'Paste any public direct video URL. No plugins, no extensions — just the native HTML5 video element.',
  },
  {
    icon: '👤',
    title: 'No sign-up',
    desc: 'Guests just pick a nickname and join. Zero friction, zero account required.',
  },
  {
    icon: '🌐',
    title: 'Share by link',
    desc: 'The room URL is the invite. Copy it, share it, and everyone\'s in within seconds.',
  },
]

</script>

<style scoped>
.home-page {
  min-height: 100vh;
  position: relative;
  overflow-x: hidden;
}

/* Orbs */
.orbs { position: fixed; inset: 0; pointer-events: none; z-index: 0; overflow: hidden; }
.orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.35;
}
.orb-1 {
  width: 600px; height: 600px;
  background: radial-gradient(circle, hsl(195,100%,45%) 0%, transparent 70%);
  top: -200px; left: -200px;
  animation: orb-move-1 18s ease-in-out infinite;
}
.orb-2 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, hsl(215,90%,50%) 0%, transparent 70%);
  bottom: -100px; right: -150px;
  animation: orb-move-2 22s ease-in-out infinite;
}
.orb-3 {
  width: 350px; height: 350px;
  background: radial-gradient(circle, hsl(205,95%,48%) 0%, transparent 70%);
  top: 40%; left: 55%;
  animation: orb-move-1 28s ease-in-out infinite reverse;
  opacity: 0.2;
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

.nav-links {
  display: flex;
  align-items: center;
  gap: var(--space-1);
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
  background: rgba(255, 255, 255, 0.05);
}

.nav-link {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all var(--transition-fast);
  text-decoration: none;
}
.nav-link:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.05);
}
.nav-link svg { opacity: 0.7; }
.nav-link:hover svg { opacity: 1; }

.nav-right {
  display: flex;
  align-items: center;
  gap: var(--space-6);
}

.nav-profile-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--color-glass);
  border: 1px solid var(--color-glass-border);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-base);
}
.nav-profile-btn:hover {
  background: var(--color-glass-hover);
  color: var(--text-primary);
  border-color: rgba(255, 255, 255, 0.15);
  box-shadow: var(--shadow-glow-sm);
  transform: translateY(-1px);
}

/* Hero Section Layout */
.hero-section {
  position: relative;
  z-index: 1;
  min-height: 90vh;
  display: flex;
  align-items: center;
  padding: var(--space-6) var(--space-6) var(--space-12);
  max-width: 1360px;
  margin: 0 auto;
}

.hero-container {
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  gap: 80px;
  align-items: center;
  width: 100%;
}

.hero-content {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  text-align: left;
  padding-left: var(--space-6);
}

.hero-tagline {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--color-primary);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: var(--space-4);
}
.tagline-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-primary);
  box-shadow: 0 0 10px var(--color-primary);
  animation: pulse-glow 2s infinite;
}

.hero-title {
  font-size: clamp(2.4rem, 4.8vw, 3.8rem);
  line-height: 1.15;
  letter-spacing: -0.04em;
  margin-bottom: var(--space-4);
}

.title-line {
  display: block;
}

@media (min-width: 901px) {
  .title-line {
    white-space: nowrap;
  }
}

.hero-subtitle {
  font-size: clamp(1.1rem, 2vw, 1.3rem);
  color: var(--text-secondary);
  max-width: 520px;
  line-height: 1.6;
  margin-bottom: var(--space-8);
}

.hero-actions {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  width: 100%;
  max-width: 600px;
}

.hero-cta {
  flex-shrink: 0;
  height: 48px;
}

.actions-divider {
  font-size: 0.9rem;
  color: var(--text-muted);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  user-select: none;
}

.join-group {
  display: flex;
  align-items: center;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-md);
  padding: 4px;
  flex: 1;
  max-width: 320px;
  height: 48px;
  transition: all var(--transition-base);
}

.join-group:focus-within {
  border-color: var(--color-primary);
  background: rgba(255, 255, 255, 0.06);
  box-shadow: 0 0 0 3px hsla(195, 100%, 45%, 0.15);
}

.join-input {
  border: none;
  background: transparent;
  padding: 0 var(--space-3);
  font-family: var(--font-sans);
  font-size: 0.9375rem;
  flex: 1;
  color: var(--text-primary);
  outline: none;
  width: 100%;
}
.join-input::placeholder {
  color: var(--text-muted);
}

.join-btn {
  padding: 0 var(--space-4);
  font-size: 0.875rem;
  font-weight: 600;
  border-radius: var(--radius-sm);
  background: var(--color-glass);
  border: 1px solid var(--color-glass-border);
  color: var(--text-secondary);
  height: 38px;
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  transition: all var(--transition-base);
}
.join-btn:hover:not(:disabled) {
  background: var(--color-glass-hover);
  color: var(--text-primary);
  border-color: rgba(255, 255, 255, 0.15);
}
.join-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-spinner {
  width: 16px; height: 16px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin-slow 0.7s linear infinite;
}

.error-msg {
  color: var(--color-accent-red);
  font-size: 0.875rem;
  text-align: left;
  margin-top: calc(-1 * var(--space-2));
}

/* Mock Room Preview */
.hero-preview-wrapper {
  position: relative;
  width: 100%;
}

/* Subtle glow behind preview */
.hero-preview-wrapper::before {
  content: '';
  position: absolute;
  inset: -15px;
  border-radius: var(--radius-xl);
  background: var(--grad-brand);
  opacity: 0.06;
  filter: blur(35px);
  pointer-events: none;
  z-index: -1;
}

.hero-preview {
  border-radius: var(--radius-xl);
  border: 1px solid var(--color-glass-border);
  box-shadow: 0 25px 60px rgba(0, 0, 0, 0.6), var(--shadow-glow-sm);
  overflow: hidden;
  transition: all var(--transition-slow);
  background: rgba(10, 11, 16, 0.4);
  animation: float 6s ease-in-out infinite;
  width: 100%;
  max-width: 720px;
}

@media (min-width: 901px) {
  .hero-preview {
    width: 680px;
  }
}

.hero-preview:hover {
  transform: translateY(-4px) scale(1.01);
  border-color: rgba(255, 255, 255, 0.1);
  box-shadow: 0 30px 70px rgba(0, 0, 0, 0.75), var(--shadow-glow-primary);
}

/* Browser Window Chrome */
.preview-header {
  display: flex;
  align-items: center;
  padding: var(--space-3) var(--space-4);
  background: rgba(25, 28, 40, 0.5);
  border-bottom: 1px solid var(--color-glass-border);
  gap: var(--space-4);
}

.window-controls {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

.dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
}
.dot-red { background-color: #ef4444; }
.dot-yellow { background-color: #f59e0b; }
.dot-green { background-color: #10b981; }

.address-bar {
  flex: 1;
  background: rgba(10, 11, 16, 0.6);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-glass-border);
  font-family: var(--font-mono);
  font-size: 0.85rem;
  color: var(--text-secondary);
  padding: 4px var(--space-3);
  display: flex;
  align-items: center;
  gap: var(--space-2);
  max-width: 340px;
}
.lock-icon {
  color: var(--color-accent-green);
  opacity: 0.85;
}

/* Simulated Interface Content */
.preview-content {
  display: grid;
  grid-template-columns: 1.45fr 1fr;
  height: 360px;
  background: rgba(10, 11, 16, 0.2);
}

/* Mock Player Area */
.mock-player-col {
  border-right: 1px solid var(--color-glass-border);
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  background: rgba(10, 11, 16, 0.15);
}

.mock-now-watching {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: rgba(25, 28, 40, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.03);
  padding: 6px 8px;
  border-radius: var(--radius-md);
  border-left: 2px solid var(--color-primary);
}

.mock-thumbnail {
  width: 32px;
  height: 44px;
  background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
  border-radius: 4px;
  border: 1px solid var(--color-glass-border);
  position: relative;
  overflow: hidden;
}
.mock-thumbnail::before {
  content: '🎬';
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.8rem;
}

.mock-meta-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  flex: 1;
  min-width: 0;
}

.mock-meta-label {
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--color-primary);
  letter-spacing: 0.05em;
}

.mock-meta-title {
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mock-status-pill {
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--color-accent-green);
  background: rgba(16, 185, 129, 0.1);
  padding: 2px 6px;
  border-radius: var(--radius-full);
  letter-spacing: 0.05em;
  flex-shrink: 0;
}

.mock-video-area {
  flex: 1;
  background: #020205;
  border-radius: var(--radius-md);
  position: relative;
  overflow: hidden;
  border: 1px solid var(--color-glass-border);
  display: flex;
  flex-direction: column;
}

.mock-video-poster {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  background: linear-gradient(135deg, rgba(14, 16, 26, 0.9) 0%, rgba(20, 24, 40, 0.95) 100%);
}

/* Decorative backdrop representation of a playing video frame */
.mock-video-poster::before {
  content: '';
  position: absolute;
  inset: 10px;
  border-radius: var(--radius-sm);
  background: radial-gradient(circle at center, rgba(0, 210, 255, 0.06) 0%, transparent 60%);
  z-index: 1;
}

.play-trigger-btn {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--grad-brand);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--shadow-glow-sm);
  position: relative;
  z-index: 2;
  transition: all var(--transition-base);
}

.hero-preview:hover .play-trigger-btn {
  transform: scale(1.08);
  box-shadow: var(--shadow-glow-primary);
}

/* Glowing abstract equalizer-like bars to denote content activity */
.poster-visuals {
  position: absolute;
  bottom: var(--space-2);
  right: var(--space-3);
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 12px;
  z-index: 2;
  opacity: 0.7;
}
.poster-visuals .bar {
  width: 2px;
  border-radius: 1px;
  background: var(--color-primary);
}
.bar-1 { height: 60%; animation: mock-eq 1.2s ease infinite alternate; }
.bar-2 { height: 100%; animation: mock-eq 0.8s ease infinite alternate 0.2s; }
.bar-3 { height: 40%; animation: mock-eq 1s ease infinite alternate 0.4s; }

@keyframes mock-eq {
  0% { height: 20%; }
  100% { height: 100%; }
}

.mock-controls {
  height: 32px;
  background: rgba(14, 16, 26, 0.95);
  border-top: 1px solid var(--color-glass-border);
  display: flex;
  align-items: center;
  padding: 0 var(--space-2);
  gap: var(--space-2);
  z-index: 2;
}

.mock-ctrl-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  cursor: default;
}

.mock-timeline {
  flex: 1;
  height: 3px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-full);
  overflow: hidden;
  position: relative;
}

.mock-timeline-progress {
  height: 100%;
  background: var(--color-primary);
  border-radius: var(--radius-full);
}

.mock-time {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--text-muted);
  white-space: nowrap;
}

/* Mock Chat Area */
.mock-chat-col {
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  background: rgba(25, 28, 40, 0.1);
}

.mock-chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--color-glass-border);
}

.mock-chat-title {
  font-size: 0.92rem;
  font-weight: 700;
  color: var(--text-primary);
}

.mock-active-users {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.78rem;
  color: var(--text-muted);
}
.eye-icon {
  display: flex;
  align-items: center;
  color: var(--color-primary);
}

.mock-chat-messages {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  overflow: hidden;
  padding: var(--space-1) 0;
}

.mock-msg {
  display: flex;
  align-items: flex-end;
  gap: var(--space-2);
  max-width: 90%;
  animation: float-bubble 4s ease-in-out infinite alternate;
}
.mock-msg.sender-other {
  align-self: flex-start;
}
.mock-msg.sender-self {
  align-self: flex-end;
  max-width: 80%;
}

.mock-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  font-size: 0.75rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
  margin-bottom: 2px;
}
.avatar-blue { background: hsl(195, 90%, 50%); }
.avatar-pink { background: hsl(330, 95%, 60%); }

.mock-msg-bubble {
  background: rgba(25, 28, 40, 0.4);
  border: 1px solid var(--color-glass-border);
  padding: 6px 8px;
  border-radius: 10px 10px 10px 3px;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.sender-self .mock-msg-bubble {
  background: var(--grad-brand-subtle);
  border-color: rgba(0, 210, 255, 0.15);
  border-radius: 10px 10px 3px 10px;
}

.mock-msg-user {
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--text-muted);
}
.sender-self .mock-msg-user {
  color: var(--color-primary);
}

.mock-msg-text {
  font-size: 0.85rem;
  line-height: 1.3;
  color: var(--text-secondary);
}
.sender-self .mock-msg-text {
  color: var(--text-primary);
}

/* Subtle float bubbles animations to look alive */
@keyframes float-bubble {
  0% { transform: translateY(0); }
  100% { transform: translateY(-3px); }
}

.mock-chat-input {
  display: flex;
  gap: var(--space-2);
  margin-top: auto;
}

.mock-input-field {
  flex: 1;
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-sm);
  padding: 4px var(--space-2);
  font-size: 0.85rem;
  color: var(--text-muted);
  cursor: default;
}

.mock-send-btn {
  background: var(--color-glass);
  border: 1px solid var(--color-glass-border);
  color: var(--text-muted);
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: default;
}

/* Features */
.features-section {
  position: relative; z-index: 1;
  padding: var(--space-16) var(--space-6);
  max-width: 1100px; margin: 0 auto;
}

.section-title {
  font-size: clamp(1.5rem, 3vw, 2.2rem);
  text-align: center;
  margin-bottom: var(--space-10);
  letter-spacing: -0.03em;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: var(--space-5);
}

.feature-card {
  padding: var(--space-6);
  border-radius: var(--radius-lg);
  display: flex; flex-direction: column; gap: var(--space-3);
  transition: all var(--transition-base);
}
.feature-card:hover {
  background: var(--color-glass-hover);
  border-color: rgba(255,255,255,0.12);
  transform: translateY(-2px);
  box-shadow: var(--shadow-card);
}

.feature-icon { font-size: 2rem; }

.feature-card h3 {
  font-size: 1rem; font-weight: 700; color: var(--text-primary);
}
.feature-card p { font-size: 0.9rem; color: var(--text-secondary); line-height: 1.6; }

/* Footer */
.home-footer {
  position: relative;
  z-index: 1;
  border-top: 1px solid var(--color-glass-border);
  background: linear-gradient(180deg, transparent 0%, rgba(10, 11, 16, 0.5) 100%);
  padding: var(--space-12) 0 var(--space-8);
  margin-top: var(--space-16);
}

.footer-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 var(--space-6);
}

.footer-grid {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1.5fr;
  gap: var(--space-10);
  margin-bottom: var(--space-12);
}

.footer-col {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  text-align: left;
}

.brand-col {
  padding-right: var(--space-8);
}

.footer-logo {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.footer-brand {
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.brand-desc {
  font-size: 0.9rem;
  color: var(--text-secondary);
  line-height: 1.6;
}

.footer-socials {
  display: flex;
  gap: var(--space-3);
  margin-top: var(--space-2);
}

.social-link {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  background: var(--color-glass);
  border: 1px solid var(--color-glass-border);
  color: var(--text-secondary);
  transition: all var(--transition-base);
}

.social-link:hover {
  background: var(--color-glass-hover);
  color: var(--text-primary);
  border-color: rgba(255, 255, 255, 0.15);
  transform: translateY(-2px);
}

.footer-title {
  font-size: 0.95rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-primary);
  margin-bottom: var(--space-2);
}

.footer-links {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: 0;
  margin: 0;
}

.footer-link {
  font-size: 0.9rem;
  color: var(--text-secondary);
  transition: color var(--transition-fast);
  text-decoration: none;
  display: inline-block;
}

.footer-link:hover {
  color: var(--text-primary);
}

.footer-details {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: 0;
  margin: 0;
}

.footer-details li {
  font-size: 0.9rem;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.detail-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.detail-value {
  color: var(--text-secondary);
  transition: color var(--transition-fast);
}

.detail-value[href]:hover {
  color: var(--text-primary);
}

.built-with .detail-value {
  color: var(--color-primary);
  font-weight: 500;
}

.footer-bottom {
  border-top: 1px solid var(--color-glass-border);
  padding-top: var(--space-6);
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-4);
}

.copyright {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 0;
}

.footer-legal {
  display: flex;
  gap: var(--space-6);
}

@media (max-width: 900px) {
  .footer-grid {
    grid-template-columns: 1fr 1fr;
    gap: var(--space-8);
  }
  .brand-col {
    grid-column: span 2;
    padding-right: 0;
  }
}

@media (max-width: 600px) {
  .footer-grid {
    grid-template-columns: 1fr;
    gap: var(--space-6);
  }
  .brand-col {
    grid-column: span 1;
  }
  .footer-bottom {
    flex-direction: column;
    align-items: center;
    text-align: center;
  }
}


@media (max-width: 900px) {
  .hero-section {
    min-height: auto;
    padding: var(--space-8) var(--space-4) var(--space-12);
  }
  .hero-container {
    grid-template-columns: 1fr;
    gap: var(--space-8);
    text-align: center;
  }
  .hero-content {
    align-items: center;
    text-align: center;
  }
  .hero-actions {
    flex-direction: column;
    gap: var(--space-3);
    margin: 0 auto;
    width: 100%;
    max-width: 420px;
  }
  .hero-cta {
    width: 100%;
  }
  .join-group {
    width: 100%;
    max-width: 100%;
  }
  .actions-divider {
    display: none;
  }
  .error-msg {
    text-align: center;
  }
  .hero-preview-wrapper {
    max-width: 550px;
    margin: 0 auto;
  }
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
  .hero-section {
    min-height: auto;
    padding: var(--space-6) var(--space-4) var(--space-10);
  }
  .preview-content {
    grid-template-columns: 1.2fr 1fr;
  }
}
</style>
