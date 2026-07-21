<template>
  <main class="home-page">
    <!-- Navigation -->
    <nav class="home-nav">
      <router-link to="/" class="nav-logo">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
          <rect x="1" y="1" width="22" height="22" rx="5" stroke="currentColor" stroke-width="1.5"/>
          <polygon points="10,7 18,12 10,17" fill="currentColor"/>
        </svg>
        <span>WatchParty</span>
      </router-link>
      <nav class="nav-links" :class="{ 'nav-links--open': mobileMenuOpen }" aria-label="Main">
        <router-link to="/docs" class="nav-link" @click="mobileMenuOpen = false">Docs</router-link>
        <router-link to="/faq" class="nav-link" @click="mobileMenuOpen = false">FAQ</router-link>
        <router-link to="/status" class="nav-link" @click="mobileMenuOpen = false">Status</router-link>
        <a href="https://github.com/litcq/streaming_platform" target="_blank" rel="noopener noreferrer" class="nav-link">GitHub</a>
      </nav>
      <div class="nav-right">
        <ThemeToggle />
        <button class="hamburger-btn" aria-label="Toggle menu" @click="mobileMenuOpen = !mobileMenuOpen">
          <svg v-if="!mobileMenuOpen" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
          <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>
    </nav>

    <!-- Flash banner -->
    <Transition name="slide-down">
      <div v-if="flashMsg" class="flash-banner" role="alert">
        <span>{{ flashMsg }}</span>
        <button class="flash-dismiss" @click="flashMsg = ''">&times;</button>
      </div>
    </Transition>

    <!-- Hero -->
    <section class="hero">
      <div class="hero-grid">
        <!-- Left: copy + actions -->
        <div class="hero-inner">
          <p class="hero-label">Watch together, perfectly in sync</p>
          <h1 class="hero-title">The simplest way to<br/>watch videos with friends.</h1>
          <p class="hero-desc">Create a room, paste a video link, and watch in perfect synchronization with anyone, anywhere. No sign-up, no plugins — just press play.</p>
          <div class="hero-actions">
            <button class="btn btn-primary btn-lg" :disabled="isCreating" @click="createRoom">
              <span v-if="isCreating" class="spinner-sm"></span>
              <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
              {{ isCreating ? 'Creating…' : 'Create Room' }}
            </button>
            <div class="hero-join">
              <input v-model="joinRoomId" type="text" placeholder="Enter room code" maxlength="20" spellcheck="false" @keydown.enter="joinRoom" />
              <button class="btn btn-ghost" :disabled="!joinRoomId.trim()" @click="joinRoom">Join</button>
            </div>
          </div>
          <Transition name="fade">
            <p v-if="errorMsg" class="hero-error" role="alert">{{ errorMsg }}</p>
          </Transition>
        </div>

        <!-- Right: mock room preview (aesthetic mini screen) -->
        <div class="hero-preview-wrapper animate-fade-in-scale" aria-hidden="true">
          <div class="hero-preview glass">
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

            <div class="preview-content">
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
                  <div class="mock-video-poster">
                    <div class="play-trigger-btn">
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                        <polygon points="5 3 19 12 5 21 5 3"/>
                      </svg>
                    </div>
                    <div class="poster-visuals">
                      <span class="bar bar-1"></span>
                      <span class="bar bar-2"></span>
                      <span class="bar bar-3"></span>
                    </div>
                  </div>

                  <div class="mock-controls">
                    <button class="mock-ctrl-btn" tabindex="-1">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor">
                        <rect x="6" y="4" width="4" height="16"/>
                        <rect x="14" y="4" width="4" height="16"/>
                      </svg>
                    </button>
                    <div class="mock-timeline">
                      <div class="mock-timeline-progress" style="width: 65%;"></div>
                    </div>
                    <span class="mock-time">14:20 / 22:05</span>
                    <button class="mock-ctrl-btn" tabindex="-1">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                        <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
                        <path d="M19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07"/>
                      </svg>
                    </button>
                  </div>
                </div>
              </div>

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
                  <input type="text" placeholder="Type a message..." disabled class="mock-input-field" tabindex="-1" />
                  <button class="mock-send-btn" disabled tabindex="-1">
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

    <!-- Features -->
    <section class="features" id="features">
      <div class="features-inner">
        <h2 class="section-heading">Everything you need.</h2>
        <p class="section-sub">No bloat, no complexity — just the essentials for a perfect watch party.</p>
        <div class="features-grid">
          <div v-for="f in features" :key="f.title" class="feature-card">
            <div class="feature-icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <path :d="f.icon"/>
              </svg>
            </div>
            <h3>{{ f.title }}</h3>
            <p>{{ f.desc }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- How it works -->
    <section class="how">
      <div class="how-inner">
        <h2 class="section-heading">Get started in seconds.</h2>
        <div class="steps">
          <div v-for="(s, i) in steps" :key="i" class="step">
            <span class="step-num">{{ String(i + 1).padStart(2, '0') }}</span>
            <h3>{{ s.title }}</h3>
            <p>{{ s.desc }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- CTA -->
    <section class="cta-section">
      <div class="cta-inner">
        <h2>Ready to watch together?</h2>
        <p>Create a room in one click and share the link with friends.</p>
        <button class="btn btn-primary btn-lg" :disabled="isCreating" @click="createRoom">
          {{ isCreating ? 'Creating…' : 'Get Started' }}
        </button>
      </div>
    </section>

    <!-- Footer -->
    <footer class="home-footer">
      <div class="footer-inner">
        <div class="footer-top">
          <div class="footer-brand">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none"><rect x="1" y="1" width="22" height="22" rx="5" stroke="currentColor" stroke-width="1.5" opacity="0.4"/><polygon points="10,7 18,12 10,17" fill="currentColor" opacity="0.4"/></svg>
            <span>WatchParty</span>
          </div>
          <div class="footer-links">
            <router-link to="/docs">Docs</router-link>
            <router-link to="/faq">FAQ</router-link>
            <router-link to="/status">Status</router-link>
            <a href="https://github.com/litcq/streaming_platform" target="_blank" rel="noopener">GitHub</a>
          </div>
        </div>
        <div class="footer-bottom">
          <p>© 2026 WatchParty. All rights reserved.</p>
        </div>
      </div>
    </footer>
  </main>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import ThemeToggle from '../components/ThemeToggle.vue'

const router = useRouter()
const route = useRoute()
const isCreating = ref(false)
const joinRoomId = ref('')
const errorMsg = ref('')
const flashMsg = ref('')
const mobileMenuOpen = ref(false)

onMounted(() => {
  try {
    const raw = sessionStorage.getItem('wp_flash')
    if (raw) {
      sessionStorage.removeItem('wp_flash')
      const data = JSON.parse(raw)
      if (data?.message) flashMsg.value = data.message
    }
  } catch {}
  if (!flashMsg.value && (route.query.kicked === '1' || route.query.kicked === 'true')) {
    flashMsg.value = 'You have been removed from the room.'
  }
  if (flashMsg.value && route.query.kicked) router.replace({ name: 'home', query: {} })
  if (flashMsg.value) setTimeout(() => { flashMsg.value = '' }, 8000)
})

async function createRoom() {
  isCreating.value = true
  errorMsg.value = ''
  try {
    const res = await fetch('/api/rooms', { method: 'POST' })
    if (!res.ok) throw new Error('Server error')
    const data = await res.json()
    if (data.hostToken && typeof sessionStorage !== 'undefined') {
      sessionStorage.setItem(`wp_host_${data.roomId}`, data.hostToken)
    }
    router.push({ name: 'room', params: { roomId: data.roomId } })
  } catch {
    const roomId = Math.random().toString(36).slice(2, 8)
    errorMsg.value = 'Server unavailable — starting a local room.'
    setTimeout(() => router.push({ name: 'room', params: { roomId } }), 1500)
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

const features = [
  { icon: 'M21.5 2v6h-6M2.5 22v-6h6M2.5 11.5a10 10 0 0 1 18.18-4.5M21.5 12.5a10 10 0 0 1-18.18 4.5', title: 'Real-time sync', desc: 'Millisecond-precision synchronization over WebSockets. Play, pause, and seek stay perfectly in step.' },
  { icon: 'M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z', title: 'Live chat', desc: 'Built-in chat for reacting to scenes together. Messages stream instantly to everyone in the room.' },
  { icon: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z', title: 'Host controls', desc: 'Only the room creator controls playback — preventing desyncs from conflicting inputs.' },
  { icon: 'M23 7l-7 5 7 5V7z M1 5h15v14H1z', title: 'Any video source', desc: 'Paste any YouTube, .mp4, or .m3u8 link. Native HTML5 playback with no plugins.' },
  { icon: 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2 M12 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8z', title: 'No sign-up', desc: 'Just pick a nickname and join. Zero friction, zero accounts.' },
  { icon: 'M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71 M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71', title: 'Share by link', desc: 'The room URL is the invite. Copy it, share it — everyone\'s in within seconds.' },
]

const steps = [
  { title: 'Create a room', desc: 'One click. Your room is ready instantly with a unique code.' },
  { title: 'Set your name', desc: 'Pick a nickname so others know who you are. No account needed.' },
  { title: 'Paste a video link', desc: 'YouTube, direct MP4, or any stream URL. The host controls playback.' },
  { title: 'Invite & watch', desc: 'Share the room link and enjoy in perfect sync with friends.' },
]
</script>

<style scoped>
.home-page {
  min-height: 100vh;
  position: relative;
}

/* Nav */
.home-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--nav-height);
  padding: 0 var(--space-8);
  position: sticky;
  top: 0;
  z-index: var(--z-overlay);
  background: rgba(10,10,10,0.85);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--color-border);
}
.nav-logo {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  text-decoration: none;
  color: var(--text-primary);
  font-weight: 600;
  font-size: 0.9375rem;
  letter-spacing: -0.02em;
}
.nav-links {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  margin-left: auto;
}
.nav-link {
  display: inline-flex;
  align-items: center;
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all var(--transition-fast);
  text-decoration: none;
}
.nav-link:hover {
  color: var(--text-primary);
  background: var(--color-fill);
}
.nav-right {
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
  border: 1px solid var(--color-border);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.hamburger-btn:hover {
  color: var(--text-primary);
  background: var(--color-fill);
}

/* Flash */
.flash-banner {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin: var(--space-3) var(--space-8) 0;
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  font-size: 0.875rem;
  color: var(--text-secondary);
}
.flash-dismiss {
  margin-left: auto;
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1.25rem;
  cursor: pointer;
}
.flash-dismiss:hover { color: var(--text-primary); }
.slide-down-enter-active, .slide-down-leave-active { transition: all 0.25s ease; }
.slide-down-enter-from, .slide-down-leave-to { opacity: 0; transform: translateY(-8px); }

/* Hero */
.hero {
  padding: var(--space-20) var(--space-8) var(--space-16);
  max-width: var(--max-width);
  margin: 0 auto;
}
.hero-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-12);
  align-items: center;
}
.hero-inner {
  max-width: 560px;
}
.hero-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: var(--space-5);
}
.hero-title {
  font-size: clamp(2.25rem, 4.5vw, 3.5rem);
  font-weight: 600;
  line-height: 1.1;
  letter-spacing: -0.04em;
  margin-bottom: var(--space-6);
  color: var(--text-primary);
}
.hero-desc {
  font-size: clamp(1rem, 1.6vw, 1.125rem);
  color: var(--text-secondary);
  line-height: 1.7;
  margin-bottom: var(--space-10);
  max-width: 520px;
}
.hero-actions {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  flex-wrap: wrap;
}
.hero-join {
  display: flex;
  align-items: center;
  background: var(--color-fill);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: 4px;
  height: 48px;
  transition: all var(--transition-base);
}
.hero-join:focus-within {
  border-color: var(--color-border-strong);
}
.hero-join input {
  border: none;
  background: transparent;
  padding: 0 var(--space-3);
  font-family: var(--font-sans);
  font-size: 0.875rem;
  color: var(--text-primary);
  outline: none;
  width: 180px;
}
.hero-join input::placeholder { color: var(--text-muted); }
.hero-join .btn {
  height: 38px;
}
.hero-error {
  margin-top: var(--space-4);
  color: var(--color-accent-red);
  font-size: 0.8125rem;
}
.spinner-sm {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(0,0,0,0.15);
  border-top-color: #0a0a0a;
  border-radius: 50%;
  animation: spin-slow 0.7s linear infinite;
}

/* ── Mock Room Preview (mini screen) ── */
.hero-preview-wrapper {
  position: relative;
  width: 100%;
}
.hero-preview-wrapper::before {
  content: '';
  position: absolute;
  inset: -12px;
  border-radius: var(--radius-xl);
  background: var(--grad-brand);
  opacity: 0.04;
  filter: blur(28px);
  pointer-events: none;
  z-index: -1;
}
.hero-preview {
  border-radius: var(--radius-xl);
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
  transition: border-color var(--transition-slow), box-shadow var(--transition-slow);
  background: var(--color-bg-surface);
  animation: hero-float 6s ease-in-out infinite;
  width: 100%;
}
.hero-preview:hover {
  border-color: var(--color-border-strong);
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.45);
}

@keyframes hero-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-6px); }
}

/* Browser chrome */
.preview-header {
  display: flex;
  align-items: center;
  padding: var(--space-3) var(--space-4);
  background: var(--color-bg-elevated);
  border-bottom: 1px solid var(--color-border);
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
  background: var(--color-bg-base);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--text-secondary);
  padding: 4px var(--space-3);
  display: flex;
  align-items: center;
  gap: var(--space-2);
  max-width: 320px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.lock-icon {
  color: var(--color-accent-green);
  opacity: 0.85;
  flex-shrink: 0;
}

/* Content grid: player + chat */
.preview-content {
  display: grid;
  grid-template-columns: 1.45fr 1fr;
  height: 340px;
  background: var(--color-bg-base);
}

/* Player column */
.mock-player-col {
  border-right: 1px solid var(--color-border);
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.mock-now-watching {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: var(--color-fill);
  border: 1px solid var(--color-border);
  padding: 6px 8px;
  border-radius: var(--radius-md);
  border-left: 2px solid var(--text-primary);
}
.mock-thumbnail {
  width: 28px;
  height: 40px;
  background: linear-gradient(135deg, var(--color-bg-elevated) 0%, var(--color-bg-base) 100%);
  border-radius: 4px;
  border: 1px solid var(--color-border);
  position: relative;
  overflow: hidden;
  flex-shrink: 0;
}
.mock-thumbnail::before {
  content: '🎬';
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.7rem;
}
.mock-meta-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  flex: 1;
  min-width: 0;
}
.mock-meta-label {
  font-size: 0.625rem;
  font-weight: 700;
  color: var(--text-muted);
  letter-spacing: 0.06em;
}
.mock-meta-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mock-status-pill {
  font-size: 0.625rem;
  font-weight: 700;
  color: var(--color-accent-green);
  background: rgba(74, 222, 128, 0.1);
  padding: 2px 6px;
  border-radius: var(--radius-full);
  letter-spacing: 0.04em;
  flex-shrink: 0;
}

.mock-video-area {
  flex: 1;
  background: var(--color-bg-base);
  border-radius: var(--radius-md);
  position: relative;
  overflow: hidden;
  border: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
}
.mock-video-poster {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  background: linear-gradient(135deg, var(--color-bg-base) 0%, var(--color-bg-elevated) 100%);
}
.mock-video-poster::before {
  content: '';
  position: absolute;
  inset: 10px;
  border-radius: var(--radius-sm);
  background: radial-gradient(circle at center, var(--color-fill-hover) 0%, transparent 60%);
  z-index: 1;
}
.play-trigger-btn {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: var(--text-primary);
  color: var(--text-inverse);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--shadow-card);
  position: relative;
  z-index: 2;
  transition: all var(--transition-base);
}
.hero-preview:hover .play-trigger-btn {
  transform: scale(1.08);
}

.poster-visuals {
  position: absolute;
  bottom: var(--space-2);
  right: var(--space-3);
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 12px;
  z-index: 2;
  opacity: 0.6;
}
.poster-visuals .bar {
  width: 2px;
  border-radius: 1px;
  background: var(--text-primary);
}
.bar-1 { height: 60%; animation: mock-eq 1.2s ease infinite alternate; }
.bar-2 { height: 100%; animation: mock-eq 0.8s ease infinite alternate 0.2s; }
.bar-3 { height: 40%; animation: mock-eq 1s ease infinite alternate 0.4s; }

@keyframes mock-eq {
  0% { height: 20%; }
  100% { height: 100%; }
}

.mock-controls {
  height: 30px;
  background: var(--color-bg-elevated);
  border-top: 1px solid var(--color-border);
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
  background: var(--color-fill-hover);
  border-radius: var(--radius-full);
  overflow: hidden;
}
.mock-timeline-progress {
  height: 100%;
  background: var(--text-primary);
  border-radius: var(--radius-full);
}
.mock-time {
  font-family: var(--font-mono);
  font-size: 0.65rem;
  color: var(--text-muted);
  white-space: nowrap;
}

/* Chat column */
.mock-chat-col {
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  background: var(--color-bg-surface);
}
.mock-chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--color-border);
}
.mock-chat-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-primary);
}
.mock-active-users {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.7rem;
  color: var(--text-muted);
}
.eye-icon {
  display: flex;
  align-items: center;
  color: var(--text-secondary);
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
  max-width: 92%;
  animation: float-bubble 4s ease-in-out infinite alternate;
}
.mock-msg.sender-other { align-self: flex-start; }
.mock-msg.sender-self {
  align-self: flex-end;
  max-width: 85%;
}

.mock-avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  font-size: 0.65rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
  margin-bottom: 2px;
}
.avatar-blue { background: #3b82f6; }
.avatar-pink { background: #ec4899; }

.mock-msg-bubble {
  background: var(--color-fill);
  border: 1px solid var(--color-border);
  padding: 5px 8px;
  border-radius: 10px 10px 10px 3px;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.sender-self .mock-msg-bubble {
  background: var(--grad-brand-subtle);
  border-color: var(--color-border-strong);
  border-radius: 10px 10px 3px 10px;
}
.mock-msg-user {
  font-size: 0.65rem;
  font-weight: 700;
  color: var(--text-muted);
}
.sender-self .mock-msg-user {
  color: var(--text-secondary);
}
.mock-msg-text {
  font-size: 0.75rem;
  line-height: 1.35;
  color: var(--text-secondary);
  margin: 0;
}
.sender-self .mock-msg-text {
  color: var(--text-primary);
}

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
  background: var(--color-bg-base);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: 4px var(--space-2);
  font-size: 0.75rem;
  color: var(--text-muted);
  cursor: default;
  font-family: var(--font-sans);
}
.mock-send-btn {
  background: var(--color-fill);
  border: 1px solid var(--color-border);
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
.features {
  padding: var(--space-20) var(--space-8);
  border-top: 1px solid var(--color-border);
}
.features-inner {
  max-width: var(--max-width);
  margin: 0 auto;
}
.section-heading {
  font-size: clamp(1.5rem, 3vw, 2.25rem);
  letter-spacing: -0.03em;
  margin-bottom: var(--space-3);
}
.section-sub {
  font-size: 1rem;
  color: var(--text-secondary);
  margin-bottom: var(--space-12);
  max-width: 480px;
}
.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1px;
  background: var(--color-border);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}
.feature-card {
  background: var(--color-bg-base);
  padding: var(--space-8);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  transition: background var(--transition-base);
}
.feature-card:hover {
  background: var(--color-bg-surface);
}
.feature-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  background: var(--color-fill);
  border: 1px solid var(--color-border);
  color: var(--text-secondary);
}
.feature-card h3 {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text-primary);
}
.feature-card p {
  font-size: 0.8125rem;
  color: var(--text-secondary);
  line-height: 1.6;
}

/* How it works */
.how {
  padding: var(--space-20) var(--space-8);
  border-top: 1px solid var(--color-border);
}
.how-inner {
  max-width: var(--max-width);
  margin: 0 auto;
}
.steps {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-8);
  margin-top: var(--space-12);
}
.step {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.step-num {
  font-size: 0.75rem;
  font-weight: 600;
  font-family: var(--font-mono);
  color: var(--text-muted);
}
.step h3 {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text-primary);
}
.step p {
  font-size: 0.8125rem;
  color: var(--text-secondary);
  line-height: 1.6;
}

/* CTA */
.cta-section {
  padding: var(--space-20) var(--space-8);
  border-top: 1px solid var(--color-border);
}
.cta-inner {
  max-width: var(--max-width);
  margin: 0 auto;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
}
.cta-inner h2 {
  font-size: clamp(1.5rem, 3vw, 2rem);
  letter-spacing: -0.03em;
}
.cta-inner p {
  font-size: 1rem;
  color: var(--text-secondary);
  margin-bottom: var(--space-2);
}

/* Footer */
.home-footer {
  border-top: 1px solid var(--color-border);
  padding: var(--space-10) var(--space-8);
}
.footer-inner {
  max-width: var(--max-width);
  margin: 0 auto;
}
.footer-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-8);
}
.footer-brand {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-muted);
  font-size: 0.875rem;
  font-weight: 500;
}
.footer-links {
  display: flex;
  gap: var(--space-6);
}
.footer-links a {
  font-size: 0.8125rem;
  color: var(--text-muted);
  text-decoration: none;
  transition: color var(--transition-fast);
}
.footer-links a:hover { color: var(--text-primary); }
.footer-bottom p {
  font-size: 0.75rem;
  color: var(--text-muted);
}

/* Responsive */
@media (max-width: 1024px) {
  .hero-grid {
    grid-template-columns: 1fr;
    gap: var(--space-10);
  }
  .hero-inner {
    max-width: 100%;
    text-align: left;
  }
  .hero-preview-wrapper {
    max-width: 560px;
    margin: 0 auto;
  }
}
@media (max-width: 900px) {
  .features-grid { grid-template-columns: repeat(2, 1fr); }
  .steps { grid-template-columns: repeat(2, 1fr); }
  .hero { padding: var(--space-16) var(--space-6) var(--space-12); }
  .hero-actions { flex-direction: column; align-items: flex-start; gap: var(--space-3); }
  .hero-join { width: 100%; }
  .hero-join input { flex: 1; width: auto; }
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
    background: rgba(10,10,10,0.96);
    backdrop-filter: blur(12px);
    border-bottom: 1px solid var(--color-border);
    padding: var(--space-2) var(--space-4);
  }
  .nav-links--open { display: flex; }
  .nav-link { padding: var(--space-3) var(--space-2); }
  .home-nav { padding: 0 var(--space-4); }
  .features-grid { grid-template-columns: 1fr; }
  .steps { grid-template-columns: 1fr; gap: var(--space-6); }
  .flash-banner { margin: var(--space-3) var(--space-4) 0; }
  .hero { padding: var(--space-12) var(--space-4) var(--space-10); }
  .features, .how, .cta-section { padding: var(--space-12) var(--space-4); }
  .home-footer { padding: var(--space-8) var(--space-4); }
  .footer-top { flex-direction: column; gap: var(--space-4); align-items: flex-start; }
  .preview-content {
    grid-template-columns: 1.2fr 1fr;
    height: 280px;
  }
  .mock-meta-title { font-size: 0.7rem; }
  .mock-status-pill { display: none; }
  .address-bar span { font-size: 0.65rem; }
}
</style>
