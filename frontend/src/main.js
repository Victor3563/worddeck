import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router.js'
import { auth } from './stores/auth.js'
import { ui } from './stores/ui.js'
import Icon from './components/Icon.vue'
import Modal from './components/Modal.vue'
import Spinner from './components/Spinner.vue'
import EmptyState from './components/EmptyState.vue'

auth.onExpire = () => {
  ui.toast('Сессия истекла — войдите снова')
  const current = router.currentRoute.value
  if (current.name !== 'login') router.push({ name: 'login', query: { redirect: current.fullPath } })
}

const app = createApp(App)
app.component('Icon', Icon)
app.component('Modal', Modal)
app.component('Spinner', Spinner)
app.component('EmptyState', EmptyState)
app.use(router)
app.mount('#app')
