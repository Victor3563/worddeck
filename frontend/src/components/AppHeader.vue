<template>
  <header class="app-header">
    <div class="container header-inner">
      <RouterLink :to="auth.loggedIn ? '/app' : '/'" class="brand">
        <img src="/favicon.svg" alt="" width="30" height="30" />
        <span>Word<span class="gradient-text">Deck</span></span>
      </RouterLink>

      <nav v-if="auth.loggedIn" class="nav hide-sm">
        <RouterLink to="/app" class="nav-link" :class="{ active: isContent }"><Icon name="layers" :size="16" /> Мои области</RouterLink>
        <RouterLink to="/stats" class="nav-link"><Icon name="chart" :size="16" /> Статистика</RouterLink>
        <RouterLink to="/shares" class="nav-link"><Icon name="link" :size="16" /> Ссылки</RouterLink>
      </nav>

      <div class="spacer"></div>

      <template v-if="auth.loggedIn">
        <div class="user-menu" ref="menuRoot">
          <button class="user-btn" type="button" @click="menu = !menu" :aria-expanded="menu">
            <span class="avatar">{{ initials }}</span>
            <span class="hide-sm user-name">{{ auth.user.display_name }}</span>
            <Icon name="chevron-down" :size="16" class="hide-sm" />
          </button>
          <Transition name="pop">
            <div v-if="menu" class="dropdown" @click="menu = false">
              <div class="dropdown-head">
                <div class="strong">{{ auth.user.display_name }}</div>
                <div class="muted small">{{ auth.user.email }}</div>
              </div>
              <RouterLink to="/app" class="dropdown-item"><Icon name="layers" :size="16" /> Мои области</RouterLink>
              <RouterLink to="/stats" class="dropdown-item"><Icon name="chart" :size="16" /> Статистика</RouterLink>
              <RouterLink to="/shares" class="dropdown-item"><Icon name="link" :size="16" /> Мои ссылки</RouterLink>
              <RouterLink to="/profile" class="dropdown-item"><Icon name="user" :size="16" /> Профиль</RouterLink>
              <button type="button" class="dropdown-item danger" @click="logout"><Icon name="logout" :size="16" /> Выйти</button>
            </div>
          </Transition>
        </div>
      </template>
      <template v-else>
        <RouterLink :to="{ name: 'login', query: redirectQuery }" class="btn ghost">Войти</RouterLink>
        <RouterLink :to="{ name: 'register', query: redirectQuery }" class="btn primary">Регистрация</RouterLink>
      </template>
    </div>
  </header>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { auth } from '../stores/auth.js'
import { ui } from '../stores/ui.js'

const route = useRoute()
const router = useRouter()
const menu = ref(false)
const menuRoot = ref(null)

const initials = computed(() => {
  const n = (auth.user && auth.user.display_name) || '?'
  return n.split(/\s+/).filter(Boolean).slice(0, 2).map(p => p[0]).join('').toUpperCase()
})
const isContent = computed(() => ['dashboard', 'workspace', 'folder'].includes(route.name))
const redirectQuery = computed(() =>
  route.name && !['landing', 'login', 'register'].includes(route.name) ? { redirect: route.fullPath } : {})

const onDoc = (e) => { if (menuRoot.value && !menuRoot.value.contains(e.target)) menu.value = false }
onMounted(() => document.addEventListener('click', onDoc))
onBeforeUnmount(() => document.removeEventListener('click', onDoc))

const logout = async () => {
  await auth.logout()
  ui.toast('Вы вышли из аккаунта')
  router.push('/')
}
</script>

<style>
.app-header {
  position: sticky; top: 0; z-index: 50;
  height: var(--header-h);
  background: rgba(11, 10, 29, 0.72);
  backdrop-filter: blur(16px) saturate(140%);
  border-bottom: 1px solid var(--border);
}
.header-inner { height: 100%; display: flex; align-items: center; gap: 12px; }
.brand { display: flex; align-items: center; gap: 10px; font-weight: 800; font-size: 20px; letter-spacing: -0.03em; color: var(--text); margin-right: 18px; }
.brand:hover { color: var(--text); }
.nav { display: flex; gap: 4px; }
.nav-link { display: inline-flex; align-items: center; gap: 7px; height: 36px; padding: 0 12px; border-radius: 10px; color: var(--text-2); font-weight: 600; font-size: 14px; }
.nav-link:hover { background: var(--surface-2); color: var(--text); }
.nav-link.router-link-exact-active, .nav-link.active { background: rgba(139, 92, 246, 0.16); color: #ddd6fe; }
.user-menu { position: relative; }
.user-btn { display: flex; align-items: center; gap: 10px; height: 42px; padding: 0 10px 0 5px; border-radius: 999px; border: 1px solid var(--border); background: var(--surface); color: var(--text); cursor: pointer; font: inherit; font-weight: 600; }
.user-btn:hover { background: var(--surface-2); }
.user-name { max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.avatar { width: 32px; height: 32px; border-radius: 50%; display: grid; place-items: center; background: var(--grad); font-size: 13px; font-weight: 800; color: #fff; }
.dropdown { position: absolute; right: 0; top: calc(100% + 8px); min-width: 230px; padding: 6px; background: #1b1745; border: 1px solid var(--border-strong); border-radius: 16px; box-shadow: var(--shadow-lg); z-index: 60; }
.dropdown-head { padding: 10px 12px 12px; border-bottom: 1px solid var(--border); margin-bottom: 6px; }
.dropdown-head .strong { font-weight: 700; }
.dropdown-item { display: flex; width: 100%; align-items: center; gap: 10px; padding: 9px 12px; border-radius: 10px; color: var(--text-2); font: inherit; font-size: 14px; font-weight: 600; background: none; border: 0; cursor: pointer; text-align: left; }
.dropdown-item:hover { background: var(--surface-2); color: var(--text); }
.dropdown-item.danger:hover { background: var(--danger-bg); color: #fecdd3; }
.pop-enter-active, .pop-leave-active { transition: opacity 0.15s, transform 0.15s; }
.pop-enter-from, .pop-leave-to { opacity: 0; transform: translateY(-4px); }
@media (max-width: 640px) { .brand { font-size: 18px; margin-right: 0; } }
</style>
