<template>
  <div class="learn-mode">
    <div class="learn-counters">
      <span class="badge green"><Icon name="check" :size="12" /> Знаю: {{ knownCount }}</span>
      <span class="badge">Осталось: {{ queue.length }}</span>
      <span class="badge red"><Icon name="rotate" :size="12" /> Повторов: {{ mistakes }}</span>
    </div>

    <div class="deck">
      <div v-if="queue.length > 2" class="deck-under u2"></div>
      <div v-if="queue.length > 1" class="deck-under u1"></div>
      <div v-if="current" :key="current.key" class="swipe"
           :style="swipeStyle" :class="{ animating }"
           @pointerdown="onDown" @pointermove="onMove" @pointerup="onUp" @pointercancel="onCancel" @contextmenu.prevent>
        <div class="swipe-label left" :style="{ opacity: leftOpacity }"><Icon name="check" :size="20" /> Знаю</div>
        <div class="swipe-label right" :style="{ opacity: rightOpacity }">Ещё раз <Icon name="rotate" :size="18" /></div>
        <FlipCard :card="current.card" :side="current.side" size="study" :interactive="false"
                  :flipped="flipped" :show-example="showExample" />
      </div>
    </div>

    <div class="learn-controls">
      <button class="btn lg known" type="button" @click="swipe('left')" title="Знаю (←)">
        <Icon name="arrow-left" /> Знаю
      </button>
      <button class="btn icon lg" type="button" @click="flipped = !flipped" title="Перевернуть (пробел)"><Icon name="rotate" /></button>
      <button class="btn icon lg" type="button" @click="showExample = !showExample" :class="{ active: showExample }" title="Пример (↑ или E)"><Icon name="bulb" /></button>
      <button class="btn icon lg" type="button" :disabled="!history.length" @click="undo" title="Отменить (Z)"><Icon name="undo" /></button>
      <button class="btn lg unknown" type="button" @click="swipe('right')" title="Не знаю (→)">
        Не знаю <Icon name="arrow-right" />
      </button>
    </div>
    <p class="muted small center hide-sm keys-hint">
      <kbd>←</kbd> знаю · <kbd>→</kbd> ещё раз · <kbd>Пробел</kbd> перевернуть · <kbd>↑</kbd> пример · <kbd>Z</kbd> отменить
    </p>
    <p class="muted small center gesture-mobile">Свайп влево — знаю, вправо — ещё раз. Касание — перевернуть, удержание — пример.</p>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import FlipCard from '../FlipCard.vue'
import { shuffle, insertRandom } from '../../utils.js'
import { LONG_PRESS_MS } from '../../composables/usePress.js'

const THRESHOLD = 110

const props = defineProps({
  cards: { type: Array, required: true },
  side: { type: String, default: 'ru' }, // ru | en | mixed
})
const emit = defineEmits(['finish'])

const started = Date.now()
let keySeq = 0
const pickSide = () => (props.side === 'mixed' ? (Math.random() < 0.5 ? 'ru' : 'en') : props.side)
const queue = ref(shuffle(props.cards).map(card => ({ card, side: pickSide(), key: ++keySeq })))
const knownCount = ref(0)
const mistakes = ref(0)
const misses = new Map() // id карточки -> сколько раз смахнули вправо
const history = ref([])
const flipped = ref(false)
const showExample = ref(false)
const current = computed(() => queue.value[0] || null)

const dx = ref(0)
const dy = ref(0)
const animating = ref(false)
let startX = 0, startY = 0, dragging = false, pointerDown = false, longTimer = null, longFired = false

const swipeStyle = computed(() => ({
  transform: `translate(${dx.value}px, ${dy.value * 0.25}px) rotate(${dx.value / 18}deg)`,
}))
const leftOpacity = computed(() => Math.min(1, Math.max(0, -dx.value / THRESHOLD)))
const rightOpacity = computed(() => Math.min(1, Math.max(0, dx.value / THRESHOLD)))

const summary = (completed) => ({
  total: props.cards.length,
  mistakes: mistakes.value,
  rounds: 1 + Math.max(0, ...misses.values()),
  durationMs: Date.now() - started,
  completed,
  missed: props.cards.filter(c => misses.has(c.id)).map(c => ({ card: c, count: misses.get(c.id) }))
    .sort((a, b) => b.count - a.count),
  learned: knownCount.value,
})
defineExpose({ summary: () => summary(false), progress: () => knownCount.value + mistakes.value })

const commit = (dir) => {
  const item = queue.value[0]
  if (!item) return
  history.value.push({ queue: queue.value.slice(), known: knownCount.value, mistakes: mistakes.value, misses: new Map(misses) })
  if (history.value.length > 50) history.value.shift()
  const rest = queue.value.slice(1)
  if (dir === 'left') {
    knownCount.value++
  } else {
    mistakes.value++
    misses.set(item.card.id, (misses.get(item.card.id) || 0) + 1)
    insertRandom(rest, { ...item, side: props.side === 'mixed' ? pickSide() : item.side, key: ++keySeq })
  }
  queue.value = rest
  flipped.value = false
  showExample.value = false
  if (!rest.length) emit('finish', summary(true))
}

const swipe = (dir) => {
  if (animating.value || !current.value) return
  animating.value = true
  dx.value = dir === 'left' ? -window.innerWidth : window.innerWidth
  setTimeout(() => {
    commit(dir)
    animating.value = false
    dx.value = 0
    dy.value = 0
  }, 260)
}

const undo = () => {
  const h = history.value.pop()
  if (!h) return
  queue.value = h.queue
  knownCount.value = h.known
  mistakes.value = h.mistakes
  misses.clear()
  h.misses.forEach((v, k) => misses.set(k, v))
  flipped.value = false
  showExample.value = false
}

const clearLong = () => { if (longTimer) { clearTimeout(longTimer); longTimer = null } }
const onDown = (e) => {
  if (animating.value || (e.button !== undefined && e.button !== 0)) return
  pointerDown = true
  dragging = false
  longFired = false
  startX = e.clientX
  startY = e.clientY
  e.currentTarget.setPointerCapture && e.currentTarget.setPointerCapture(e.pointerId)
  clearLong()
  longTimer = setTimeout(() => {
    if (pointerDown && !dragging) {
      longFired = true
      if (navigator.vibrate) navigator.vibrate(15)
      showExample.value = !showExample.value
    }
  }, LONG_PRESS_MS)
}
const onMove = (e) => {
  if (!pointerDown) return
  const mx = e.clientX - startX
  const my = e.clientY - startY
  if (!dragging && (Math.abs(mx) > 10 || Math.abs(my) > 10)) { dragging = true; clearLong() }
  if (dragging) { dx.value = mx; dy.value = my }
}
const onUp = () => {
  if (!pointerDown) return
  pointerDown = false
  clearLong()
  if (dragging) {
    if (dx.value <= -THRESHOLD) swipe('left')
    else if (dx.value >= THRESHOLD) swipe('right')
    else { animating.value = true; dx.value = 0; dy.value = 0; setTimeout(() => { animating.value = false }, 220) }
  } else if (!longFired) {
    flipped.value = !flipped.value
  }
}
const onCancel = () => { pointerDown = false; clearLong(); dx.value = 0; dy.value = 0 }

const onKey = (e) => {
  if (e.target && ['INPUT', 'TEXTAREA'].includes(e.target.tagName)) return
  if (e.key === 'ArrowLeft') { e.preventDefault(); swipe('left') }
  else if (e.key === 'ArrowRight') { e.preventDefault(); swipe('right') }
  else if (e.key === ' ' || e.key === 'Enter') { e.preventDefault(); flipped.value = !flipped.value }
  else if (e.key === 'ArrowUp' || e.key.toLowerCase() === 'e' || e.key.toLowerCase() === 'у') { e.preventDefault(); showExample.value = !showExample.value }
  else if (e.key.toLowerCase() === 'z' || e.key.toLowerCase() === 'я' || e.key === 'Backspace') { e.preventDefault(); undo() }
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => { window.removeEventListener('keydown', onKey); clearLong() })
</script>

<style>
.learn-mode { display: flex; flex-direction: column; align-items: center; gap: 22px; width: 100%; }
.learn-counters { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.deck { position: relative; width: min(560px, 100%); height: min(380px, 52vh); }
.deck-under { position: absolute; inset: 0; border-radius: 20px; border: 1px solid var(--border); }
.deck-under.u1 { transform: translateY(12px) scale(0.96); background: rgba(99, 102, 241, 0.2); }
.deck-under.u2 { transform: translateY(24px) scale(0.92); background: rgba(59, 130, 246, 0.12); }
.swipe {
  position: absolute; inset: 0; z-index: 2;
  touch-action: none; user-select: none; -webkit-user-select: none; -webkit-touch-callout: none;
  cursor: grab; will-change: transform;
}
.swipe:active { cursor: grabbing; }
.swipe.animating { transition: transform 0.26s ease-out; }
.swipe .flip-card { cursor: inherit; }
.swipe-label {
  position: absolute; top: 22px; z-index: 5; display: flex; align-items: center; gap: 6px;
  padding: 8px 14px; border-radius: 12px; font-weight: 800; font-size: 16px; letter-spacing: 0.02em;
  pointer-events: none; opacity: 0; border: 2px solid;
}
.swipe-label.left { left: 22px; color: #6ee7b7; border-color: #34d399; background: rgba(6, 78, 59, 0.6); transform: rotate(-8deg); }
.swipe-label.right { right: 22px; color: #fda4af; border-color: #fb7185; background: rgba(136, 19, 55, 0.55); transform: rotate(8deg); }
.learn-controls { display: flex; gap: 10px; align-items: center; justify-content: center; flex-wrap: wrap; margin-top: 10px; }
.learn-controls .known { background: var(--success-bg); border-color: rgba(52, 211, 153, 0.4); color: #a7f3d0; }
.learn-controls .known:hover { background: rgba(52, 211, 153, 0.25); }
.learn-controls .unknown { background: var(--danger-bg); border-color: rgba(251, 113, 133, 0.4); color: #fecdd3; }
.learn-controls .unknown:hover { background: rgba(251, 113, 133, 0.25); }
.learn-controls .btn.active { border-color: var(--warning); color: var(--warning); }
.keys-hint kbd { margin: 0 2px; }
.gesture-mobile { display: none; }
@media (max-width: 640px) {
  .gesture-mobile { display: block; }
  .learn-controls .btn.lg { height: 46px; padding: 0 14px; }
  .learn-controls .btn.icon.lg { width: 46px; }
}
</style>
