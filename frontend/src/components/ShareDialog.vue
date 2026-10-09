<template>
  <Modal :open="open" title="Поделиться по ссылке" width="520px" @close="$emit('close')">
    <div class="stack">
      <div class="share-target">
        <div class="share-icon"><Icon :name="icon" :size="20" /></div>
        <div>
          <div class="muted small">{{ typeLabel }}</div>
          <div class="strong">{{ name }}</div>
        </div>
      </div>

      <div v-if="loading" class="center"><Spinner /></div>
      <template v-else-if="link">
        <p class="muted small">Любой, у кого есть ссылка, сможет посмотреть и потренировать эти карточки, а после входа — скопировать их к себе. Изменения, которые вы вносите, сразу видны по ссылке.</p>
        <div class="share-url">
          <input class="input" :value="url" readonly @focus="$event.target.select()" />
          <button class="btn primary" type="button" @click="copy"><Icon name="copy" :size="16" /> Копировать</button>
        </div>
        <div class="row wrap">
          <a class="btn sm" :href="url" target="_blank" rel="noopener"><Icon name="external" :size="14" /> Открыть</a>
          <div class="spacer"></div>
          <button class="btn sm danger" type="button" :disabled="busy" @click="revoke"><Icon name="x" :size="14" /> Отозвать ссылку</button>
        </div>
      </template>
      <template v-else>
        <p class="muted">Ссылки пока нет. Создайте её, чтобы делиться {{ typeGenitive }} с кем угодно — даже без аккаунта.</p>
        <button class="btn primary" type="button" :disabled="busy" @click="create">
          <Spinner v-if="busy" :size="16" /><Icon v-else name="link" :size="16" /> Создать ссылку
        </button>
      </template>
    </div>
  </Modal>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { api, shareUrl } from '../api.js'
import { ui, copyText } from '../stores/ui.js'

const LABELS = {
  workspace: ['Область', 'областью', 'layers'],
  folder: ['Папка', 'папкой', 'folder'],
  card: ['Карточка', 'карточкой', 'cards'],
}

const props = defineProps({
  open: Boolean,
  resourceType: { type: String, required: true },
  resourceId: { type: Number, default: 0 },
  name: { type: String, default: '' },
})
defineEmits(['close'])

const link = ref(null)
const loading = ref(false)
const busy = ref(false)

watch(() => [props.open, props.resourceId], async ([open]) => {
  if (!open || !props.resourceId) return
  loading.value = true
  link.value = null
  try {
    const list = await api.shares({ type: props.resourceType, id: props.resourceId })
    link.value = list[0] || null
  } catch (e) {
    ui.error(e)
  } finally {
    loading.value = false
  }
}, { immediate: true })

const url = computed(() => (link.value ? shareUrl(link.value.token) : ''))
const create = async () => {
  busy.value = true
  try {
    link.value = await api.createShare(props.resourceType, props.resourceId)
    await copyText(url.value)
  } catch (e) { ui.error(e) } finally { busy.value = false }
}
const revoke = async () => {
  const ok = await ui.confirm({ title: 'Отозвать ссылку?', message: 'Ссылка перестанет работать у всех, с кем вы ей поделились.', confirmText: 'Отозвать', danger: true })
  if (!ok) return
  busy.value = true
  try {
    await api.deleteShare(link.value.token)
    link.value = null
    ui.success('Ссылка отозвана')
  } catch (e) { ui.error(e) } finally { busy.value = false }
}
const copy = () => copyText(url.value)

const typeLabel = computed(() => LABELS[props.resourceType][0])
const typeGenitive = computed(() => LABELS[props.resourceType][1])
const icon = computed(() => LABELS[props.resourceType][2])
</script>

<style>
.share-target { display: flex; align-items: center; gap: 12px; padding: 12px; border-radius: 14px; background: var(--surface); border: 1px solid var(--border); }
.share-target .strong { font-weight: 700; word-break: break-word; }
.share-url { display: flex; gap: 8px; }
.share-url .input { font-size: 13px; }
@media (max-width: 480px) { .share-url { flex-direction: column; } }
</style>
