<template>
  <div class="container page">
    <div v-if="loading" class="page-loader"><Spinner :size="30" /></div>
    <EmptyState v-else-if="notFound" icon="alert" title="Область не найдена" text="Возможно, она была удалена.">
      <RouterLink to="/app" class="btn primary">К моим областям</RouterLink>
    </EmptyState>
    <template v-else>
      <nav class="crumbs">
        <RouterLink to="/app"><Icon name="layers" :size="14" /> Мои области</RouterLink>
        <Icon name="chevron-right" :size="14" />
        <span>{{ ws.name }}</span>
      </nav>

      <div class="page-head">
        <div class="head-main">
          <div class="ws-badge" :style="{ background: color + '33', color }"><Icon name="layers" :size="22" /></div>
          <div>
            <h1>{{ ws.name }}</h1>
            <p v-if="ws.description" class="sub">{{ ws.description }}</p>
            <p class="sub small">{{ folders.length }} {{ plural(folders.length, 'папка', 'папки', 'папок') }} · {{ cardsWord(totalCards) }}</p>
          </div>
        </div>
        <div class="row wrap">
          <button class="btn" type="button" @click="shareOpen = true"><Icon name="link" :size="16" /> Поделиться</button>
          <button class="btn icon" type="button" title="Изменить" @click="wsDialog = true"><Icon name="edit" :size="16" /></button>
          <button class="btn icon danger" type="button" title="Удалить область" @click="removeWorkspace"><Icon name="trash" :size="16" /></button>
        </div>
      </div>

      <div v-if="totalCards" class="study-strip panel tight">
        <div class="study-strip-text">
          <div class="strong">Тренировать всю область</div>
          <div class="muted small">Карточки из всех папок сразу</div>
        </div>
        <div class="row wrap">
          <RouterLink class="btn primary" :to="studyLink('learn')"><Icon name="hand" :size="16" /> Учить</RouterLink>
          <RouterLink class="btn" :to="studyLink('dictation')"><Icon name="keyboard" :size="16" /> Диктант</RouterLink>
          <RouterLink class="btn" :class="{ disabled: totalCards < 2 }" :to="studyLink('match')"><Icon name="match" :size="16" /> Сопоставление</RouterLink>
        </div>
      </div>

      <div class="section-title">
        <h2><Icon name="folder" /> Папки</h2>
        <button class="btn primary sm" type="button" @click="openCreate"><Icon name="plus" :size="15" /> Папка</button>
      </div>

      <EmptyState v-if="!folders.length" icon="folder" title="Здесь пока нет папок"
                  text="Создайте первую папку — например, «Еда» или «Глаголы движения» — и добавьте в неё карточки.">
        <button class="btn primary" type="button" @click="openCreate"><Icon name="plus" /> Создать папку</button>
      </EmptyState>

      <div v-else class="grid cols-auto">
        <RouterLink v-for="f in folders" :key="f.id" :to="`/folders/${f.id}`" class="tile fade-up folder-tile">
          <div class="folder-icon"><Icon name="folder" :size="22" /></div>
          <div class="tile-title">{{ f.name }}</div>
          <div v-if="f.description" class="tile-desc">{{ f.description }}</div>
          <div class="tile-meta">
            <span class="badge violet"><Icon name="cards" :size="12" /> {{ cardsWord(f.card_count) }}</span>
          </div>
          <div class="tile-actions" @click.prevent.stop>
            <button class="btn ghost icon sm" type="button" title="Поделиться" @click="shareFolder(f)"><Icon name="link" :size="15" /></button>
            <button class="btn ghost icon sm" type="button" title="Изменить" @click="openEdit(f)"><Icon name="edit" :size="15" /></button>
            <button v-if="otherWorkspaces.length" class="btn ghost icon sm" type="button" title="Переместить" @click="openMove(f)"><Icon name="arrow-right" :size="15" /></button>
            <button class="btn ghost icon sm" type="button" title="Удалить" @click="removeFolder(f)"><Icon name="trash" :size="15" /></button>
          </div>
        </RouterLink>
        <button class="tile add" type="button" @click="openCreate">
          <Icon name="plus" :size="26" />
          <span class="strong">Новая папка</span>
        </button>
      </div>
    </template>

    <EntityDialog :open="wsDialog" kind="workspace" :initial="ws" :action="saveWorkspace" @close="wsDialog = false" />
    <EntityDialog :open="folderDialog.open" kind="folder" :initial="folderDialog.item" :action="saveFolder" @close="folderDialog.open = false" />
    <ShareDialog v-if="ws" :open="shareOpen" resource-type="workspace" :resource-id="ws.id" :name="ws.name" @close="shareOpen = false" />
    <ShareDialog :open="folderShare.open" resource-type="folder" :resource-id="folderShare.id" :name="folderShare.name" @close="folderShare.open = false" />

    <Modal :open="move.open" title="Переместить папку" width="420px" @close="move.open = false">
      <div class="field">
        <label for="move-target">В какую область переместить «{{ move.folder && move.folder.name }}»?</label>
        <select id="move-target" v-model="move.target" class="select">
          <option v-for="w in otherWorkspaces" :key="w.id" :value="w.id">{{ w.name }}</option>
        </select>
      </div>
      <template #footer>
        <button class="btn ghost" type="button" @click="move.open = false">Отмена</button>
        <button class="btn primary" type="button" :disabled="!move.target" @click="doMove">Переместить</button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'
import { ui } from '../stores/ui.js'
import { WORKSPACE_COLORS, cardsWord, plural } from '../utils.js'
import EntityDialog from '../components/EntityDialog.vue'
import ShareDialog from '../components/ShareDialog.vue'

const props = defineProps({ id: { type: String, required: true } })

const router = useRouter()
const ws = ref(null)
const folders = ref([])
const allWorkspaces = ref([])
const loading = ref(true)
const notFound = ref(false)
const wsDialog = ref(false)
const shareOpen = ref(false)
const folderDialog = reactive({ open: false, item: null })
const folderShare = reactive({ open: false, id: 0, name: '' })
const move = reactive({ open: false, folder: null, target: null })

const load = async () => {
  loading.value = true
  notFound.value = false
  try {
    const [w, f, all] = await Promise.all([api.workspace(props.id), api.folders(props.id), api.workspaces()])
    ws.value = w
    folders.value = f
    allWorkspaces.value = all
    document.title = w.name + ' · WordDeck'
  } catch (e) {
    if (e.status === 404) notFound.value = true
    else ui.error(e)
  } finally {
    loading.value = false
  }
}
watch(() => props.id, load, { immediate: true })

const color = computed(() => (ws.value && WORKSPACE_COLORS[ws.value.color]) || WORKSPACE_COLORS.violet)
const totalCards = computed(() => folders.value.reduce((s, f) => s + f.card_count, 0))
const otherWorkspaces = computed(() => allWorkspaces.value.filter(w => ws.value && w.id !== ws.value.id))
const studyLink = (mode) => ({ name: 'study', params: { mode }, query: { source: 'workspace', id: props.id } })

const saveWorkspace = async (payload) => {
  ws.value = await api.updateWorkspace(ws.value.id, payload)
  ui.success('Область сохранена')
}
const removeWorkspace = async () => {
  const ok = await ui.confirm({
    title: `Удалить область «${ws.value.name}»?`,
    message: 'Все папки и карточки внутри будут удалены без возможности восстановления.',
    confirmText: 'Удалить', danger: true,
  })
  if (!ok) return
  try {
    await api.deleteWorkspace(ws.value.id)
    ui.success('Область удалена')
    router.push('/app')
  } catch (e) { ui.error(e) }
}

const openCreate = () => { folderDialog.item = null; folderDialog.open = true }
const openEdit = (f) => { folderDialog.item = f; folderDialog.open = true }
const saveFolder = async (payload) => {
  if (folderDialog.item) {
    const upd = await api.updateFolder(folderDialog.item.id, payload)
    folders.value = folders.value.map(f => (f.id === upd.id ? upd : f))
    ui.success('Папка сохранена')
  } else {
    const f = await api.createFolder(ws.value.id, payload)
    folders.value.push(f)
    ui.success('Папка создана')
  }
}
const removeFolder = async (f) => {
  const ok = await ui.confirm({
    title: `Удалить папку «${f.name}»?`,
    message: `Вместе с ней будут удалены ${cardsWord(f.card_count)}.`,
    confirmText: 'Удалить', danger: true,
  })
  if (!ok) return
  try {
    await api.deleteFolder(f.id)
    folders.value = folders.value.filter(x => x.id !== f.id)
    ui.success('Папка удалена')
  } catch (e) { ui.error(e) }
}
const shareFolder = (f) => { folderShare.id = f.id; folderShare.name = f.name; folderShare.open = true }
const openMove = (f) => { move.folder = f; move.target = otherWorkspaces.value[0] ? otherWorkspaces.value[0].id : null; move.open = true }
const doMove = async () => {
  try {
    await api.updateFolder(move.folder.id, { workspace_id: move.target })
    folders.value = folders.value.filter(x => x.id !== move.folder.id)
    const target = allWorkspaces.value.find(w => w.id === move.target)
    ui.success(`Папка перемещена в «${target ? target.name : 'другую область'}»`)
    move.open = false
  } catch (e) { ui.error(e) }
}
</script>

<style>
.ws-badge { width: 52px; height: 52px; border-radius: 16px; display: grid; place-items: center; flex-shrink: 0; }
.study-strip { display: flex; align-items: center; justify-content: space-between; gap: 14px; flex-wrap: wrap; background: var(--grad-soft); border-color: rgba(139, 92, 246, 0.3); }
</style>
