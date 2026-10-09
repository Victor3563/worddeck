<template>
  <div class="result panel fade-up">
    <div class="result-icon" :class="{ perfect: result.mistakes === 0 }">
      <Icon :name="result.completed ? 'trophy' : 'clock'" :size="34" />
    </div>
    <h1>{{ headline }}</h1>
    <p class="muted">{{ subline }}</p>

    <div class="result-kpis">
      <div class="kpi"><div class="kpi-value">{{ result.total }}</div><div class="kpi-label">{{ plural(result.total, 'карточка', 'карточки', 'карточек') }}</div></div>
      <div class="kpi"><div class="kpi-value" :class="{ bad: result.mistakes }">{{ result.mistakes }}</div><div class="kpi-label">{{ mode === 'learn' ? 'повторов' : 'ошибок' }}</div></div>
      <div class="kpi"><div class="kpi-value">{{ accuracy }}%</div><div class="kpi-label">с первого раза</div></div>
      <div class="kpi"><div class="kpi-value">{{ formatDuration(result.durationMs) }}</div><div class="kpi-label">время</div></div>
    </div>

    <div v-if="result.missed.length" class="hardest">
      <h3>Сложные слова</h3>
      <div class="hardest-list">
        <div v-for="m in result.missed.slice(0, 12)" :key="m.card.id" class="hard-item">
          <span class="strong">{{ m.card.ru_word }}</span>
          <Icon name="arrow-right" :size="14" class="muted" />
          <span>{{ m.card.en_word }}</span>
          <span class="badge red">×{{ m.count }}</span>
        </div>
      </div>
    </div>

    <div class="row wrap result-actions">
      <button class="btn primary lg" type="button" @click="$emit('restart', null)"><Icon name="rotate" /> Ещё раз</button>
      <button v-if="result.missed.length" class="btn lg" type="button" @click="$emit('restart', result.missed.map(m => m.card))">
        <Icon name="target" /> Только сложные ({{ result.missed.length }})
      </button>
    </div>
    <div class="row wrap result-actions">
      <RouterLink v-for="m in otherModes" :key="m.mode" class="btn ghost" :class="{ disabled: m.mode === 'match' && result.total < 2 }"
                  :to="{ name: 'study', params: { mode: m.mode }, query: $route.query }">
        <Icon :name="m.icon" :size="16" /> {{ m.label }}
      </RouterLink>
      <RouterLink class="btn ghost" :to="backTo"><Icon name="arrow-left" :size="16" /> Вернуться</RouterLink>
    </div>
    <p v-if="saved === 'guest'" class="muted small">Войдите в аккаунт, чтобы результаты сохранялись в статистике.</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { formatDuration, plural } from '../../utils.js'

const MODES = [
  { mode: 'learn', label: 'Учить', icon: 'hand' },
  { mode: 'dictation', label: 'Диктант', icon: 'keyboard' },
  { mode: 'match', label: 'Сопоставление', icon: 'match' },
]

const props = defineProps({
  result: { type: Object, required: true },
  mode: { type: String, required: true },
  backTo: { type: [String, Object], required: true },
  saved: { type: String, default: '' },
})
defineEmits(['restart'])

const accuracy = computed(() => {
  if (!props.result.total) return 0
  return Math.round(((props.result.total - props.result.missed.length) / props.result.total) * 100)
})
const headline = computed(() => {
  if (!props.result.completed) return 'Тренировка прервана'
  if (props.result.mistakes === 0) return 'Идеально! 🎉'
  if (accuracy.value >= 70) return 'Отличный результат!'
  return 'Готово — так держать!'
})
const subline = computed(() => {
  if (!props.result.completed) return 'Сохранили то, что вы успели пройти.'
  if (props.mode === 'learn') return 'Все карточки оказались в стопке «Знаю».'
  if (props.mode === 'dictation') return `Все слова написаны без ошибок за ${props.result.rounds} ${plural(props.result.rounds, 'раунд', 'раунда', 'раундов')}.`
  return 'Все пары сопоставлены.'
})
const otherModes = computed(() => MODES.filter(m => m.mode !== props.mode))
</script>

<style>
.result { width: min(640px, 100%); display: flex; flex-direction: column; align-items: center; text-align: center; gap: 14px; padding: 32px 24px; }
.result-icon { width: 76px; height: 76px; border-radius: 24px; display: grid; place-items: center; background: var(--grad); color: #fff; box-shadow: var(--glow); }
.result-icon.perfect { background: linear-gradient(135deg, #f59e0b, #ec4899 60%, #8b5cf6); }
.result-kpis { width: 100%; display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; margin-top: 6px; }
.result-kpis .kpi { padding: 12px; }
.result-kpis .kpi-value { font-size: 21px; }
.kpi-value.bad { color: #fda4af; }
.hardest { width: 100%; text-align: left; display: flex; flex-direction: column; gap: 10px; margin-top: 4px; }
.hardest-list { display: flex; flex-direction: column; gap: 6px; max-height: 220px; overflow-y: auto; }
.hard-item { display: flex; align-items: center; gap: 10px; padding: 8px 12px; border-radius: 12px; background: var(--surface); border: 1px solid var(--border); }
.hard-item .badge { margin-left: auto; }
.result-actions { justify-content: center; }
@media (max-width: 560px) { .result-kpis { grid-template-columns: repeat(2, 1fr); } }
</style>
