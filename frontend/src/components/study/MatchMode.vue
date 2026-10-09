<template>
  <div class="match-mode">
    <div class="match-counters">
      <span class="badge violet">Раунд {{ roundIndex + 1 }} / {{ rounds.length }}</span>
      <span class="badge green">Пар: {{ matchedTotal }} / {{ cards.length }}</span>
      <span class="badge red">Ошибок: {{ mistakes }}</span>
      <span class="badge"><Icon name="clock" :size="12" /> {{ timer }}</span>
    </div>
    <p class="muted small center">Перетащите слово к его переводу — или нажмите на слово, а затем на перевод.</p>

    <div class="match-board" :class="{ done: roundDone }">
      <div class="match-col">
        <div class="col-title"><span class="lang-chip">{{ leftLang.toUpperCase() }}</span></div>
        <div v-for="it in left" :key="it.uid" class="match-item" :data-match-id="it.uid" data-col="left"
             :class="stateClass(it)" @pointerdown="onDown($event, it)">
          {{ it.text }}
        </div>
      </div>
      <div class="match-col">
        <div class="col-title"><span class="lang-chip">{{ rightLang.toUpperCase() }}</span></div>
        <div v-for="it in right" :key="it.uid" class="match-item" :data-match-id="it.uid" data-col="right"
             :class="stateClass(it)" @pointerdown="onDown($event, it)">
          {{ it.text }}
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="drag.active" class="match-ghost" :style="{ left: drag.x + 'px', top: drag.y + 'px', width: drag.w + 'px' }">
        {{ drag.item.text }}
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { shuffle, normalizeAnswer, formatDuration } from '../../utils.js'

const props = defineProps({
  cards: { type: Array, required: true },
  side: { type: String, default: 'ru' }, // язык левой колонки
  pairs: { type: Number, default: 6 },
})
const emit = defineEmits(['finish'])

const started = Date.now()
const now = ref(Date.now())
const tick = setInterval(() => { now.value = Date.now() }, 1000)
const timer = computed(() => formatDuration(now.value - started))

const leftLang = props.side === 'en' ? 'en' : 'ru'
const rightLang = leftLang === 'ru' ? 'en' : 'ru'

// Последний раунд из одной пары присоединяется к предыдущему.
const deck = shuffle(props.cards)
const rounds = []
for (let i = 0; i < deck.length; i += props.pairs) rounds.push(deck.slice(i, i + props.pairs))
if (rounds.length > 1 && rounds[rounds.length - 1].length < 2) {
  rounds[rounds.length - 2].push(...rounds.pop())
}

const roundIndex = ref(0)
const left = ref([])
const right = ref([])
const matched = reactive(new Set())
const wrong = reactive(new Set())
const selected = ref(null)
const mistakes = ref(0)
const matchedTotal = ref(0)
const misses = new Map()
const roundDone = ref(false)
let uid = 0

const word = (card, lang) => (lang === 'ru' ? card.ru_word : card.en_word)
const setupRound = () => {
  const r = rounds[roundIndex.value]
  left.value = shuffle(r.map(card => ({ uid: ++uid, card, col: 'left', text: word(card, leftLang) })))
  right.value = shuffle(r.map(card => ({ uid: ++uid, card, col: 'right', text: word(card, rightLang) })))
  matched.clear()
  wrong.clear()
  selected.value = null
  roundDone.value = false
}
setupRound()

const summary = (completed) => ({
  total: props.cards.length,
  mistakes: mistakes.value,
  rounds: rounds.length,
  durationMs: Date.now() - started,
  completed,
  missed: props.cards.filter(c => misses.has(c.id)).map(c => ({ card: c, count: misses.get(c.id) })).sort((a, b) => b.count - a.count),
})
defineExpose({ summary: () => summary(false), progress: () => matchedTotal.value + mistakes.value })

const stateClass = (it) => ({
  matched: matched.has(it.uid),
  wrong: wrong.has(it.uid),
  selected: selected.value && selected.value.uid === it.uid,
  dragging: drag.active && drag.item && drag.item.uid === it.uid,
})

// Пара верна и для разных карточек с одинаковыми словами (дубли в папке).
const isPair = (a, b) => {
  const l = a.col === 'left' ? a : b
  const r = a.col === 'left' ? b : a
  if (l.card.id === r.card.id) return true
  return normalizeAnswer(word(l.card, rightLang)) === normalizeAnswer(r.text) &&
         normalizeAnswer(word(r.card, leftLang)) === normalizeAnswer(l.text)
}

const attempt = (a, b) => {
  selected.value = null
  if (!a || !b || a.col === b.col || matched.has(a.uid) || matched.has(b.uid)) return
  if (isPair(a, b)) {
    matched.add(a.uid)
    matched.add(b.uid)
    matchedTotal.value++
    if (navigator.vibrate) navigator.vibrate(10)
    if (matched.size === left.value.length * 2) {
      roundDone.value = true
      setTimeout(() => {
        if (roundIndex.value + 1 < rounds.length) { roundIndex.value++; setupRound() }
        else emit('finish', summary(true))
      }, 650)
    }
  } else {
    mistakes.value++
    const l = a.col === 'left' ? a : b
    misses.set(l.card.id, (misses.get(l.card.id) || 0) + 1)
    wrong.add(a.uid)
    wrong.add(b.uid)
    setTimeout(() => { wrong.delete(a.uid); wrong.delete(b.uid) }, 550)
  }
}

const drag = reactive({ active: false, item: null, x: 0, y: 0, w: 0, offX: 0, offY: 0 })
let pending = null

const onDown = (e, it) => {
  if (matched.has(it.uid) || (e.button !== undefined && e.button !== 0)) return
  e.preventDefault()
  const rect = e.currentTarget.getBoundingClientRect()
  pending = { it, sx: e.clientX, sy: e.clientY, rect }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp, { once: true })
  window.addEventListener('pointercancel', onCancel, { once: true })
}
const onMove = (e) => {
  if (!pending) return
  if (!drag.active) {
    if (Math.abs(e.clientX - pending.sx) < 6 && Math.abs(e.clientY - pending.sy) < 6) return
    drag.active = true
    drag.item = pending.it
    drag.w = pending.rect.width
    drag.offX = pending.sx - pending.rect.left
    drag.offY = pending.sy - pending.rect.top
    selected.value = null
  }
  drag.x = e.clientX - drag.offX
  drag.y = e.clientY - drag.offY
}
const targetAt = (x, y) => {
  const el = document.elementFromPoint(x, y)
  const node = el && el.closest('[data-match-id]')
  if (!node) return null
  const id = Number(node.dataset.matchId)
  return [...left.value, ...right.value].find(i => i.uid === id) || null
}
const cleanup = () => {
  window.removeEventListener('pointermove', onMove)
  window.removeEventListener('pointerup', onUp)
  window.removeEventListener('pointercancel', onCancel)
}
const onUp = (e) => {
  cleanup()
  if (!pending) return
  const it = pending.it
  pending = null
  if (drag.active) {
    drag.active = false
    const target = targetAt(e.clientX, e.clientY)
    if (target && target.col !== it.col) attempt(it, target)
    drag.item = null
    return
  }
  if (selected.value && selected.value.col !== it.col) attempt(selected.value, it)
  else selected.value = selected.value && selected.value.uid === it.uid ? null : it
}
const onCancel = () => { cleanup(); pending = null; drag.active = false; drag.item = null }

onMounted(() => document.body.classList.add('no-select'))
onBeforeUnmount(() => { clearInterval(tick); cleanup(); document.body.classList.remove('no-select') })
</script>

<style>
.match-mode { width: min(760px, 100%); display: flex; flex-direction: column; gap: 14px; }
.match-counters { display: flex; gap: 8px; justify-content: center; flex-wrap: wrap; }
.match-board { display: grid; grid-template-columns: 1fr 1fr; gap: clamp(12px, 4vw, 40px); margin-top: 6px; transition: opacity 0.3s; }
.match-board.done { opacity: 0.5; }
.match-col { display: flex; flex-direction: column; gap: 10px; }
.col-title { display: flex; justify-content: center; margin-bottom: 2px; }
.match-item {
  min-height: 58px; padding: 12px 14px; border-radius: 14px; display: flex; align-items: center; justify-content: center; text-align: center;
  font-weight: 700; font-size: 16px; line-height: 1.25; word-break: break-word;
  background: var(--surface-2); border: 1px solid var(--border-strong); color: var(--text);
  cursor: grab; touch-action: none; user-select: none; -webkit-user-select: none;
  transition: transform 0.15s, background 0.2s, border-color 0.2s, opacity 0.35s, box-shadow 0.2s;
}
.match-col:first-child .match-item { background: linear-gradient(145deg, rgba(139, 92, 246, 0.24), rgba(139, 92, 246, 0.08)); }
.match-col:last-child .match-item { background: linear-gradient(145deg, rgba(59, 130, 246, 0.24), rgba(59, 130, 246, 0.08)); }
.match-item:hover { transform: translateY(-2px); border-color: var(--violet-2); }
.match-item.selected { border-color: var(--violet-2); box-shadow: 0 0 0 3px rgba(167, 139, 250, 0.35); transform: scale(1.02); }
.match-item.dragging { opacity: 0.35; }
.match-item.wrong { border-color: var(--danger); background: var(--danger-bg) !important; animation: shake 0.45s both; }
.match-item.matched { border-color: rgba(52, 211, 153, 0.6); background: var(--success-bg) !important; color: #a7f3d0; opacity: 0.35; pointer-events: none; transform: scale(0.97); }
.match-ghost {
  position: fixed; z-index: 300; pointer-events: none; min-height: 58px; padding: 12px 14px; border-radius: 14px;
  display: flex; align-items: center; justify-content: center; text-align: center; font-weight: 700; font-size: 16px;
  background: var(--grad); color: #fff; box-shadow: 0 16px 40px rgba(3, 2, 20, 0.6); transform: rotate(-2deg);
}
.no-select { user-select: none; -webkit-user-select: none; }
</style>
