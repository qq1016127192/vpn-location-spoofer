import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

const routes = [
  { path: '/', redirect: '/vpn' },
  { path: '/map', component: () => import('./views/MapView.vue') },
  { path: '/vpn', component: () => import('./views/VPNView.vue') },
  { path: '/vpn-download', component: () => import('./views/VPNDownloadView.vue') },
  { path: '/register', component: () => import('./views/RegisterView.vue') },
  { path: '/help', component: () => import('./views/HelpView.vue') }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

const app = createApp(App)
app.use(router)
app.mount('#app')
