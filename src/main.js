import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import './style.css'
import App from './App.vue'
// Eager: first paint / primary UX
import HomePage from './pages/HomePage.vue'
import RoomPage from './pages/RoomPage.vue'
// Lazy: secondary pages (code-split)
const LobbyPage = () => import('./pages/LobbyPage.vue')
const DocumentationPage = () => import('./pages/DocumentationPage.vue')
const FAQPage = () => import('./pages/FAQPage.vue')
const ServerStatusPage = () => import('./pages/ServerStatusPage.vue')

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior() {
    return { top: 0 }
  },
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomePage,
    },
    {
      path: '/lobby',
      name: 'lobby',
      component: LobbyPage,
    },
    {
      path: '/room/:roomId',
      name: 'room',
      component: RoomPage,
    },
    {
      path: '/docs',
      name: 'docs',
      component: DocumentationPage,
    },
    {
      path: '/faq',
      name: 'faq',
      component: FAQPage,
    },
    {
      path: '/status',
      name: 'status',
      component: ServerStatusPage,
    },
    {
      // 404 — redirect home
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})

const app = createApp(App)
app.use(router)

// FE-M01: Global Vue error handler + unhandled promise rejections
app.config.errorHandler = (err, instance, info) => {
  console.error('[Vue Error]', err, info)
}
if (typeof window !== 'undefined') {
  window.addEventListener('unhandledrejection', (e) => {
    console.error('[Unhandled Promise]', e.reason)
  })
}

app.mount('#app')
