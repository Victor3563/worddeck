<template>
  <div v-if="!auth.ready" class="page-loader boot"><Spinner :size="34" /></div>
  <template v-else>
    <AppHeader v-if="!hideChrome" />
    <main :class="{ 'main-full': hideChrome }">
      <RouterView v-slot="{ Component, route }">
        <Transition name="page" mode="out-in">
          <component :is="Component" :key="route.path" />
        </Transition>
      </RouterView>
    </main>
    <nav v-if="auth.loggedIn && !hideChrome" class="tabbar">
      <RouterLink to="/app"><Icon name="layers" /><span>Области</span></RouterLink>
      <RouterLink to="/stats"><Icon name="chart" /><span>Статистика</span></RouterLink>
      <RouterLink to="/shares"><Icon name="link" /><span>Ссылки</span></RouterLink>
      <RouterLink to="/profile"><Icon name="user" /><span>Профиль</span></RouterLink>
    </nav>
  </template>
  <UiHost />
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { auth } from './stores/auth.js'
import AppHeader from './components/AppHeader.vue'
import UiHost from './components/UiHost.vue'

const route = useRoute()
const hideChrome = computed(() => !!route.meta.fullscreen)
</script>

<style>
.boot { min-height: 100vh; }
.tabbar { display: none; }
@media (max-width: 640px) {
  main:not(.main-full) { padding-bottom: 70px; }
  .tabbar {
    position: fixed; left: 0; right: 0; bottom: 0; z-index: 50;
    display: grid; grid-template-columns: repeat(4, 1fr);
    padding: 6px 6px calc(6px + env(safe-area-inset-bottom));
    background: rgba(13, 11, 34, 0.92); backdrop-filter: blur(14px);
    border-top: 1px solid var(--border);
  }
  .tabbar a { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 6px 0; border-radius: 12px; color: var(--muted); font-size: 11px; font-weight: 600; }
  .tabbar a.router-link-active { color: #ddd6fe; background: rgba(139, 92, 246, 0.14); }
}
</style>
