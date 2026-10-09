<template>
  <Modal :open="open" :title="title" width="480px" @close="$emit('close')">
    <form class="stack" @submit.prevent="submit" :id="uid + '-form'">
      <div class="field">
        <label :for="uid + '-name'">Название</label>
        <input :id="uid + '-name'" v-model="form.name" class="input" :maxlength="limits.name" autofocus
               :placeholder="kind === 'workspace' ? 'Например, Английский B2' : 'Например, Путешествия'" />
      </div>
      <div class="field">
        <label :for="uid + '-desc'">Описание <span class="muted">(необязательно)</span></label>
        <textarea :id="uid + '-desc'" v-model="form.description" class="textarea" :maxlength="limits.description" rows="3"
                  placeholder="Коротко о том, что здесь будет"></textarea>
      </div>
      <div v-if="kind === 'workspace'" class="field">
        <span class="label">Цвет</span>
        <div class="color-row">
          <button v-for="(hex, key) in WORKSPACE_COLORS" :key="key" type="button" class="color-dot" :class="{ active: form.color === key }"
                  :style="{ background: hex }" :aria-label="key" @click="form.color = key">
            <Icon v-if="form.color === key" name="check" :size="14" />
          </button>
        </div>
      </div>
      <div v-if="error" class="form-error">{{ error }}</div>
    </form>
    <template #footer>
      <button class="btn ghost" type="button" @click="$emit('close')">Отмена</button>
      <button class="btn primary" type="submit" :form="uid + '-form'" :disabled="busy || !form.name.trim()">
        <Spinner v-if="busy" :size="16" /> {{ initial ? 'Сохранить' : 'Создать' }}
      </button>
    </template>
  </Modal>
</template>

<script>
// Общий для всех экземпляров, чтобы id элементов не повторялись на странице.
let instances = 0
const nextUid = () => `ent${++instances}`
</script>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { WORKSPACE_COLORS } from '../utils.js'
import { limits } from '../stores/limits.js'

const props = defineProps({
  open: Boolean,
  kind: { type: String, default: 'folder' }, // workspace | folder
  initial: { type: Object, default: null },
  action: { type: Function, required: true }, // async (payload) => void
})
const emit = defineEmits(['close'])

const uid = nextUid()
const form = reactive({ name: '', description: '', color: 'violet' })
const busy = ref(false)
const error = ref('')

watch(() => props.open, (v) => {
  if (!v) return
  error.value = ''
  form.name = props.initial ? props.initial.name : ''
  form.description = props.initial ? props.initial.description : ''
  form.color = (props.initial && props.initial.color) || 'violet'
}, { immediate: true })

const title = computed(() => {
  const what = props.kind === 'workspace' ? 'область' : 'папку'
  return props.initial ? `Изменить ${what}` : `Новая ${props.kind === 'workspace' ? 'область' : 'папка'}`
})

const submit = async () => {
  if (!form.name.trim()) return
  busy.value = true
  error.value = ''
  try {
    const payload = { name: form.name.trim(), description: form.description.trim() }
    if (props.kind === 'workspace') payload.color = form.color
    await props.action(payload)
    emit('close')
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<style>
.color-row { display: flex; gap: 10px; flex-wrap: wrap; }
.color-dot { width: 32px; height: 32px; border-radius: 50%; border: 2px solid transparent; cursor: pointer; display: grid; place-items: center; color: #fff; transition: transform 0.15s; }
.color-dot:hover { transform: scale(1.1); }
.color-dot.active { border-color: #fff; box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.15); }
</style>
