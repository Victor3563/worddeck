import { createRouter, createWebHistory } from 'vue-router'
import { auth } from './stores/auth.js'
import { loadLimits } from './stores/limits.js'

import LandingView from './views/LandingView.vue'
import AuthView from './views/AuthView.vue'
import DashboardView from './views/DashboardView.vue'
import WorkspaceView from './views/WorkspaceView.vue'
import FolderView from './views/FolderView.vue'
import StudyView from './views/StudyView.vue'
import SharedView from './views/SharedView.vue'
import SharesView from './views/SharesView.vue'
import StatsView from './views/StatsView.vue'
import ProfileView from './views/ProfileView.vue'
import NotFoundView from './views/NotFoundView.vue'

const routes = [
  { path: '/', name: 'landing', component: LandingView, meta: { guestOnly: true, title: 'Карточки для английского' } },
  { path: '/login', name: 'login', component: AuthView, props: { mode: 'login' }, meta: { guestOnly: true, title: 'Вход' } },
  { path: '/register', name: 'register', component: AuthView, props: { mode: 'register' }, meta: { guestOnly: true, title: 'Регистрация' } },

  { path: '/app', name: 'dashboard', component: DashboardView, meta: { auth: true, title: 'Мои области' } },
  { path: '/workspaces/:id(\\d+)', name: 'workspace', component: WorkspaceView, props: true, meta: { auth: true, title: 'Область' } },
  { path: '/folders/:id(\\d+)', name: 'folder', component: FolderView, props: true, meta: { auth: true, title: 'Папка' } },

  // /study/learn?source=folder&id=1 | source=workspace&id=1 | source=share&token=abc[&folder=5]
  { path: '/study/:mode(learn|dictation|match)', name: 'study', component: StudyView, props: true, meta: { fullscreen: true, title: 'Тренировка' } },

  { path: '/s/:token', name: 'shared', component: SharedView, props: true, meta: { title: 'Общий доступ' } },
  { path: '/shares', name: 'shares', component: SharesView, meta: { auth: true, title: 'Мои ссылки' } },
  { path: '/stats', name: 'stats', component: StatsView, meta: { auth: true, title: 'Статистика' } },
  { path: '/profile', name: 'profile', component: ProfileView, meta: { auth: true, title: 'Профиль' } },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView, meta: { title: 'Страница не найдена' } },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, saved) {
    if (saved) return saved
    if (to.path !== from.path) return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  if (!auth.ready) await Promise.all([auth.init(), loadLimits()])
  if (to.meta.auth && !auth.loggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  // Тренировка по общей ссылке доступна без аккаунта.
  if (to.name === 'study' && to.query.source !== 'share' && !auth.loggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guestOnly && auth.loggedIn) {
    return typeof to.query.redirect === 'string' && to.query.redirect.startsWith('/') ? to.query.redirect : { name: 'dashboard' }
  }
})

router.afterEach((to) => {
  document.title = (to.meta.title ? to.meta.title + ' · ' : '') + 'WordDeck'
})
