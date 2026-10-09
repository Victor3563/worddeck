<template>
  <div class="container page">
    <div class="page-head">
      <div>
        <h1>Статистика</h1>
        <p class="sub">Сколько вы тренировались и как часто ошибались.</p>
      </div>
      <button v-if="total" class="btn danger sm" type="button" @click="clearAll"><Icon name="trash" :size="14" /> Очистить историю</button>
    </div>

    <div v-if="!stats" class="page-loader"><Spinner :size="30" /></div>
    <template v-else>
      <div class="kpi-grid">
        <div class="kpi"><div class="kpi-value">{{ stats.sessions }}</div><div class="kpi-label">тренировок</div></div>
        <div class="kpi"><div class="kpi-value">{{ stats.cards_studied }}</div><div class="kpi-label">карточек пройдено</div></div>
        <div class="kpi"><div class="kpi-value">{{ stats.minutes_studied }}</div><div class="kpi-label">{{ plural(stats.minutes_studied, 'минута', 'минуты', 'минут') }} обучения</div></div>
        <div class="kpi"><div class="kpi-value">{{ stats.sessions ? Math.round(stats.accuracy * 100) + '%' : '—' }}</div><div class="kpi-label">точность</div></div>
        <div class="kpi"><div class="kpi-value"><Icon name="flame" :size="22" class="flame" /> {{ stats.streak_days }}</div><div class="kpi-label">{{ plural(stats.streak_days, 'день', 'дня', 'дней') }} подряд</div></div>
      </div>

      <div class="stats-row">
        <div class="panel chart-panel">
          <h3>Активность за 2 недели</h3>
          <div class="chart">
            <div v-for="d in stats.last_days" :key="d.day" class="chart-col" :title="`${d.day}: ${d.sessions} трен., ${d.cards} карточек`">
              <span class="chart-val">{{ d.cards || '' }}</span>
              <div class="chart-bar" :class="{ empty: !d.cards }" :style="{ height: barHeight(d.cards) }"></div>
              <span class="chart-day">{{ dayLabel(d.day) }}</span>
            </div>
          </div>
          <p class="muted small">Высота столбца — число карточек, пройденных за день.</p>
        </div>
        <div class="panel modes-panel">
          <h3>По режимам</h3>
          <div v-for="m in modes" :key="m.key" class="mode-row">
            <span class="mode-icon" :class="m.key"><Icon :name="m.icon" :size="16" /></span>
            <span class="grow">{{ m.label }}</span>
            <b>{{ stats.by_mode[m.key] || 0 }}</b>
          </div>
        </div>
      </div>

      <div class="section-title"><h2><Icon name="clock" /> История</h2></div>
      <EmptyState v-if="!sessions.length" icon="play" title="Тренировок пока не было" text="Откройте любую папку и выберите режим: «Учить», «Диктант» или «Сопоставление».">
        <RouterLink to="/app" class="btn primary">К моим областям</RouterLink>
      </EmptyState>
      <div v-else class="panel tight table-wrap">
        <table class="table">
          <thead><tr><th>Набор</th><th>Режим</th><th>Карточек</th><th>Ошибок</th><th class="hide-sm">Раундов</th><th>Время</th><th class="hide-sm">Когда</th><th></th></tr></thead>
          <tbody>
            <tr v-for="s in sessions" :key="s.id">
              <td>
                <RouterLink v-if="s.folder_id" :to="`/folders/${s.folder_id}`" class="strong">{{ s.title || 'Папка' }}</RouterLink>
                <RouterLink v-else-if="s.workspace_id" :to="`/workspaces/${s.workspace_id}`" class="strong">{{ s.title || 'Область' }}</RouterLink>
                <span v-else class="strong">{{ s.title || '—' }}</span>
                <span v-if="!s.completed" class="badge">прервано</span>
              </td>
              <td><span class="nowrap">{{ modeLabel(s.mode) }}</span></td>
              <td>{{ s.total_cards }}</td>
              <td><span :class="s.mistakes ? 'bad' : 'good'">{{ s.mistakes }}</span></td>
              <td class="hide-sm">{{ s.rounds }}</td>
              <td class="nowrap">{{ formatDuration(s.duration_ms) }}</td>
              <td class="hide-sm nowrap muted">{{ formatDate(s.created_at) }}</td>
              <td><button class="btn ghost icon sm" type="button" title="Удалить запись" @click="removeSession(s)"><Icon name="x" :size="14" /></button></td>
            </tr>
          </tbody>
        </table>
        <div v-if="total > sessions.length" class="center load-more">
          <button class="btn" type="button" :disabled="busy" @click="loadMore"><Spinner v-if="busy" :size="16" /> Показать ещё</button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { api } from '../api.js'
import { ui } from '../stores/ui.js'
import { MODE_LABELS, formatDate, formatDuration, plural } from '../utils.js'

const PAGE = 20

const stats = ref(null)
const sessions = ref([])
const total = ref(0)
const busy = ref(false)

const load = async () => {
  try {
    const [st, ss] = await Promise.all([api.stats(), api.sessions(PAGE, 0)])
    stats.value = st
    sessions.value = ss.items
    total.value = ss.total
  } catch (e) { ui.error(e) }
}
onMounted(load)

const loadMore = async () => {
  busy.value = true
  try {
    const ss = await api.sessions(PAGE, sessions.value.length)
    sessions.value.push(...ss.items)
    total.value = ss.total
  } catch (e) { ui.error(e) } finally { busy.value = false }
}
const removeSession = async (s) => {
  try {
    await api.deleteSession(s.id)
    sessions.value = sessions.value.filter(x => x.id !== s.id)
    total.value--
    stats.value = await api.stats()
  } catch (e) { ui.error(e) }
}
const clearAll = async () => {
  const ok = await ui.confirm({ title: 'Очистить историю?', message: 'Все записи о тренировках будут удалены. Карточки останутся.', confirmText: 'Очистить', danger: true })
  if (!ok) return
  try {
    await api.clearSessions()
    await load()
    ui.success('История очищена')
  } catch (e) { ui.error(e) }
}

const maxCards = computed(() => Math.max(1, ...((stats.value && stats.value.last_days) || []).map(d => d.cards)))
const barHeight = (n) => (n ? Math.max(6, Math.round((n / maxCards.value) * 100)) : 3) + '%'
const dayLabel = (d) => new Date(d + 'T12:00:00').toLocaleDateString('ru-RU', { weekday: 'short' }).replace('.', '')
const modes = [
  { key: 'learn', label: 'Учить', icon: 'hand' },
  { key: 'dictation', label: 'Диктант', icon: 'keyboard' },
  { key: 'match', label: 'Сопоставление', icon: 'match' },
]
const modeLabel = (m) => MODE_LABELS[m]
</script>

<style>
.stats-row { display: grid; grid-template-columns: 2fr 1fr; gap: 16px; margin-top: 16px; }
.chart-panel, .modes-panel { display: flex; flex-direction: column; gap: 14px; }
.chart { display: flex; align-items: flex-end; gap: 6px; height: 180px; }
.chart-col { flex: 1; height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: flex-end; gap: 4px; min-width: 0; }
.chart-bar { width: 100%; max-width: 34px; border-radius: 8px 8px 4px 4px; background: var(--grad); }
.chart-bar.empty { background: var(--surface-3); }
.chart-val { font-size: 11px; color: var(--text-2); min-height: 14px; }
.chart-day { font-size: 11px; color: var(--muted); }
.mode-row { display: flex; align-items: center; gap: 12px; padding: 8px 0; border-bottom: 1px solid var(--border); }
.mode-row:last-child { border-bottom: 0; }
.bad { color: #fda4af; font-weight: 700; }
.good { color: #6ee7b7; font-weight: 700; }
.load-more { padding: 12px; }
@media (max-width: 860px) { .stats-row { grid-template-columns: 1fr; } }
</style>
