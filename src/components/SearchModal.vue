<template>
  <div v-if="isOpen" class="modal-backdrop" @click.self="close">
    <div class="search-modal glass" role="dialog" aria-modal="true" aria-labelledby="catalog-search-title">
      <!-- Header -->
      <div class="search-header">
        <div class="search-title-row">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="search-icon">
            <circle cx="11" cy="11" r="8"/>
            <line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          <h3 id="catalog-search-title">Search Movies & Anime</h3>
          <button class="close-btn" @click="close" aria-label="Close search">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>

        <!-- Search Bar -->
        <div class="search-input-wrap">
          <input
            ref="searchInputRef"
            v-model="query"
            type="text"
            class="input search-input"
            placeholder="Type a title e.g. Insidious, Frieren, Dandadan…"
            @keydown.enter="onSearch"
          />
          <button
            class="btn btn-primary btn-sm search-btn"
            :disabled="!query.trim() || isSearching"
            @click="onSearch"
          >
            <span v-if="isSearching" class="spinner-sm"></span>
            <span v-else>Search</span>
          </button>
        </div>

        <!-- Filter tabs -->
        <div class="provider-filters">
          <button
            v-for="prov in providers"
            :key="prov.id"
            class="filter-pill"
            :class="{ 'filter-pill--active': selectedProvider === prov.id }"
            @click="selectFilter(prov.id)"
          >
            {{ prov.label }}
          </button>
        </div>
      </div>

      <!-- Content Area -->
      <div class="search-body">
        <!-- Loading state -->
        <div v-if="isSearching" class="state-message">
          <div class="spinner"></div>
          <span>Searching IDLIX & Samehadaku…</span>
        </div>

        <!-- Error state -->
        <div v-else-if="errorMessage" class="state-message state-error">
          <p>{{ errorMessage }}</p>
        </div>

        <!-- Empty query prompt -->
        <div v-else-if="!hasSearched" class="state-message state-hint">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
            <line x1="8" y1="21" x2="16" y2="21"/>
            <line x1="12" y1="17" x2="12" y2="21"/>
          </svg>
          <p>Find movies from IDLIX and anime episodes from Samehadaku to watch together in this room.</p>
        </div>

        <!-- No results -->
        <div v-else-if="results.length === 0" class="state-message state-empty">
          <p>No results found for "{{ lastQuery }}".</p>
          <span class="hint-sub">Try searching with a different keyword.</span>
        </div>

        <!-- Results List / Grid -->
        <div v-else class="results-grid">
          <div
            v-for="item in results"
            :key="item.id"
            class="media-card"
            :class="{ 'is-selected': selectedItem?.id === item.id }"
            @click="selectItem(item)"
          >
            <div class="media-poster-wrap">
              <img
                v-if="item.poster"
                :src="item.poster"
                :alt="item.title"
                class="media-poster"
                loading="lazy"
                referrerpolicy="no-referrer"
              />
              <div v-else class="media-poster-placeholder">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <polygon points="5 3 19 12 5 21 5 3"/>
                </svg>
              </div>
              <span class="provider-badge" :class="'badge-' + item.provider.toLowerCase()">
                {{ item.provider }}
              </span>
              <span v-if="item.quality" class="quality-badge">{{ item.quality }}</span>
              <span v-else-if="item.latestEpisode" class="quality-badge">{{ item.latestEpisode }}</span>
            </div>

            <div class="media-info">
              <h4 class="media-title" :title="item.title">{{ item.title }}</h4>
              <div class="media-meta">
                <span v-if="item.year" class="media-year">{{ item.year }}</span>
                <span class="media-type">{{ item.type }}</span>
              </div>
              <p v-if="item.overview" class="media-overview">{{ item.overview }}</p>

              <div class="media-actions">
                <button
                  v-if="item.type === 'movie' || item.type === 'anime'"
                  class="btn btn-primary btn-xs"
                  @click.stop="playMedia(item.detailUrl)"
                >
                  Watch in Room
                </button>
                <button
                  v-if="item.type === 'series'"
                  class="btn btn-secondary btn-xs"
                  @click.stop="fetchEpisodes(item)"
                >
                  View Episodes
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Episodes Drawer / Modal overlay -->
        <Transition name="fade">
          <div v-if="showingEpisodes" class="episodes-drawer glass">
            <div class="drawer-header">
              <h4>{{ selectedSeries?.title }} — Episodes</h4>
              <button class="btn btn-secondary btn-xs" @click="showingEpisodes = false">Back to Results</button>
            </div>

            <div v-if="isLoadingEpisodes" class="state-message">
              <div class="spinner-sm"></div>
              <span>Loading episode list…</span>
            </div>

            <div v-else class="episodes-list">
              <div
                v-for="ep in episodesList"
                :key="ep.episodeNumber"
                class="episode-row"
                @click="playMedia(ep.pageUrl)"
              >
                <span class="ep-num">Ep {{ ep.episodeNumber }}</span>
                <span class="ep-name">{{ ep.title || 'Episode ' + ep.episodeNumber }}</span>
                <button class="btn btn-primary btn-xs">Play</button>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, watch } from 'vue'
import { API_BASE } from '../config.js'

const props = defineProps({
  isOpen: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['close', 'select-video'])

const searchInputRef = ref(null)
const query = ref('')
const lastQuery = ref('')
const isSearching = ref(false)
const hasSearched = ref(false)
const errorMessage = ref('')
const results = ref([])
const selectedProvider = ref('all')
const selectedItem = ref(null)

const showingEpisodes = ref(false)
const selectedSeries = ref(null)
const episodesList = ref([])
const isLoadingEpisodes = ref(false)

const providers = [
  { id: 'all', label: 'All Providers' },
  { id: 'idlix', label: 'Movies & Series (IDLIX)' },
  { id: 'samehadaku', label: 'Anime (Samehadaku)' }
]

watch(() => props.isOpen, (open) => {
  if (open) {
    nextTick(() => {
      searchInputRef.value?.focus()
    })
  } else {
    showingEpisodes.value = false
  }
})

function close() {
  emit('close')
}

function selectFilter(id) {
  selectedProvider.value = id
  if (hasSearched.value && query.value.trim()) {
    onSearch()
  }
}

async function onSearch() {
  const q = query.value.trim()
  if (!q) return

  isSearching.value = true
  errorMessage.value = ''
  showingEpisodes.value = false
  lastQuery.value = q

  try {
    const provParam = selectedProvider.value !== 'all' ? `&provider=${selectedProvider.value}` : ''
    const res = await fetch(`${API_BASE}/api/search?q=${encodeURIComponent(q)}${provParam}`)
    if (!res.ok) {
      throw new Error(`Search failed with status ${res.status}`)
    }
    const data = await res.json()
    results.value = data.results || []
    hasSearched.value = true
  } catch (err) {
    errorMessage.value = 'Failed to fetch search results. Please try again.'
  } finally {
    isSearching.value = false
  }
}

function selectItem(item) {
  selectedItem.value = item
}

function playMedia(url) {
  if (!url) return
  emit('select-video', url)
  close()
}

async function fetchEpisodes(item) {
  selectedSeries.value = item
  showingEpisodes.value = true
  isLoadingEpisodes.value = true
  episodesList.value = []

  try {
    const res = await fetch(`${API_BASE}/api/catalog/episodes?provider=idlix&slug=${encodeURIComponent(item.slug)}&season=1`)
    if (!res.ok) throw new Error('Failed to load episodes')
    const data = await res.json()
    episodesList.value = data.episodes || []
  } catch (err) {
    episodesList.value = []
  } finally {
    isLoadingEpisodes.value = false
  }
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1.5rem;
}

.search-modal {
  width: 100%;
  max-width: 760px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  background: var(--surface-card, #12151d);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: var(--radius-lg, 12px);
  overflow: hidden;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.5);
}

.search-header {
  padding: 1.25rem 1.5rem 1rem;
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.search-title-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.search-icon {
  color: var(--color-primary, #6366f1);
}

.search-title-row h3 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary, #f3f4f6);
  flex: 1;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-muted, #9ca3af);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.15s;
}

.close-btn:hover {
  color: var(--text-primary, #fff);
}

.search-input-wrap {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.search-input {
  flex: 1;
  background: var(--surface-elevated, #1a1e28);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  color: var(--text-primary, #fff);
  padding: 0.6rem 0.85rem;
  border-radius: var(--radius-md, 8px);
  font-size: 0.95rem;
}

.search-input:focus {
  outline: none;
  border-color: var(--color-primary, #6366f1);
}

.provider-filters {
  display: flex;
  gap: 0.5rem;
  overflow-x: auto;
}

.filter-pill {
  background: var(--surface-elevated, #1a1e28);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  color: var(--text-muted, #9ca3af);
  padding: 0.35rem 0.75rem;
  font-size: 0.8rem;
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.15s;
}

.filter-pill:hover {
  color: var(--text-primary, #fff);
  background: rgba(255, 255, 255, 0.08);
}

.filter-pill--active {
  background: var(--color-primary, #6366f1);
  color: #fff;
  border-color: var(--color-primary, #6366f1);
}

.search-body {
  position: relative;
  flex: 1;
  overflow-y: auto;
  padding: 1.25rem 1.5rem;
  min-height: 320px;
}

.state-message {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  min-height: 240px;
  color: var(--text-muted, #9ca3af);
  gap: 0.75rem;
}

.state-hint p {
  max-width: 380px;
  line-height: 1.5;
  font-size: 0.9rem;
}

.hint-sub {
  font-size: 0.8rem;
  color: var(--text-secondary, #6b7280);
}

.results-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1rem;
}

.media-card {
  display: flex;
  gap: 0.85rem;
  background: var(--surface-elevated, #171b24);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.06));
  border-radius: var(--radius-md, 8px);
  padding: 0.75rem;
  cursor: pointer;
  transition: transform 0.15s, border-color 0.15s;
}

.media-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.18);
}

.media-poster-wrap {
  position: relative;
  width: 76px;
  height: 108px;
  flex-shrink: 0;
  border-radius: 6px;
  overflow: hidden;
  background: #0f1218;
}

.media-poster {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.media-poster-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #4b5563;
}

.provider-badge {
  position: absolute;
  top: 4px;
  left: 4px;
  font-size: 0.65rem;
  font-weight: 600;
  padding: 2px 5px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.75);
}

.badge-idlix {
  color: #38bdf8;
}

.badge-samehadaku {
  color: #f43f5e;
}

.quality-badge {
  position: absolute;
  bottom: 4px;
  right: 4px;
  font-size: 0.65rem;
  padding: 1px 4px;
  border-radius: 3px;
  background: rgba(0, 0, 0, 0.85);
  color: #e5e7eb;
}

.media-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.media-title {
  margin: 0 0 0.25rem;
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary, #f9fafb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.media-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  color: var(--text-secondary, #9ca3af);
  margin-bottom: 0.35rem;
}

.media-type {
  text-transform: capitalize;
}

.media-overview {
  font-size: 0.75rem;
  color: var(--text-muted, #9ca3af);
  margin: 0 0 0.5rem;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.35;
}

.media-actions {
  margin-top: auto;
  display: flex;
  gap: 0.5rem;
}

.episodes-drawer {
  position: absolute;
  inset: 0;
  background: rgba(18, 21, 29, 0.97);
  display: flex;
  flex-direction: column;
  padding: 1.25rem;
  z-index: 10;
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.drawer-header h4 {
  margin: 0;
  font-size: 1rem;
  color: #fff;
}

.episodes-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  overflow-y: auto;
  flex: 1;
}

.episode-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.episode-row:hover {
  background: rgba(255, 255, 255, 0.08);
}

.ep-num {
  font-weight: 600;
  font-size: 0.85rem;
  color: var(--color-primary, #6366f1);
  min-width: 44px;
}

.ep-name {
  flex: 1;
  font-size: 0.85rem;
  color: #e5e7eb;
}

.btn-xs {
  padding: 0.25rem 0.6rem;
  font-size: 0.75rem;
}

.spinner-sm {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  display: inline-block;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
