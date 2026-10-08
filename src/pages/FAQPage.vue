<template>
  <main class="faq-page">
    <!-- Nav -->
    <nav class="home-nav">
      <router-link to="/" class="nav-logo">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
          <rect x="2" y="2" width="24" height="24" rx="6" fill="#fff" fill-opacity="0.08" stroke="#fff" stroke-width="1.2" stroke-opacity="0.2"/>
          <polygon points="12,9 21,14 12,19" fill="#fff" fill-opacity="0.7"/>
        </svg>
        <span class="nav-brand">LitcqnWatch</span>
      </router-link>
      <nav class="nav-links" :class="{ 'nav-links--open': mobileMenuOpen }" aria-label="Main">
        <router-link to="/docs" class="nav-link" @click="mobileMenuOpen = false">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z"/>
            <path d="M14 2v5a1 1 0 0 0 1 1h5"/>
            <path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/>
          </svg>
          Docs
        </router-link>
        <router-link to="/faq" class="nav-link active" @click="mobileMenuOpen = false">
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
        <a href="https://github.com/shevaaqilaramadhan/streaming_platform" target="_blank" rel="noopener noreferrer" class="nav-link" @click="mobileMenuOpen = false">
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

    <!-- Hero -->
    <header class="faq-hero">
      <div class="faq-hero-inner">
        <span class="faq-hero-badge">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/>
          </svg>
          FAQ
        </span>
        <h1 class="faq-hero-title">
          Frequently Asked <span class="gradient-text">Questions</span>
        </h1>
        <p class="faq-hero-subtitle">
          Find answers to the most common questions about LitcqnWatch.
          Can't find what you're looking for? Reach out on GitHub.
        </p>
      </div>
    </header>

    <!-- Search -->
    <div class="faq-search-wrapper">
      <div class="faq-search glass">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search questions..."
          class="search-input"
        />
      </div>
    </div>

    <!-- FAQ Content -->
    <div class="faq-content">
      <!-- Category Tabs -->
      <div class="faq-tabs">
        <button
          class="faq-tab"
          :class="{ active: activeCategory === 'all' }"
          @click="activeCategory = 'all'"
        >
          <span class="tab-icon">📋</span>
          All
        </button>
        <button
          v-for="cat in categories"
          :key="cat.id"
          class="faq-tab"
          :class="{ active: activeCategory === cat.id }"
          @click="activeCategory = cat.id"
        >
          <span class="tab-icon">{{ cat.icon }}</span>
          {{ cat.label }}
        </button>
      </div>

      <!-- FAQ List -->
      <div class="faq-list">
        <template v-for="cat in filteredCategories" :key="cat.id">
          <div class="faq-category-header">
            <span class="cat-icon">{{ cat.icon }}</span>
            <h2>{{ cat.label }}</h2>
          </div>

          <div
            v-for="(item, idx) in filteredItems(cat.items)"
            :key="idx"
            class="faq-item glass"
            :class="{ open: openItems[cat.id + '-' + idx] }"
          >
            <button
              class="faq-question"
              @click="toggle(cat.id + '-' + idx)"
            >
              <span class="q-text">{{ item.q }}</span>
              <span class="q-toggle">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="6 9 12 15 18 9"/>
                </svg>
              </span>
            </button>
            <Transition name="accordion">
              <div v-if="openItems[cat.id + '-' + idx]" class="faq-answer">
                <p v-html="item.a"></p>
              </div>
            </Transition>
          </div>
        </template>

        <!-- Empty state -->
        <div v-if="hasNoResults" class="faq-empty glass">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
            <line x1="8" y1="11" x2="14" y2="11"/>
          </svg>
          <p>No questions match your search.</p>
          <button class="btn btn-ghost" @click="searchQuery = ''">Clear search</button>
        </div>
      </div>

      <!-- Still have questions -->
      <div class="faq-cta glass">
        <div class="cta-content">
          <h3>Still have questions?</h3>
          <p>Can't find the answer you're looking for? Open an issue on our GitHub repository.</p>
        </div>
        <a href="https://github.com/shevaaqilaramadhan/streaming_platform/issues" target="_blank" rel="noopener noreferrer" class="btn btn-primary">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4"/>
            <path d="M9 18c-4.51 2-5-2-7-2"/>
          </svg>
          Open GitHub Issue
        </a>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref, computed } from 'vue'
import ThemeToggle from '../components/ThemeToggle.vue'

const searchQuery = ref('')
const activeCategory = ref('all')
const openItems = ref({})
const mobileMenuOpen = ref(false)

const categories = [
  {
    id: 'general',
    label: 'General',
    icon: '💡',
    items: [
      {
        q: 'What is LitcqnWatch?',
        a: 'LitcqnWatch is a fast, lightweight platform that lets you watch anime, YouTube, and direct video links together with friends in perfect synchronization — no account or app installation required.',
      },
      {
        q: 'Do I need to create an account?',
        a: 'No! LitcqnWatch is completely account-free. Simply enter a room name and your nickname, or click a shared room link to join immediately.',
      },
      {
        q: 'Is LitcqnWatch free to use?',
        a: 'Yes, LitcqnWatch is 100% free and open-source. The full repository is available on GitHub.',
      },
      {
        q: 'What browsers are supported?',
        a: 'LitcqnWatch works on all modern browsers including Chrome, Firefox, Safari, Edge, Brave, and mobile web browsers.',
      },
    ],
  },
  {
    id: 'rooms',
    label: 'Rooms & Security',
    icon: '🏠',
    items: [
      {
        q: 'How do I create a room?',
        a: 'Click <strong>"Create Watch Room"</strong> on the home page. A unique room URL is created instantly for you to share with friends.',
      },
      {
        q: 'Can I protect my room with a PIN code?',
        a: 'Yes! Hosts can set a 4–6 digit security PIN in the Room Settings modal. Any guest who wants to join must enter the correct PIN.',
      },
      {
        q: 'How many people can join a room?',
        a: 'There is no hard limit on participants. Rooms of 10 to 30 people run smoothly with instant chat delivery and playback synchronization.',
      },
      {
        q: 'Are rooms public or private?',
        a: 'Rooms are private by default (accessible only via link or PIN). Hosts can toggle rooms to Public in Room Settings to list them in the Public Lobby.',
      },
    ],
  },
  {
    id: 'playback',
    label: 'Playback & Shortcuts',
    icon: '🎬',
    items: [
      {
        q: 'What video formats and sites are supported?',
        a: 'LitcqnWatch supports direct <strong>MP4</strong>, <strong>HLS (.m3u8)</strong>, <strong>YouTube</strong>, as well as supported anime episode links with auto-stream resolution.',
      },
      {
        q: 'Does LitcqnWatch support Picture-in-Picture (PiP) and Chat Overlay?',
        a: 'Yes! Press <kbd>P</kbd> or click the PiP button to pop out native Picture-in-Picture video. In fullscreen mode, press <kbd>C</kbd> or click the Overlay button to toggle transparent floating chat.',
      },
      {
        q: 'How does Next Episode work?',
        a: 'When watching episodic content, LitcqnWatch automatically detects the next episode link and displays a convenient "Next Episode" button when the video approaches its end.',
      },
      {
        q: 'Who controls the video playback?',
        a: 'Only the room <strong>host</strong> controls playback (play, pause, seek, and video selection) to avoid desync conflicts. Hosts can transfer host privileges to any participant.',
      },
    ],
  },
  {
    id: 'technical',
    label: 'Technical',
    icon: '⚙️',
    items: [
      {
        q: 'What technology does LitcqnWatch use?',
        a: 'LitcqnWatch is built with <strong>Vue 3</strong> and <strong>Vite</strong> on the frontend, and <strong>Go</strong> with high-performance WebSockets and stream proxying on the backend.',
      },
      {
        q: 'Is my data collected or stored?',
        a: 'No. LitcqnWatch collects zero personal data. All room states and chats exist strictly in server memory during the watch session.',
      },
      {
        q: 'Can I self-host LitcqnWatch?',
        a: 'Yes! LitcqnWatch is open-source. You can clone the repo from GitHub and host the backend in Go and frontend on Vercel or any static host.',
      },
    ],
  },
  {
    id: 'troubleshooting',
    label: 'Troubleshooting',
    icon: '🔧',
    items: [
      {
        q: 'The video won\'t play. What should I do?',
        a: 'Make sure the video URL is publicly accessible. Direct MP4 links must not require authentication. For HLS streams, ensure the .m3u8 playlist is reachable. Try opening the URL directly in your browser to verify.',
      },
      {
        q: 'Chat messages aren\'t sending.',
        a: 'Check your internet connection and ensure WebSockets aren\'t blocked by your network (some corporate firewalls block WebSocket connections). Try refreshing the page or using a different network.',
      },
      {
        q: 'The video is out of sync for some users.',
        a: 'Minor sync differences (under 1 second) can occur due to network latency. If the desync is significant, the host can seek to a position to re-sync everyone. Ensure all participants have stable internet connections.',
      },
      {
        q: 'I get a "Room not found" error.',
        a: 'The room may have been closed due to inactivity, or the room code may be incorrect. Double-check the URL and try creating a new room if the issue persists.',
      },
    ],
  },
]

const filteredCategories = computed(() => {
  if (activeCategory.value === 'all') return categories
  return categories.filter(c => c.id === activeCategory.value)
})

function filteredItems(items) {
  if (!searchQuery.value.trim()) return items
  const q = searchQuery.value.toLowerCase()
  return items.filter(
    item => item.q.toLowerCase().includes(q) || item.a.toLowerCase().includes(q)
  )
}

const hasNoResults = computed(() => {
  return filteredCategories.value.every(cat => filteredItems(cat.items).length === 0)
})

function toggle(key) {
  openItems.value[key] = !openItems.value[key]
}
</script>

<style scoped>
.faq-page {
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
  background: rgba(10,10,10,0.85);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--color-border);
}
.nav-logo {
  display: flex; align-items: center; gap: var(--space-2);
  text-decoration: none;
}
.nav-brand {
  color: var(--text-primary);
  font-size: 0.9375rem; font-weight: 600; letter-spacing: -0.02em;
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
  border: 1px solid var(--color-border);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.hamburger-btn:hover {
  color: var(--text-primary);
  background: var(--color-fill-hover);
}
.nav-link {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 6px 14px; border-radius: var(--radius-sm);
  font-size: 0.8125rem; font-weight: 500;
  color: var(--text-secondary); text-decoration: none;
  transition: all var(--transition-fast);
}
.nav-link:hover {
  color: var(--text-primary);
  background: var(--color-fill-hover);
}
.nav-link.active {
  color: var(--text-primary);
  background: var(--color-fill-hover);
}
.nav-link svg { opacity: 0.7; }
.nav-link:hover svg, .nav-link.active svg { opacity: 1; }
.nav-right { display: flex; align-items: center; gap: var(--space-6); }

/* Hero */
.faq-hero {
  padding: var(--space-12) var(--space-6) var(--space-8);
  text-align: center;
  max-width: 700px;
  margin: 0 auto;
}
.faq-hero-badge {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 4px 14px; border-radius: var(--radius-full);
  font-size: 0.8rem; font-weight: 600;
  background: var(--color-fill);
  border: 1px solid var(--color-border);
  color: var(--text-secondary);
  margin-bottom: var(--space-4);
}
.faq-hero-title {
  font-size: clamp(2rem, 4vw, 3rem);
  font-weight: 800; letter-spacing: -0.04em;
  line-height: 1.15; margin-bottom: var(--space-4);
}
.faq-hero-subtitle {
  font-size: 1.1rem; color: var(--text-secondary);
  line-height: 1.7; max-width: 550px; margin: 0 auto;
}

/* Search */
.faq-search-wrapper {
  max-width: 560px;
  margin: 0 auto var(--space-8);
  padding: 0 var(--space-6);
}
.faq-search {
  display: flex; align-items: center; gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-lg);
  transition: all var(--transition-base);
}
.faq-search:focus-within {
  border-color: var(--color-border-strong);
  box-shadow: 0 0 0 3px rgba(255,255,255,0.04);
}
.faq-search svg { color: var(--text-muted); flex-shrink: 0; }
.search-input {
  flex: 1; background: none; border: none; outline: none;
  font-family: var(--font-sans); font-size: 0.95rem;
  color: var(--text-primary);
}
.search-input::placeholder { color: var(--text-muted); }

/* Content */
.faq-content {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 var(--space-6) var(--space-16);
}

/* Tabs */
.faq-tabs {
  display: flex; gap: var(--space-2);
  margin-bottom: var(--space-8);
  overflow-x: auto;
  padding-bottom: var(--space-2);
  -webkit-overflow-scrolling: touch;
}
.faq-tab {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 16px; border-radius: var(--radius-full);
  font-family: var(--font-sans); font-size: 0.85rem;
  font-weight: 600; color: var(--text-secondary);
  background: var(--color-fill);
  border: 1px solid var(--color-border);
  cursor: pointer; white-space: nowrap;
  transition: all var(--transition-base);
}
.faq-tab:hover {
  background: var(--color-fill-hover);
  color: var(--text-primary);
}
.faq-tab.active {
  background: var(--color-fill-hover);
  border-color: var(--color-border-strong);
  color: var(--text-primary);
}
.tab-icon { font-size: 1rem; }

/* Category Header */
.faq-category-header {
  display: flex; align-items: center; gap: var(--space-3);
  margin-bottom: var(--space-4);
  margin-top: var(--space-2);
}
.cat-icon { font-size: 1.3rem; }
.faq-category-header h2 {
  font-size: 1.2rem; font-weight: 800;
  letter-spacing: -0.02em; margin: 0;
}

/* FAQ Items */
.faq-list {
  display: flex; flex-direction: column;
  gap: var(--space-3);
  margin-bottom: var(--space-10);
}

.faq-item {
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: all var(--transition-base);
}
.faq-item:hover {
  border-color: rgba(255, 255, 255, 0.1);
}
.faq-item.open {
  border-color: var(--color-border-strong);
  background: var(--color-fill);
}

.faq-question {
  width: 100%;
  display: flex; align-items: center; justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  background: none; border: none; cursor: pointer;
  font-family: var(--font-sans); font-size: 0.95rem;
  font-weight: 600; color: var(--text-primary);
  text-align: left;
  transition: color var(--transition-fast);
}
.faq-question:hover { color: var(--color-primary); }

.q-text { flex: 1; line-height: 1.5; }
.q-toggle {
  flex-shrink: 0; color: var(--text-muted);
  transition: transform var(--transition-base), color var(--transition-fast);
}
.faq-item.open .q-toggle {
  transform: rotate(180deg);
  color: var(--color-primary);
}

.faq-answer {
  padding: 0 var(--space-5) var(--space-5);
}
.faq-answer p {
  font-size: 0.9rem; color: var(--text-secondary);
  line-height: 1.7; margin: 0;
}
.faq-answer :deep(a) {
  color: var(--color-primary);
  text-decoration: underline;
  text-underline-offset: 2px;
}

/* Accordion Transition */
.accordion-enter-active, .accordion-leave-active {
  transition: all 0.25s ease;
  overflow: hidden;
}
.accordion-enter-from, .accordion-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}
.accordion-enter-to, .accordion-leave-from {
  opacity: 1;
  max-height: 300px;
}

/* Empty */
.faq-empty {
  display: flex; flex-direction: column;
  align-items: center; gap: var(--space-3);
  padding: var(--space-10) var(--space-4);
  text-align: center; border-radius: var(--radius-lg);
  color: var(--text-muted);
}

/* CTA */
.faq-cta {
  display: flex; align-items: center; justify-content: space-between;
  padding: var(--space-6) var(--space-8);
  border-radius: var(--radius-lg);
  gap: var(--space-6);
}
.cta-content h3 {
  font-size: 1.1rem; font-weight: 700; margin: 0 0 var(--space-1);
}
.cta-content p {
  font-size: 0.9rem; color: var(--text-secondary); margin: 0;
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
    background: rgba(10,10,10,0.96);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border-bottom: 1px solid var(--color-border);
    padding: var(--space-2) var(--space-4);
    z-index: var(--z-overlay);
  }
  .nav-links--open { display: flex; }
  .nav-link { padding: var(--space-3) var(--space-2); }
  .home-nav { padding: var(--space-3) var(--space-4); position: relative; }
  .faq-hero { padding: var(--space-8) var(--space-4) var(--space-6); }
  .faq-cta {
    flex-direction: column; text-align: center;
    padding: var(--space-5);
  }
  .faq-question { padding: var(--space-3) var(--space-4); }
  .faq-answer { padding: 0 var(--space-4) var(--space-4); }
}
</style>
