import { reactive } from 'vue'
import { api, setUnauthenticatedHandler } from '../api.js'

export const auth = reactive({
  user: null,
  ready: false,
  onExpire: null, // задаётся в main.js

  get loggedIn() { return !!this.user },

  async init() {
    try {
      this.user = await api.me()
    } catch {
      this.user = null
    }
    this.ready = true
  },

  async login(email, password) {
    this.user = (await api.login({ email, password })).user
  },

  async register(email, display_name, password) {
    this.user = (await api.register({ email, display_name, password })).user
  },

  async logout() {
    try { await api.logout() } finally { this.clear() }
  },

  async logoutEverywhere() {
    await api.logoutAll()
    this.clear()
  },

  // Вызывается, когда сессия на сервере уже завершена.
  clear() { this.user = null },

  // Вызывается при 401: сессия истекла, отозвана или аккаунт удалён.
  expire() {
    if (!this.user) return
    this.clear()
    if (typeof this.onExpire === 'function') this.onExpire()
  },
})

setUnauthenticatedHandler(() => auth.expire())
