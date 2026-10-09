<template>
  <div class="container page">
    <div class="page-head">
      <div>
        <h1>Привет, {{ firstName }} 👋</h1>
        <p class="sub">Ваши области — отдельные пространства для разных целей. Внутри — папки с карточками.</p>
      </div>
      <button class="btn primary" type="button" @click="openCreate"><Icon name="plus" /> Новая область</button>
    </div>

    <div v-if="stats" class="kpi-grid fade-up">
      <div class="kpi"><div class="kpi-value">{{ stats.cards }}</div><div class="kpi-label">карточек в {{ stats.folders }} {{ plural(stats.folders, 'папке', 'папках', 'папках') }}</div></div>
      <div class="kpi"><div class="kpi-value">{{ stats.sessions }}</div><div class="kpi-label">тренировок пройдено</div></div>
      <div class="kpi"><div class="kpi-value"><Icon name="flame" :size="22" class="flame" /> {{ stats.streak_days }}</div><div class="kpi-label">{{ plural(stats.streak_days, 'день', 'дня', 'дней') }} подряд</div></div>
      <div class="kpi"><div class="kpi-value">{{ stats.sessions ? Math.round(stats.accuracy * 100) + '%' : '—' }}</div><div class="kpi-label">точность ответов</div></div>
      <div class="kpi activity-kpi hide-sm">
        <div class="bars">
          <div v-for="d in stats.last_days" :key="d.day" class="bar" :title="`${d.day}: ${d.sessions} трен.`"
               :style="{ height: barHeight(d.cards) }" :class="{ empty: !d.cards }"></div>
        </div>
        <div class="kpi-label">активность за 14 дней</div>
      </div>
    </div>

    <div class="section-title">
      <h2><Icon name="layers" /> Мои области</h2>
    </div>

    <div v-if="loading" class="grid cols-auto">
      <div v-for="i in 3" :key="i" class="skeleton" style="height: 150px"></div>
    </div>
    <div v-else class="grid cols-auto">
      <RouterLink v-for="ws in workspaces" :key="ws.id" :to="`/workspaces/${ws.id}`" class="tile fade-up">
        <span class="accent-bar" :style="{ background: colorOf(ws) }"></span>
        <div class="ws-icon" :style="{ background: colorOf(ws) + '33', color: colorOf(ws) }"><Icon name="layers" /></div>
        <div class="tile-title">{{ ws.name }}</div>
        <div v-if="ws.description" class="tile-desc">{{ ws.description }}</div>
        <div class="tile-meta">
          <span class="badge"><Icon name="folder" :size="12" /> {{ ws.folder_count }}</span>
          <span class="badge"><Icon name="cards" :size="12" /> {{ ws.card_count }}</span>
        </div>
        <div class="tile-actions" @click.prevent.stop>
          <button class="btn ghost icon sm" type="button" title="Поделиться" @click="share(ws)"><Icon name="link" :size="15" /></button>
          <button class="btn ghost icon sm" type="button" title="Изменить" @click="openEdit(ws)"><Icon name="edit" :size="15" /></button>
          <button class="btn ghost icon sm" type="button" title="Удалить" @click="remove(ws)"><Icon name="trash" :size="15" /></button>
        </div>
      </RouterLink>
      <button class="tile add" type="button" @click="openCreate">
        <Icon name="plus" :size="26" />
        <span class="strong">Новая область</span>
        <span class="muted small">например, «IT-лексика» или «Путешествия»</span>
      </button>
    </div>

    <template v-if="recent.length">
      <div class="section-title">
        <h2><Icon name="clock" /> Недавние тренировки</h2>
        <RouterLink to="/stats" class="btn ghost sm">Вся история <Icon name="arrow-right" :size="14" /></RouterLink>
      </div>
      <div class="panel tight">
        <div class="list">
          <div v-for="s in recent" :key="s.id" class="list-item">
            <span class="mode-icon" :class="s.mode"><Icon :name="modeIcon(s.mode)" :size="16" /></span>
            <div class="grow">
              <div class="strong">{{ s.title || 'Тренировка' }}</div>
              <div class="muted small">{{ modeLabel(s.mode) }} · {{ cardsWord(s.total_cards) }} · {{ formatDuration(s.duration_ms) }}</div>
            </div>
            <span class="badge" :class="s.mistakes ? 'red' : 'green'">{{ s.mistakes ? 'ошибок: ' + s.mistakes : 'без ошибок' }}</span>
            <span class="muted small hide-sm nowrap">{{ formatDate(s.created_at) }}</span>
          </div>
        </div>
      </div>
    </template>

    <EntityDialog :open="dialog.open" kind="workspace" :initial="dialog.item" :action="saveWorkspace" @close="dialog.open = false" />
    <ShareDialog :open="shareState.open" resource-type="workspace" :resource-id="shareState.id" :name="shareState.name" @close="shareState.open = false" />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { api } from '../api.js'
import { auth } from '../stores/auth.js'
import { ui } from '../stores/ui.js'
import { WORKSPACE_COLORS, MODE_LABELS, cardsWord, formatDuration, formatDate, plural } from '../utils.js'
import EntityDialog from '../components/EntityDialog.vue'
import ShareDialog from '../components/ShareDialog.vue'

const workspaces = ref([])
const stats = ref(null)
const recent = ref([])
const loading = ref(true)
const dialog = reactive({ open: false, item: null })
const shareState = reactive({ open: false, id: 0, name: '' })

const load = async () => {
  try {
    const [ws, st, ss] = await Promise.all([api.workspaces(), api.stats(), api.sessions(5, 0)])
    workspaces.value = ws
    stats.value = st
    recent.value = ss.items
  } catch (e) {
    ui.error(e)
  } finally {
    loading.value = false
  }
}
onMounted(load)

const firstName = computed(() => (auth.user ? auth.user.display_name.split(' ')[0] : ''))
const colorOf = (ws) => WORKSPACE_COLORS[ws.color] || WORKSPACE_COLORS.violet
const maxCards = computed(() => Math.max(1, ...((stats.value && stats.value.last_days) || []).map(d => d.cards)))
const barHeight = (n) => (n ? Math.max(12, Math.round((n / maxCards.value) * 100)) : 6) + '%'

const openCreate = () => { dialog.item = null; dialog.open = true }
const openEdit = (ws) => { dialog.item = ws; dialog.open = true }
const saveWorkspace = async (payload) => {
  if (dialog.item) {
    const upd = await api.updateWorkspace(dialog.item.id, payload)
    workspaces.value = workspaces.value.map(w => (w.id === upd.id ? upd : w))
    ui.success('Область сохранена')
  } else {
    const ws = await api.createWorkspace(payload)
    workspaces.value.push(ws)
    ui.success('Область создана')
  }
}
const remove = async (ws) => {
  const ok = await ui.confirm({
    title: `Удалить область «${ws.name}»?`,
    message: `Будут удалены все папки (${ws.folder_count}) и карточки (${ws.card_count}) внутри. Это действие нельзя отменить.`,
    confirmText: 'Удалить', danger: true,
  })
  if (!ok) return
  try {
    await api.deleteWorkspace(ws.id)
    workspaces.value = workspaces.value.filter(w => w.id !== ws.id)
    ui.success('Область удалена')
    stats.value = await api.stats()
  } catch (e) { ui.error(e) }
}
const share = (ws) => { shareState.id = ws.id; shareState.name = ws.name; shareState.open = true }
const modeIcon = (m) => ({ learn: 'hand', dictation: 'keyboard', match: 'match' }[m])
const modeLabel = (m) => MODE_LABELS[m]
</script>

<style>
.ws-icon { width: 42px; height: 42px; border-radius: 14px; display: grid; place-items: center; }
.activity-kpi { display: flex; flex-direction: column; justify-content: space-between; gap: 8px; }
.bars { display: flex; align-items: flex-end; gap: 4px; height: 38px; }
.bar { flex: 1; border-radius: 4px; background: var(--grad); min-height: 3px; opacity: 0.9; }
.bar.empty { background: var(--surface-3); opacity: 1; }
</style>
