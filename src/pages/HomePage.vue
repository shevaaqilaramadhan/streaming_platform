<template>
  <main class="home-page">
    <!-- Animated background orbs -->
    <div class="orbs" aria-hidden="true">
      <div class="orb orb-1"></div>
      <div class="orb orb-2"></div>
      <div class="orb orb-3"></div>
    </div>

    <!-- Navigation bar -->
    <nav class="home-nav glass">
      <div class="nav-logo">
        <svg width="32" height="32" viewBox="0 0 28 28" fill="none">
          <circle cx="14" cy="14" r="14" fill="url(#nav-logo-grad)"/>
          <polygon points="11,9 21,14 11,19" fill="white"/>
          <defs>
            <linearGradient id="nav-logo-grad" x1="0" y1="0" x2="28" y2="28">
              <stop offset="0%" stop-color="hsl(260,80%,62%)"/>
              <stop offset="100%" stop-color="hsl(220,85%,60%)"/>
            </linearGradient>
          </defs>
        </svg>
        <span class="gradient-text nav-brand">WatchParty</span>
      </div>
      <div class="nav-links">
        <a href="#features" class="btn btn-ghost btn-sm">Features</a>
        <a href="#how-it-works" class="btn btn-ghost btn-sm">How it works</a>
      </div>
    </nav>

    <!-- Hero -->
    <section class="hero animate-fade-in" aria-labelledby="hero-heading">
      <div class="hero-badge">
        <span class="badge-dot"></span>
        Live Sync • No account needed
      </div>

      <h1 id="hero-heading" class="hero-title">
        Watch videos<br/>
        <span class="gradient-text">together, in sync.</span>
      </h1>

      <p class="hero-subtitle">
        Create a room, share the link, and enjoy perfectly synchronized video
        playback with friends — anywhere in the world.
      </p>

      <!-- CTA buttons -->
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

        <div class="join-form" @submit.prevent="joinRoom">
          <input
            id="join-room-input"
            v-model="joinRoomId"
            type="text"
            class="input hero-input"
            placeholder="Room code (e.g. xyz123)"
            maxlength="20"
            spellcheck="false"
            @keydown.enter="joinRoom"
          />
          <button
            id="join-room-submit-btn"
            type="button"
            class="btn btn-ghost"
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

    <!-- How it works -->
    <section id="how-it-works" class="steps-section" aria-labelledby="steps-heading">
      <h2 id="steps-heading" class="section-title">Up and running in seconds</h2>
      <div class="steps-list">
        <div v-for="(step, i) in steps" :key="step.title" class="step-item">
          <div class="step-number gradient-text">0{{ i + 1 }}</div>
          <div class="step-content">
            <h3>{{ step.title }}</h3>
            <p>{{ step.desc }}</p>
          </div>
          <div v-if="i < steps.length - 1" class="step-connector" aria-hidden="true"></div>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="home-footer">
      <p>WatchParty — Built with 🎬 and WebSockets</p>
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
    router.push({ name: 'room', params: { roomId: data.roomId }, query: { host: 'true' } })
  } catch {
    // Fallback: generate a local room ID so the UI is still explorable
    const roomId = generateLocalId()
    router.push({ name: 'room', params: { roomId }, query: { host: 'true' } })
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

const steps = [
  {
    title: 'Create a room',
    desc: 'Click "Create Watch Room" and you become the host with full playback control.',
  },
  {
    title: 'Share the link',
    desc: 'Copy the room URL and send it to your friends. They click it and pick a nickname.',
  },
  {
    title: 'Paste a video & watch',
    desc: 'Drop in any .mp4 link. Hit play and everyone\'s video starts in perfect sync.',
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
  background: radial-gradient(circle, hsl(260,80%,50%) 0%, transparent 70%);
  top: -200px; left: -200px;
  animation: orb-move-1 18s ease-in-out infinite;
}
.orb-2 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, hsl(220,85%,55%) 0%, transparent 70%);
  bottom: -100px; right: -150px;
  animation: orb-move-2 22s ease-in-out infinite;
}
.orb-3 {
  width: 350px; height: 350px;
  background: radial-gradient(circle, hsl(300,70%,50%) 0%, transparent 70%);
  top: 40%; left: 55%;
  animation: orb-move-1 28s ease-in-out infinite reverse;
  opacity: 0.2;
}

/* Nav */
.home-nav {
  display: flex; align-items: center; justify-content: space-between;
  padding: var(--space-4) var(--space-8);
  position: sticky; top: 0; z-index: var(--z-overlay);
  border-radius: 0; border-top: 0; border-left: 0; border-right: 0;
}
.nav-logo { display: flex; align-items: center; gap: var(--space-2); }
.nav-brand { font-size: 1.2rem; font-weight: 800; letter-spacing: -0.03em; }
.nav-links { display: flex; gap: var(--space-2); }

/* Hero */
.hero {
  position: relative; z-index: 1;
  display: flex; flex-direction: column; align-items: center;
  text-align: center;
  padding: var(--space-16) var(--space-6) var(--space-16);
  max-width: 720px; margin: 0 auto;
  gap: var(--space-6);
}

.hero-badge {
  display: inline-flex; align-items: center; gap: 8px;
  padding: 6px 16px;
  border-radius: var(--radius-full);
  background: var(--grad-brand-subtle);
  border: 1px solid hsla(260,80%,62%,0.3);
  font-size: 0.8125rem; font-weight: 600;
  color: hsl(260,80%,80%);
  letter-spacing: 0.04em;
}
.badge-dot {
  width: 7px; height: 7px; border-radius: 50%;
  background: hsl(260,80%,62%);
  animation: pulse-glow 2s infinite;
}

.hero-title {
  font-size: clamp(2.5rem, 6vw, 4.5rem);
  line-height: 1.1;
  letter-spacing: -0.04em;
}

.hero-subtitle {
  font-size: clamp(1rem, 2vw, 1.2rem);
  color: var(--text-secondary);
  max-width: 520px;
  line-height: 1.7;
}

.hero-actions {
  display: flex; flex-direction: column; align-items: center;
  gap: var(--space-4); width: 100%; max-width: 440px;
}

.hero-cta { min-width: 200px; }

.join-form {
  display: flex; gap: var(--space-2); width: 100%;
}
.hero-input { flex: 1; }

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
  text-align: center;
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

/* Steps */
.steps-section {
  position: relative; z-index: 1;
  padding: var(--space-16) var(--space-6);
  max-width: 800px; margin: 0 auto;
}

.steps-list {
  display: flex; flex-direction: column; gap: var(--space-4);
  position: relative;
}

.step-item {
  display: flex; align-items: flex-start; gap: var(--space-6);
  padding: var(--space-6);
  background: var(--color-glass);
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-lg);
  position: relative;
  transition: all var(--transition-base);
}
.step-item:hover { background: var(--color-glass-hover); }

.step-number {
  font-size: 2.5rem; font-weight: 900; letter-spacing: -0.04em;
  line-height: 1; flex-shrink: 0;
  min-width: 60px;
}
.step-content h3 { font-size: 1.1rem; margin-bottom: var(--space-1); }
.step-content p { font-size: 0.9rem; color: var(--text-secondary); }

/* Footer */
.home-footer {
  position: relative; z-index: 1;
  text-align: center;
  padding: var(--space-8);
  color: var(--text-muted);
  font-size: 0.875rem;
  border-top: 1px solid var(--color-glass-border);
}

@media (max-width: 640px) {
  .nav-links { display: none; }
  .home-nav { padding: var(--space-3) var(--space-4); }
  .hero { padding: var(--space-10) var(--space-4) var(--space-10); }
  .step-number { font-size: 1.8rem; min-width: 48px; }
}
</style>
