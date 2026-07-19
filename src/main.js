import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import './style.css'
import App from './App.vue'
import HomePage from './pages/HomePage.vue'
import LobbyPage from './pages/LobbyPage.vue'
import RoomPage from './pages/RoomPage.vue'
import DocumentationPage from './pages/DocumentationPage.vue'
import FAQPage from './pages/FAQPage.vue'
import ServerStatusPage from './pages/ServerStatusPage.vue'

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
app.mount('#app')
