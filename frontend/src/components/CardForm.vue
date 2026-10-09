<template>
  <form class="card-form" @submit.prevent="submit">
    <div class="card-form-grid">
      <div class="cf-side ru">
        <div class="cf-label"><span class="lang-chip">RU</span> Русский</div>
        <input ref="first" v-model="form.ru_word" class="input" placeholder="Слово *" :maxlength="limits.word" />
        <textarea v-model="form.ru_example" class="textarea" rows="2" :maxlength="limits.example" placeholder="Пример употребления в контексте"
                  @keydown.enter.exact.prevent="submit"></textarea>
      </div>
      <div class="cf-side en">
        <div class="cf-label"><span class="lang-chip">EN</span> English</div>
        <input v-model="form.en_word" class="input" placeholder="Word *" :maxlength="limits.word" />
        <textarea v-model="form.en_example" class="textarea" rows="2" :maxlength="limits.example" placeholder="Usage example in context"
                  @keydown.enter.exact.prevent="submit"></textarea>
      </div>
    </div>
    <div v-if="error" class="form-error">{{ error }}</div>
    <div class="row wrap cf-actions">
      <span class="muted small hide-sm"><kbd>Enter</kbd> — сохранить</span>
      <div class="spacer"></div>
      <slot name="extra" />
      <button v-if="cancelable" class="btn ghost" type="button" @click="$emit('cancel')">Отмена</button>
      <button class="btn primary" type="submit" :disabled="busy || !valid">
        <Spinner v-if="busy" :size="16" /><Icon v-else :name="initial ? 'check' : 'plus'" :size="16" />
        {{ initial ? 'Сохранить' : 'Добавить карточку' }}
      </button>
    </div>
  </form>
</template>

<script setup>
import { computed, reactive, ref, watch, onMounted } from 'vue'
import { limits } from '../stores/limits.js'

const props = defineProps({
  initial: { type: Object, default: null },
  action: { type: Function, required: true }, // async (card) => void
  cancelable: { type: Boolean, default: false },
  autofocus: { type: Boolean, default: true },
})
const emit = defineEmits(['cancel', 'saved'])

const form = reactive({ ru_word: '', ru_example: '', en_word: '', en_example: '' })
const busy = ref(false)
const error = ref('')
const first = ref(null)

const fill = () => {
  const src = props.initial || {}
  form.ru_word = src.ru_word || ''
  form.ru_example = src.ru_example || ''
  form.en_word = src.en_word || ''
  form.en_example = src.en_example || ''
}
watch(() => props.initial, fill, { immediate: true })
onMounted(() => { if (props.autofocus && first.value) first.value.focus() })

const valid = computed(() => form.ru_word.trim() && form.en_word.trim())

const submit = async () => {
  if (!valid.value || busy.value) return
  busy.value = true
  error.value = ''
  try {
    await props.action({
      ru_word: form.ru_word.trim(), ru_example: form.ru_example.trim(),
      en_word: form.en_word.trim(), en_example: form.en_example.trim(),
    })
    emit('saved')
    if (!props.initial) {
      fill()
      if (first.value) first.value.focus()
    }
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<style>
.card-form { display: flex; flex-direction: column; gap: 12px; }
.card-form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.cf-side { display: flex; flex-direction: column; gap: 8px; padding: 14px; border-radius: 16px; border: 1px solid var(--border); }
.cf-side.ru { background: linear-gradient(160deg, rgba(139, 92, 246, 0.14), rgba(139, 92, 246, 0.03)); }
.cf-side.en { background: linear-gradient(160deg, rgba(59, 130, 246, 0.14), rgba(59, 130, 246, 0.03)); }
.cf-label { display: flex; align-items: center; gap: 8px; font-size: 13px; font-weight: 700; color: var(--text-2); }
.cf-side .textarea { min-height: 64px; }
.cf-actions { gap: 8px; }
@media (max-width: 720px) { .card-form-grid { grid-template-columns: 1fr; } }
</style>
