<template>
  <Modal :open="open" title="Импорт карточек" width="760px" @close="$emit('close')">
    <div class="stack">
      <div class="segmented">
        <button type="button" :class="{ active: tab === 'file' }" @click="tab = 'file'"><Icon name="upload" :size="14" /> Файл</button>
        <button type="button" :class="{ active: tab === 'text' }" @click="tab = 'text'"><Icon name="file" :size="14" /> Вставить текст</button>
      </div>

      <div v-if="tab === 'file'" class="dropzone" :class="{ over: dragOver, has: file }"
           @dragover.prevent="dragOver = true" @dragleave.prevent="dragOver = false" @drop.prevent="onDrop"
           @click="$refs.fileInput.click()">
        <input ref="fileInput" type="file" accept=".csv,.tsv,.txt,.json,text/plain,text/csv,application/json" hidden @change="onPick" />
        <Icon name="upload" :size="28" />
        <template v-if="file">
          <div class="strong">{{ file.name }}</div>
          <div class="muted small">{{ (file.size / 1024).toFixed(1) }} КБ · нажмите, чтобы выбрать другой</div>
        </template>
        <template v-else>
          <div class="strong">Перетащите файл сюда или нажмите, чтобы выбрать</div>
          <div class="muted small">CSV, TSV, TXT или JSON, до {{ formatBytes(limits.import_bytes) }}, в кодировке UTF-8</div>
        </template>
      </div>

      <div v-else class="field">
        <textarea v-model="text" class="textarea mono" rows="7"
                  placeholder="дом; Я иду домой.; house; I'm going home.&#10;кот; ; cat;&#10;солнце; sun"></textarea>
      </div>

      <details class="format-help">
        <summary><Icon name="info" :size="14" /> Формат файла</summary>
        <div class="muted small format-body">
          <p>Одна карточка на строку. Разделитель определяется автоматически: <kbd>Tab</kbd>, <kbd>;</kbd>, <kbd>|</kbd> или <kbd>,</kbd>.</p>
          <p><b>4 колонки:</b> слово на русском; пример на русском; слово на английском; пример на английском</p>
          <p><b>2 колонки:</b> слово на русском; слово на английском</p>
          <p>Можно добавить строку-заголовок (<code>ru_word;ru_example;en_word;en_example</code>) — тогда колонки могут идти в любом порядке. Строки, начинающиеся с <code>#</code>, пропускаются.</p>
          <p>JSON: <code>[{"ru_word": "дом", "ru_example": "…", "en_word": "house", "en_example": "…"}]</code></p>
          <p><a href="/sample-cards.csv" download>Скачать пример файла</a></p>
        </div>
      </details>

      <div v-if="error" class="form-error">{{ error }}</div>

      <div v-if="preview" class="preview">
        <div class="row wrap">
          <span class="badge green"><Icon name="check" :size="12" /> {{ preview.parsed }} к импорту</span>
          <span v-if="preview.errors.length" class="badge red"><Icon name="alert" :size="12" /> {{ preview.errors.length }} с ошибками</span>
          <span class="badge">формат: {{ preview.format === 'json' ? 'JSON' : 'текст, разделитель «' + preview.delimiter + '»' }}</span>
        </div>
        <div v-if="preview.errors.length" class="preview-errors">
          <div v-for="e in preview.errors.slice(0, 8)" :key="e.line" class="small">
            <b>Строка {{ e.line }}:</b> {{ errorText(e) }}
          </div>
          <div v-if="preview.errors.length > 8" class="small muted">…и ещё {{ preview.errors.length - 8 }}</div>
        </div>
        <div v-if="preview.cards.length" class="table-wrap preview-table">
          <table class="table">
            <thead><tr><th>Русский</th><th>Пример</th><th>English</th><th>Example</th></tr></thead>
            <tbody>
              <tr v-for="(c, i) in preview.cards.slice(0, 50)" :key="i">
                <td class="strong">{{ c.ru_word }}</td><td class="muted">{{ c.ru_example }}</td>
                <td class="strong">{{ c.en_word }}</td><td class="muted">{{ c.en_example }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="preview.cards.length > 50" class="small muted center">Показаны первые 50 из {{ preview.cards.length }}</div>
        </div>
      </div>
    </div>

    <template #footer>
      <button class="btn ghost" type="button" @click="$emit('close')">Отмена</button>
      <button v-if="!preview" class="btn primary" type="button" :disabled="busy || !hasInput" @click="check">
        <Spinner v-if="busy" :size="16" /> Проверить
      </button>
      <template v-else>
        <button class="btn" type="button" @click="preview = null">Изменить</button>
        <button class="btn primary" type="button" :disabled="busy || !preview.parsed" @click="doImport">
          <Spinner v-if="busy" :size="16" /> Импортировать {{ preview.parsed }}
        </button>
      </template>
    </template>
  </Modal>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { api } from '../api.js'
import { errorText } from '../errors.js'
import { formatBytes, limits } from '../stores/limits.js'
import { ui } from '../stores/ui.js'
import { cardsWord } from '../utils.js'

const props = defineProps({
  open: Boolean,
  folderId: { type: Number, required: true },
})
const emit = defineEmits(['close', 'imported'])

const tab = ref('file')
const file = ref(null)
const text = ref('')
const dragOver = ref(false)
const preview = ref(null)
const error = ref('')
const busy = ref(false)

watch(() => props.open, (v) => {
  if (v) { file.value = null; text.value = ''; preview.value = null; error.value = ''; tab.value = 'file' }
})
watch([tab, file, text], () => { preview.value = null; error.value = '' })

const hasInput = computed(() => (tab.value === 'file' ? !!file.value : text.value.trim().length > 0))
const source = () => (tab.value === 'file' ? { file: file.value } : { text: text.value })

const onPick = (e) => { file.value = e.target.files[0] || null; e.target.value = '' }
const onDrop = (e) => { dragOver.value = false; file.value = e.dataTransfer.files[0] || null }

const check = async () => {
  busy.value = true
  error.value = ''
  try {
    preview.value = await api.importCards(props.folderId, source(), true)
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
const doImport = async () => {
  busy.value = true
  try {
    const res = await api.importCards(props.folderId, source(), false)
    ui.success(`Добавлено: ${cardsWord(res.created)}`)
    emit('imported', res.created)
    emit('close')
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<style>
.dropzone {
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 6px; text-align: center;
  padding: 34px 20px; border-radius: 18px; border: 2px dashed var(--border-strong); background: rgba(8, 6, 26, 0.35);
  color: var(--text-2); cursor: pointer; transition: border-color 0.2s, background 0.2s;
}
.dropzone:hover, .dropzone.over { border-color: var(--violet-2); background: rgba(139, 92, 246, 0.08); }
.dropzone.has { border-style: solid; border-color: rgba(52, 211, 153, 0.45); }
.dropzone .strong { font-weight: 700; color: var(--text); }
.mono { font-family: ui-monospace, 'SFMono-Regular', Menlo, monospace; font-size: 13.5px; }
.format-help summary { cursor: pointer; color: var(--text-2); font-weight: 600; font-size: 14px; display: flex; align-items: center; gap: 6px; }
.format-body { display: flex; flex-direction: column; gap: 6px; padding: 10px 2px 0; }
.format-body code { background: var(--surface-2); padding: 1px 6px; border-radius: 6px; font-size: 12px; }
.preview { display: flex; flex-direction: column; gap: 12px; }
.preview-errors { padding: 10px 12px; border-radius: 12px; background: var(--danger-bg); border: 1px solid rgba(251, 113, 133, 0.25); display: flex; flex-direction: column; gap: 4px; }
.preview-table { max-height: 280px; overflow: auto; border: 1px solid var(--border); border-radius: 12px; }
.preview-table .strong { font-weight: 700; }
</style>
