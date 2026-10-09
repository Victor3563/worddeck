<template>
  <div class="study-page">
    <header class="study-bar">
      <button class="btn ghost icon" type="button" title="Выйти" @click="exit"><Icon name="x" /></button>
      <div class="study-title">
        <div class="muted small">{{ modeLabel }}</div>
        <div class="strong ellipsis">{{ title || 'Тренировка' }}</div>
      </div>
      <div class="spacer"></div>
      <span v-if="phase === 'play'" class="badge">{{ cards.length }} {{ plural(cards.length, 'карточка', 'карточки', 'карточек') }}</span>
    </header>

    <div class="study-stage">
      <div v-if="phase === 'loading'" class="page-loader"><Spinner :size="32" /></div>

      <EmptyState v-else-if="phase === 'error'" icon="alert" :title="errorText">
        <RouterLink class="btn primary" :to="backTo">Вернуться</RouterLink>
      </EmptyState>

      <form v-else-if="phase === 'setup'" class="setup panel fade-up" @submit.prevent="start">
        <div class="setup-icon" :class="mode"><Icon :name="modeIcon" :size="28" /></div>
        <h1>{{ modeLabel }}</h1>
        <p class="muted">{{ modeDescription }}</p>

        <div v-if="!allCards.length" class="form-error">В этом наборе нет карточек.</div>
        <div v-else-if="mode === 'match' && allCards.length < 2" class="form-error">Для сопоставления нужно минимум 2 карточки.</div>
        <template v-else>
          <div v-if="folderOptions.length > 1" class="field">
            <span class="label">Папка</span>
            <select v-model="folderFilter" class="select">
              <option value="">Все папки ({{ allSourceCards.length }})</option>
              <option v-for="f in folderOptions" :key="f.id" :value="String(f.id)">{{ f.name }} ({{ f.cards.length }})</option>
            </select>
          </div>

          <div class="field">
            <span class="label">{{ mode === 'dictation' ? 'Показывать слово на' : mode === 'match' ? 'Левая колонка' : 'Лицевая сторона' }}</span>
            <div class="segmented">
              <button type="button" :class="{ active: side === 'ru' }" @click="side = 'ru'">Русском</button>
              <button type="button" :class="{ active: side === 'en' }" @click="side = 'en'">Английском</button>
              <button v-if="mode !== 'match'" type="button" :class="{ active: side === 'mixed' }" @click="side = 'mixed'">Вперемешку</button>
            </div>
          </div>

          <div v-if="mode === 'match'" class="field">
            <span class="label">Пар в раунде</span>
            <div class="segmented">
              <button v-for="n in [4, 6, 8]" :key="n" type="button" :class="{ active: pairs === n }" @click="pairs = n">{{ n }}</button>
            </div>
          </div>

          <div v-if="allCards.length > 10" class="field">
            <span class="label">Сколько карточек</span>
            <div class="segmented">
              <button v-for="n in limitOptions" :key="n" type="button" :class="{ active: limit === n }" @click="limit = n">
                {{ n === 0 ? `Все (${allCards.length})` : n }}
              </button>
            </div>
          </div>

          <button class="btn primary lg block" type="submit" autofocus><Icon name="play" :size="16" /> Начать</button>
        </template>
      </form>

      <template v-else-if="phase === 'play'">
        <LearnMode v-if="mode === 'learn'" ref="modeRef" :key="'learn-' + runKey" :cards="cards" :side="side" @finish="finish" />
        <DictationMode v-else-if="mode === 'dictation'" ref="modeRef" :key="'dictation-' + runKey" :cards="cards" :side="side" @finish="finish" />
        <MatchMode v-else ref="modeRef" :key="'match-' + runKey" :cards="cards" :side="side" :pairs="pairs" @finish="finish" />
      </template>

      <StudyResult v-else-if="phase === 'result'" :result="result" :mode="mode" :back-to="backTo" :saved="saved" @restart="restart" />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api.js'
import { auth } from '../stores/auth.js'
import { ui } from '../stores/ui.js'
import { shuffle, plural, MODE_LABELS } from '../utils.js'
import LearnMode from '../components/study/LearnMode.vue'
import DictationMode from '../components/study/DictationMode.vue'
import MatchMode from '../components/study/MatchMode.vue'
import StudyResult from '../components/study/StudyResult.vue'
import { limits } from '../stores/limits.js'

const DESCRIPTIONS = {
  learn: 'Карточки появляются по одной в случайном порядке. Знаете перевод — смахните влево, не знаете — вправо: такие карточки вернутся позже. Тренировка закончится, когда все окажутся слева.',
  dictation: 'Напишите перевод слова без ошибок. Слова, в которых вы ошиблись, придётся написать заново в следующем раунде.',
  match: 'Перед вами слова и переводы в случайном порядке. Перетащите каждое слово к его переводу.',
}
const PREFS_KEY = 'worddeck.study'

const props = defineProps({ mode: { type: String, required: true } })

const route = useRoute()
const router = useRouter()
const prefs = (() => { try { return JSON.parse(localStorage.getItem(PREFS_KEY) || '{}') } catch { return {} } })()

const phase = ref('loading')
const errorText = ref('')
const title = ref('')
const allSourceCards = ref([])
const folderOptions = ref([])
const folderFilter = ref(route.query.folder ? String(route.query.folder) : '')
const side = ref(prefs.side || 'ru')
const pairs = ref(prefs.pairs || 6)
const limit = ref(0)
const cards = ref([])
const result = ref(null)
const saved = ref('')
const modeRef = ref(null)
const runKey = ref(0)
let meta = { folder_id: null, workspace_id: null }

const source = route.query.source
const sourceId = route.query.id
const token = route.query.token

const backTo = computed(() => {
  if (source === 'folder') return `/folders/${sourceId}`
  if (source === 'workspace') return `/workspaces/${sourceId}`
  if (source === 'share') return `/s/${token}`
  return '/app'
})

const allCards = computed(() => {
  if (!folderFilter.value) return allSourceCards.value
  const f = folderOptions.value.find(x => String(x.id) === folderFilter.value)
  return f ? f.cards : allSourceCards.value
})
const limitOptions = computed(() => [10, 20, 50].filter(n => n < allCards.value.length).concat([0]))

const load = async () => {
  try {
    if (source === 'folder') {
      const [f, c] = await Promise.all([api.folder(sourceId), api.cards(sourceId)])
      title.value = f.name
      allSourceCards.value = c
      meta = { folder_id: f.id, workspace_id: f.workspace_id }
    } else if (source === 'workspace') {
      const [w, c, folders] = await Promise.all([api.workspace(sourceId), api.workspaceCards(sourceId), api.folders(sourceId)])
      title.value = w.name
      allSourceCards.value = c
      folderOptions.value = folders.map(f => ({ id: f.id, name: f.name, cards: c.filter(x => x.folder_id === f.id) })).filter(f => f.cards.length)
      meta = { folder_id: null, workspace_id: w.id }
    } else if (source === 'share' && token) {
      const s = await api.publicShare(token)
      if (s.workspace) {
        title.value = s.workspace.name
        folderOptions.value = s.workspace.folders.filter(f => f.cards.length)
        allSourceCards.value = s.workspace.folders.flatMap(f => f.cards)
      } else if (s.folder) {
        title.value = s.folder.name
        allSourceCards.value = s.folder.cards
      } else if (s.card) {
        title.value = `${s.card.ru_word} — ${s.card.en_word}`
        allSourceCards.value = [s.card]
      }
    } else {
      throw Object.assign(new Error('Не указан набор карточек'), { status: 400 })
    }
    document.title = `${MODE_LABELS[props.mode]}: ${title.value} · WordDeck`
    phase.value = 'setup'
  } catch (e) {
    errorText.value = e.status === 404 ? 'Набор карточек не найден' : e.message
    phase.value = 'error'
  }
}
load()

watch([side, pairs], () => {
  try { localStorage.setItem(PREFS_KEY, JSON.stringify({ side: side.value, pairs: pairs.value })) } catch { /* localStorage недоступен, например в приватном режиме */ }
})

const start = (subset) => {
  let deck = Array.isArray(subset) ? subset : allCards.value
  if (!Array.isArray(subset) && limit.value) deck = shuffle(deck).slice(0, limit.value)
  if (props.mode === 'match' && side.value === 'mixed') side.value = 'ru'
  cards.value = deck
  result.value = null
  saved.value = ''
  runKey.value++
  phase.value = 'play'
}

const save = async (res) => {
  if (!auth.loggedIn) { saved.value = 'guest'; return }
  if (res.total === 0) return
  try {
    await api.saveSession({
      mode: props.mode,
      folder_id: meta.folder_id || (folderFilter.value ? Number(folderFilter.value) : null),
      workspace_id: meta.workspace_id,
      title: title.value.slice(0, limits.session_title),
      total_cards: res.total,
      mistakes: res.mistakes,
      rounds: res.rounds,
      duration_ms: res.durationMs,
      completed: res.completed,
    })
    saved.value = 'ok'
  } catch (e) {
    saved.value = 'error'
    ui.error('Не удалось сохранить результат: ' + e.message)
  }
}

const finish = (res) => {
  result.value = res
  phase.value = 'result'
  save(res)
}

const exit = async () => {
  if (phase.value === 'play' && modeRef.value) {
    const progress = modeRef.value.progress()
    if (progress > 0) {
      const ok = await ui.confirm({
        title: 'Завершить тренировку?',
        message: 'Прогресс этой тренировки будет сохранён в статистике как незавершённый.',
        confirmText: 'Выйти',
      })
      if (!ok) return
      const res = modeRef.value.summary()
      save(res)
    }
  }
  router.push(backTo.value)
}

const restart = (subset) => start(subset || cards.value)

const modeLabel = computed(() => MODE_LABELS[props.mode])
const modeIcon = computed(() => ({ learn: 'hand', dictation: 'keyboard', match: 'match' }[props.mode]))
const modeDescription = computed(() => DESCRIPTIONS[props.mode])
</script>

<style>
.study-page { min-height: 100vh; display: flex; flex-direction: column; }
.study-bar { position: sticky; top: 0; z-index: 20; display: flex; align-items: center; gap: 12px; padding: 12px 18px; background: rgba(11, 10, 29, 0.7); backdrop-filter: blur(14px); border-bottom: 1px solid var(--border); }
.study-title { min-width: 0; }
.ellipsis { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 60vw; }
.study-stage { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 28px 16px 40px; }
.setup { width: min(520px, 100%); display: flex; flex-direction: column; gap: 16px; padding: 30px; }
.setup h1 { font-size: 28px; }
.setup-icon { width: 58px; height: 58px; border-radius: 18px; display: grid; place-items: center; }
.setup-icon.learn { background: rgba(139, 92, 246, 0.2); color: #c4b5fd; }
.setup-icon.dictation { background: rgba(59, 130, 246, 0.2); color: #93c5fd; }
.setup-icon.match { background: rgba(34, 211, 238, 0.16); color: #67e8f9; }
.setup .segmented { flex-wrap: wrap; }
@media (max-width: 640px) { .study-stage { justify-content: flex-start; padding-top: 18px; } .setup { padding: 22px; } }
</style>
