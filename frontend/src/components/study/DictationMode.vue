<template>
  <div class="dictation-mode">
    <template v-if="phase === 'interlude'">
      <div class="interlude panel fade-up">
        <div class="inter-icon"><Icon name="rotate" :size="28" /></div>
        <h2>Раунд {{ round }} завершён</h2>
        <p class="muted">Ошибок в раунде: <b class="text">{{ nextRound.length }}</b>. Сейчас нужно написать эти слова заново — до тех пор, пока не будет ни одной ошибки.</p>
        <div class="inter-words">
          <span v-for="it in nextRound" :key="it.card.id" class="badge red">{{ promptOf(it) }}</span>
        </div>
        <button class="btn primary lg" type="button" ref="continueBtn" @click="startNextRound">Продолжить <Icon name="arrow-right" /></button>
      </div>
    </template>

    <template v-else-if="item">
      <div class="dict-counters">
        <span class="badge violet">Раунд {{ round }}</span>
        <span class="badge">{{ index + 1 }} / {{ roundItems.length }}</span>
        <span class="badge red">Ошибок: {{ totalMistakes }}</span>
      </div>

      <div class="prompt-card" :class="[`lang-${item.side}`, { shake: shakeIt }]" v-bind="press.handlers" tabindex="0">
        <span class="lang-chip">{{ item.side === 'ru' ? 'RU' : 'EN' }}</span>
        <div class="prompt-word">{{ promptOf(item) }}</div>
        <Transition name="example">
          <div v-if="showContext && contextOf(item)" class="prompt-example">{{ contextOf(item) }}</div>
        </Transition>
        <div v-if="showContext && !contextOf(item)" class="prompt-example muted">Пример не добавлен</div>
        <div class="face-hint">удерживайте — {{ showContext ? 'скрыть' : 'показать' }} пример</div>
      </div>

      <form class="answer" @submit.prevent="onEnter">
        <label class="muted small" for="dict-input">
          Напишите {{ item.side === 'ru' ? 'по-английски' : 'по-русски' }}
          <span v-if="hintShown" class="hint-pattern">{{ hintPattern }}</span>
        </label>
        <input id="dict-input" ref="inputEl" v-model="answer" class="input answer-input" :class="status"
               :readonly="status !== 'input'" autocomplete="off" autocapitalize="off" autocorrect="off" spellcheck="false"
               :placeholder="item.side === 'ru' ? 'Your answer…' : 'Ваш ответ…'" />

        <div v-if="status === 'wrong'" class="verdict wrong fade-up">
          <div class="row"><Icon name="x-circle" /> <b>Неверно</b></div>
          <div v-if="answer.trim()" class="diff">
            <span class="muted small">Ваш ответ:</span>
            <span class="diff-line">
              <span v-for="(d, i) in diff" :key="i" :class="`d-${d.type}`">{{ d.ch }}</span>
            </span>
          </div>
          <div class="correct">Правильно: <b>{{ targetOf(item) }}</b></div>
          <div v-if="targetExample(item)" class="muted small ex">{{ targetExample(item) }}</div>
        </div>
        <div v-if="status === 'right'" class="verdict right fade-up">
          <div class="row"><Icon name="check-circle" /> <b>Верно!</b> <span class="muted">{{ targetOf(item) }}</span></div>
        </div>

        <div class="row wrap answer-actions">
          <template v-if="status === 'input'">
            <button class="btn" type="button" @click="showContext = !showContext"><Icon name="bulb" :size="16" /> Контекст</button>
            <button class="btn" type="button" :disabled="hintShown" @click="hintShown = true"><Icon name="sparkles" :size="16" /> Подсказка</button>
            <button class="btn ghost" type="button" @click="giveUp">Не знаю</button>
            <div class="spacer"></div>
            <button class="btn primary" type="submit" :disabled="!answer.trim()">Проверить <kbd>Enter</kbd></button>
          </template>
          <template v-else-if="status === 'wrong'">
            <div class="spacer"></div>
            <button class="btn primary" type="submit" ref="nextBtn">Дальше <kbd>Enter</kbd></button>
          </template>
        </div>
      </form>
    </template>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { shuffle, isCorrectAnswer, diffChars } from '../../utils.js'
import { usePress } from '../../composables/usePress.js'

const props = defineProps({
  cards: { type: Array, required: true },
  side: { type: String, default: 'ru' }, // показываемая сторона; другую нужно ввести
})
const emit = defineEmits(['finish'])

const started = Date.now()
const pickSide = () => (props.side === 'mixed' ? (Math.random() < 0.5 ? 'ru' : 'en') : props.side)
const makeItems = (cards) => shuffle(cards).map(card => ({ card, side: pickSide() }))

const round = ref(1)
const roundItems = ref(makeItems(props.cards))
const index = ref(0)
const nextRound = ref([])
const phase = ref('play') // play | interlude
const status = ref('input') // input | wrong | right
const answer = ref('')
const totalMistakes = ref(0)
const misses = new Map()
const showContext = ref(false)
const hintShown = ref(false)
const shakeIt = ref(false)
const inputEl = ref(null)
const nextBtn = ref(null)
const continueBtn = ref(null)
let advanceTimer = null

const item = computed(() => roundItems.value[index.value] || null)
const promptOf = (it) => (it.side === 'ru' ? it.card.ru_word : it.card.en_word)
const targetOf = (it) => (it.side === 'ru' ? it.card.en_word : it.card.ru_word)
const contextOf = (it) => (it.side === 'ru' ? it.card.ru_example : it.card.en_example)
const targetExample = (it) => (it.side === 'ru' ? it.card.en_example : it.card.ru_example)
// Сравнение с ближайшим допустимым вариантом ("house, home" → два варианта).
const diff = computed(() => {
  if (!item.value) return []
  const typed = answer.value.trim()
  const variants = targetOf(item.value).split(/[,;/]/).map(s => s.trim()).filter(Boolean)
  let best = []
  let bestScore = -Infinity
  for (const v of variants) {
    const d = diffChars(typed, v)
    const score = d.filter(x => x.type === 'ok').length - 0.5 * d.filter(x => x.type !== 'ok').length
    if (score > bestScore) { best = d; bestScore = score }
  }
  return best
})
const hintPattern = computed(() => {
  if (!item.value) return ''
  const t = targetOf(item.value).split(/[,;/]/)[0].trim()
  return [...t].map((ch, i) => (i === 0 || ch === ' ' || ch === '-' ? ch : '·')).join('')
})

const press = usePress({ onLongPress: () => { showContext.value = !showContext.value } })

const focusInput = async () => { await nextTick(); if (inputEl.value) inputEl.value.focus() }
const summary = (completed) => ({
  total: props.cards.length,
  mistakes: totalMistakes.value,
  rounds: round.value,
  durationMs: Date.now() - started,
  completed,
  missed: props.cards.filter(c => misses.has(c.id)).map(c => ({ card: c, count: misses.get(c.id) })).sort((a, b) => b.count - a.count),
})
defineExpose({ summary: () => summary(false), progress: () => index.value + (round.value - 1) * 1000 })

const advance = () => {
  clearTimeout(advanceTimer)
  status.value = 'input'
  answer.value = ''
  showContext.value = false
  hintShown.value = false
  if (index.value + 1 < roundItems.value.length) {
    index.value++
    focusInput()
    return
  }
  if (nextRound.value.length) {
    phase.value = 'interlude'
    nextTick(() => continueBtn.value && continueBtn.value.focus())
  } else {
    emit('finish', summary(true))
  }
}
const startNextRound = () => {
  round.value++
  roundItems.value = shuffle(nextRound.value).map(it => ({ ...it, side: props.side === 'mixed' ? pickSide() : it.side }))
  nextRound.value = []
  index.value = 0
  phase.value = 'play'
  focusInput()
}
const markWrong = () => {
  status.value = 'wrong'
  totalMistakes.value++
  misses.set(item.value.card.id, (misses.get(item.value.card.id) || 0) + 1)
  if (!nextRound.value.some(it => it.card.id === item.value.card.id)) nextRound.value.push(item.value)
  shakeIt.value = true
  setTimeout(() => { shakeIt.value = false }, 500)
  nextTick(() => nextBtn.value && nextBtn.value.focus())
}
const check = () => {
  if (!answer.value.trim()) return
  if (isCorrectAnswer(answer.value, targetOf(item.value))) {
    status.value = 'right'
    advanceTimer = setTimeout(advance, 700)
  } else {
    markWrong()
  }
}
const giveUp = () => { answer.value = ''; markWrong() }
const onEnter = () => {
  if (status.value === 'input') check()
  else if (status.value === 'wrong') advance()
  else if (status.value === 'right') advance()
}
const onKey = (e) => {
  if (phase.value === 'interlude' && e.key === 'Enter' && e.target.tagName !== 'BUTTON') { e.preventDefault(); startNextRound() }
}
onMounted(() => { focusInput(); window.addEventListener('keydown', onKey) })
onBeforeUnmount(() => { clearTimeout(advanceTimer); window.removeEventListener('keydown', onKey) })
</script>

<style>
.dictation-mode { width: min(620px, 100%); display: flex; flex-direction: column; align-items: stretch; gap: 20px; }
.dict-counters { display: flex; gap: 8px; justify-content: center; flex-wrap: wrap; }
.prompt-card {
  position: relative; min-height: 200px; border-radius: 22px; padding: 18px; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 12px; text-align: center; border: 1px solid rgba(196, 181, 253, 0.22);
  box-shadow: var(--shadow); user-select: none; -webkit-user-select: none; -webkit-touch-callout: none; cursor: default; outline: none;
}
.prompt-card.lang-ru { background: linear-gradient(145deg, #3a2a8a 0%, #2a1f6b 45%, #1c1650 100%); }
.prompt-card.lang-en { background: linear-gradient(145deg, #1f3f9a 0%, #1b2f78 45%, #141f55 100%); }
.prompt-card .lang-chip { position: absolute; top: 14px; left: 16px; }
.prompt-word { font-size: clamp(28px, 5vw, 40px); font-weight: 800; color: #fff; line-height: 1.15; word-break: break-word; }
.prompt-example { font-style: italic; color: rgba(236, 235, 255, 0.85); max-width: 90%; }
.prompt-card .face-hint { position: absolute; bottom: 12px; left: 0; right: 0; }
.answer { display: flex; flex-direction: column; gap: 10px; }
.answer label { display: flex; align-items: center; gap: 10px; }
.hint-pattern { font-family: ui-monospace, Menlo, monospace; letter-spacing: 0.2em; color: var(--warning); font-size: 15px; }
.answer-input { height: 58px; font-size: 22px; font-weight: 700; text-align: center; letter-spacing: 0.01em; }
.answer-input.right { border-color: var(--success); box-shadow: 0 0 0 4px rgba(52, 211, 153, 0.18); }
.answer-input.wrong { border-color: var(--danger); box-shadow: 0 0 0 4px rgba(251, 113, 133, 0.18); }
.verdict { padding: 14px 16px; border-radius: 16px; display: flex; flex-direction: column; gap: 8px; }
.verdict.wrong { background: var(--danger-bg); border: 1px solid rgba(251, 113, 133, 0.35); }
.verdict.wrong .row .icon { color: var(--danger); }
.verdict.right { background: var(--success-bg); border: 1px solid rgba(52, 211, 153, 0.35); }
.verdict.right .row .icon { color: var(--success); }
.diff { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
.diff-line { font-family: ui-monospace, Menlo, monospace; font-size: 18px; letter-spacing: 0.04em; }
.d-ok { color: var(--text); }
.d-extra { color: var(--danger); text-decoration: line-through; }
.d-missing { color: var(--success); background: rgba(52, 211, 153, 0.18); border-radius: 3px; text-decoration: underline; }
.correct { font-size: 17px; }
.correct b { color: #a7f3d0; }
.verdict .ex { font-style: italic; }
.answer-actions { gap: 8px; }
.interlude { display: flex; flex-direction: column; align-items: center; text-align: center; gap: 14px; padding: 34px 24px; }
.inter-icon { width: 60px; height: 60px; border-radius: 18px; display: grid; place-items: center; background: var(--danger-bg); color: #fda4af; }
.inter-words { display: flex; gap: 6px; flex-wrap: wrap; justify-content: center; }
.text { color: var(--text); }
</style>
