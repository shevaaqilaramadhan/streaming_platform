import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import './style.css'
import App from './App.vue'
import HomePage from './pages/HomePage.vue'
import RoomPage from './pages/RoomPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomePage,
    },
    {
      path: '/room/:roomId',
      name: 'room',
      component: RoomPage,
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
