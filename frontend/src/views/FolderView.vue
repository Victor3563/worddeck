<template>
  <div class="container page">
    <div v-if="loading" class="page-loader"><Spinner :size="30" /></div>
    <EmptyState v-else-if="notFound" icon="alert" title="Папка не найдена" text="Возможно, она была удалена или перемещена.">
      <RouterLink to="/app" class="btn primary">К моим областям</RouterLink>
    </EmptyState>
    <template v-else>
      <nav class="crumbs">
        <RouterLink to="/app"><Icon name="layers" :size="14" /> Мои области</RouterLink>
        <Icon name="chevron-right" :size="14" />
        <RouterLink :to="`/workspaces/${folder.workspace_id}`">{{ folder.workspace_name }}</RouterLink>
        <Icon name="chevron-right" :size="14" />
        <span>{{ folder.name }}</span>
      </nav>

      <div class="page-head">
        <div class="head-main">
          <div class="folder-icon big"><Icon name="folder" :size="24" /></div>
          <div>
            <h1>{{ folder.name }}</h1>
            <p v-if="folder.description" class="sub">{{ folder.description }}</p>
            <p class="sub small">{{ cardsWord(cards.length) }}</p>
          </div>
        </div>
        <div class="row wrap">
          <button class="btn" type="button" @click="importOpen = true"><Icon name="upload" :size="16" /> Импорт</button>
          <div class="menu-wrap" ref="exportRoot">
            <button class="btn" type="button" :disabled="!cards.length" @click="exportMenu = !exportMenu"><Icon name="download" :size="16" /> Экспорт</button>
            <div v-if="exportMenu" class="dropdown">
              <button class="dropdown-item" type="button" @click="doExport('csv')"><Icon name="file" :size="16" /> CSV (Excel, Google Таблицы)</button>
              <button class="dropdown-item" type="button" @click="doExport('json')"><Icon name="file" :size="16" /> JSON</button>
            </div>
          </div>
          <button class="btn" type="button" @click="shareFolderOpen = true"><Icon name="link" :size="16" /> Поделиться</button>
          <button class="btn icon" type="button" title="Изменить папку" @click="folderDialog = true"><Icon name="edit" :size="16" /></button>
          <button class="btn icon danger" type="button" title="Удалить папку" @click="removeFolder"><Icon name="trash" :size="16" /></button>
        </div>
      </div>

      <div class="modes">
        <RouterLink class="mode-card learn" :class="{ disabled: !cards.length }" :to="studyLink('learn')">
          <span class="mode-icon learn"><Icon name="hand" :size="20" /></span>
          <div><div class="strong">Учить</div><div class="muted small">Свайп влево — знаю, вправо — ещё раз</div></div>
          <Icon name="arrow-right" class="go" />
        </RouterLink>
        <RouterLink class="mode-card dictation" :class="{ disabled: !cards.length }" :to="studyLink('dictation')">
          <span class="mode-icon dictation"><Icon name="keyboard" :size="20" /></span>
          <div><div class="strong">Диктант</div><div class="muted small">Напишите перевод без ошибок</div></div>
          <Icon name="arrow-right" class="go" />
        </RouterLink>
        <RouterLink class="mode-card match" :class="{ disabled: cards.length < 2 }" :to="studyLink('match')">
          <span class="mode-icon match"><Icon name="match" :size="20" /></span>
          <div><div class="strong">Сопоставление</div><div class="muted small">Перетащите слово к переводу</div></div>
          <Icon name="arrow-right" class="go" />
        </RouterLink>
      </div>

      <section class="panel add-panel">
        <button class="add-toggle" type="button" @click="addOpen = !addOpen" :aria-expanded="addOpen">
          <span class="row"><Icon name="plus" /> <span class="strong">Добавить карточку</span></span>
          <span class="muted small hide-sm">или загрузите сразу много через «Импорт»</span>
          <Icon name="chevron-down" class="chev" :class="{ open: addOpen }" />
        </button>
        <div v-if="addOpen" class="add-body">
          <CardForm :action="addCard" :autofocus="cards.length > 0" />
        </div>
      </section>

      <div v-if="cards.length" class="toolbar">
        <div class="input-icon search">
          <Icon name="search" :size="16" />
          <input v-model="query" class="input" placeholder="Поиск по словам и примерам" />
        </div>
        <div class="segmented" role="group" aria-label="Какую сторону показывать">
          <button type="button" :class="{ active: side === 'ru' }" @click="side = 'ru'">RU → EN</button>
          <button type="button" :class="{ active: side === 'en' }" @click="side = 'en'">EN → RU</button>
        </div>
        <div class="spacer"></div>
        <template v-if="selectMode">
          <span class="muted small">Выбрано: {{ selected.size }}</span>
          <button class="btn sm" type="button" @click="toggleAll">{{ selected.size === filtered.length ? 'Снять все' : 'Выбрать все' }}</button>
          <button class="btn sm danger" type="button" :disabled="!selected.size" @click="deleteSelected"><Icon name="trash" :size="14" /> Удалить</button>
          <button class="btn sm ghost" type="button" @click="exitSelect">Готово</button>
        </template>
        <button v-else class="btn sm" type="button" @click="selectMode = true"><Icon name="check-square" :size="14" /> Выбрать</button>
      </div>

      <p v-if="cards.length" class="muted small gesture-hint">
        <Icon name="info" :size="14" /> Нажмите на карточку, чтобы перевернуть. Удерживайте, чтобы увидеть пример употребления.
      </p>

      <EmptyState v-if="!cards.length" icon="cards" title="В папке пока нет карточек"
                  text="Добавьте первую карточку через форму выше или импортируйте список слов из файла.">
        <button class="btn primary" type="button" @click="importOpen = true"><Icon name="upload" /> Импорт из файла</button>
      </EmptyState>
      <EmptyState v-else-if="!filtered.length" icon="search" title="Ничего не найдено" :text="`По запросу «${query}» карточек нет.`" />

      <div v-else class="card-grid">
        <div v-for="c in filtered" :key="c.id" class="card-cell" :class="{ selecting: selectMode, selected: selected.has(c.id) }">
          <FlipCard :card="c" :side="side" :interactive="!selectMode">
            <template #actions>
              <button class="btn icon sm" type="button" title="Поделиться карточкой" @click="shareCard(c)"><Icon name="link" :size="14" /></button>
              <button class="btn icon sm" type="button" title="Изменить" @click="editCard(c)"><Icon name="edit" :size="14" /></button>
              <button class="btn icon sm" type="button" title="Удалить" @click="removeCard(c)"><Icon name="trash" :size="14" /></button>
            </template>
          </FlipCard>
          <button v-if="selectMode" class="select-overlay" type="button" @click="toggleSelect(c.id)" :aria-pressed="selected.has(c.id)">
            <Icon :name="selected.has(c.id) ? 'check-square' : 'square'" :size="22" />
          </button>
        </div>
      </div>
    </template>

    <EntityDialog :open="folderDialog" kind="folder" :initial="folder" :action="saveFolder" @close="folderDialog = false" />
    <ImportDialog v-if="folder" :open="importOpen" :folder-id="folder.id" @close="importOpen = false" @imported="reloadCards" />
    <ShareDialog v-if="folder" :open="shareFolderOpen" resource-type="folder" :resource-id="folder.id" :name="folder.name" @close="shareFolderOpen = false" />
    <ShareDialog :open="cardShare.open" resource-type="card" :resource-id="cardShare.id" :name="cardShare.name" @close="cardShare.open = false" />

    <Modal :open="edit.open" title="Изменить карточку" width="760px" @close="edit.open = false">
      <CardForm v-if="edit.card" :initial="edit.card" :action="saveCard" cancelable @cancel="edit.open = false">
        <template #extra>
          <select v-if="siblings.length > 1" v-model="edit.folderId" class="select move-select" title="Папка">
            <option v-for="f in siblings" :key="f.id" :value="f.id">📁 {{ f.name }}</option>
          </select>
        </template>
      </CardForm>
    </Modal>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'
import { ui } from '../stores/ui.js'
import { cardsWord } from '../utils.js'
import FlipCard from '../components/FlipCard.vue'
import CardForm from '../components/CardForm.vue'
import EntityDialog from '../components/EntityDialog.vue'
import ImportDialog from '../components/ImportDialog.vue'
import ShareDialog from '../components/ShareDialog.vue'

const SIDE_KEY = 'worddeck.side'

const props = defineProps({ id: { type: String, required: true } })

const router = useRouter()
const folder = ref(null)
const cards = ref([])
const siblings = ref([])
const loading = ref(true)
const notFound = ref(false)
const query = ref('')
const side = ref((() => { try { return localStorage.getItem(SIDE_KEY) || 'ru' } catch { return 'ru' } })())
const addOpen = ref(false)
const importOpen = ref(false)
const folderDialog = ref(false)
const shareFolderOpen = ref(false)
const exportMenu = ref(false)
const exportRoot = ref(null)
const cardShare = reactive({ open: false, id: 0, name: '' })
const edit = reactive({ open: false, card: null, folderId: null })
const selectMode = ref(false)
const selected = reactive(new Set())

watch(side, (v) => { try { localStorage.setItem(SIDE_KEY, v) } catch { /* localStorage недоступен, например в приватном режиме */ } })

const load = async () => {
  loading.value = true
  notFound.value = false
  try {
    const [f, c] = await Promise.all([api.folder(props.id), api.cards(props.id)])
    folder.value = f
    cards.value = c
    addOpen.value = c.length === 0
    document.title = f.name + ' · WordDeck'
    api.folders(f.workspace_id).then(list => { siblings.value = list }).catch(() => {})
  } catch (e) {
    if (e.status === 404) notFound.value = true
    else ui.error(e)
  } finally {
    loading.value = false
  }
}
watch(() => props.id, load, { immediate: true })

const reloadCards = async () => { cards.value = await api.cards(props.id) }

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return cards.value
  return cards.value.filter(c => [c.ru_word, c.en_word, c.ru_example, c.en_example].some(s => s.toLowerCase().includes(q)))
})

const studyLink = (mode) => ({ name: 'study', params: { mode }, query: { source: 'folder', id: props.id } })

const addCard = async (payload) => {
  const c = await api.createCard(folder.value.id, payload)
  cards.value.push(c)
  ui.success(`Карточка «${c.en_word}» добавлена`)
}
const editCard = (c) => { edit.card = c; edit.folderId = c.folder_id; edit.open = true }
const saveCard = async (payload) => {
  const body = { ...payload }
  if (edit.folderId && edit.folderId !== edit.card.folder_id) body.folder_id = edit.folderId
  const upd = await api.updateCard(edit.card.id, body)
  if (upd.folder_id !== folder.value.id) {
    cards.value = cards.value.filter(c => c.id !== upd.id)
    const target = siblings.value.find(f => f.id === upd.folder_id)
    ui.success(`Карточка перемещена в «${target ? target.name : 'другую папку'}»`)
  } else {
    cards.value = cards.value.map(c => (c.id === upd.id ? upd : c))
    ui.success('Карточка сохранена')
  }
  edit.open = false
}
const removeCard = async (c) => {
  const ok = await ui.confirm({ title: 'Удалить карточку?', message: `«${c.ru_word} — ${c.en_word}» будет удалена.`, confirmText: 'Удалить', danger: true })
  if (!ok) return
  try {
    await api.deleteCard(c.id)
    cards.value = cards.value.filter(x => x.id !== c.id)
    ui.success('Карточка удалена')
  } catch (e) { ui.error(e) }
}
const shareCard = (c) => { cardShare.id = c.id; cardShare.name = `${c.ru_word} — ${c.en_word}`; cardShare.open = true }

const toggleSelect = (id) => { selected.has(id) ? selected.delete(id) : selected.add(id) }
const toggleAll = () => {
  if (selected.size === filtered.value.length) selected.clear()
  else filtered.value.forEach(c => selected.add(c.id))
}
const exitSelect = () => { selectMode.value = false; selected.clear() }
const deleteSelected = async () => {
  const n = selected.size
  const ok = await ui.confirm({ title: `Удалить ${cardsWord(n)}?`, message: 'Это действие нельзя отменить.', confirmText: 'Удалить', danger: true })
  if (!ok) return
  try {
    const ids = [...selected]
    await api.deleteCards(folder.value.id, ids)
    cards.value = cards.value.filter(c => !selected.has(c.id))
    ui.success(`Удалено: ${cardsWord(n)}`)
    exitSelect()
  } catch (e) { ui.error(e) }
}

const saveFolder = async (payload) => {
  const upd = await api.updateFolder(folder.value.id, payload)
  folder.value = upd
  ui.success('Папка сохранена')
}
const removeFolder = async () => {
  const ok = await ui.confirm({
    title: `Удалить папку «${folder.value.name}»?`,
    message: `Вместе с ней будут удалены ${cardsWord(cards.value.length)}.`,
    confirmText: 'Удалить', danger: true,
  })
  if (!ok) return
  try {
    await api.deleteFolder(folder.value.id)
    ui.success('Папка удалена')
    router.push(`/workspaces/${folder.value.workspace_id}`)
  } catch (e) { ui.error(e) }
}

const doExport = async (format) => {
  exportMenu.value = false
  try {
    const res = await api.exportCards(folder.value.id, format)
    const blob = await res.blob()
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `${folder.value.name}.${format}`
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(a.href), 1000)
  } catch (e) { ui.error(e) }
}
const onDoc = (e) => { if (exportRoot.value && !exportRoot.value.contains(e.target)) exportMenu.value = false }
onMounted(() => document.addEventListener('click', onDoc))
onBeforeUnmount(() => document.removeEventListener('click', onDoc))
</script>

<style>
.folder-icon.big { width: 52px; height: 52px; border-radius: 16px; flex-shrink: 0; }
.menu-wrap { position: relative; }
.menu-wrap .dropdown { min-width: 250px; }
.modes { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; margin-bottom: 18px; }
.mode-card {
  display: flex; align-items: center; gap: 14px; padding: 16px; border-radius: var(--radius-lg);
  background: var(--surface); border: 1px solid var(--border); color: var(--text); transition: all 0.2s;
}
.mode-card:hover { color: var(--text); transform: translateY(-2px); box-shadow: var(--shadow); border-color: var(--border-strong); background: var(--surface-2); }
.mode-card.learn { background: linear-gradient(135deg, rgba(139, 92, 246, 0.2), rgba(139, 92, 246, 0.04)); }
.mode-card.dictation { background: linear-gradient(135deg, rgba(59, 130, 246, 0.2), rgba(59, 130, 246, 0.04)); }
.mode-card.match { background: linear-gradient(135deg, rgba(34, 211, 238, 0.14), rgba(34, 211, 238, 0.03)); }
.mode-card .mode-icon { width: 44px; height: 44px; border-radius: 14px; }
.mode-card > div { flex: 1; min-width: 0; }
.mode-card .go { color: var(--muted); transition: transform 0.2s; }
.mode-card:hover .go { transform: translateX(3px); color: var(--text); }
.mode-card.disabled { opacity: 0.45; pointer-events: none; }
.add-panel { padding: 0; margin-bottom: 18px; overflow: hidden; }
.add-toggle { width: 100%; display: flex; align-items: center; gap: 14px; padding: 16px 20px; background: none; border: 0; color: var(--text); font: inherit; cursor: pointer; text-align: left; }
.add-toggle:hover { background: var(--surface); }
.add-toggle .chev { margin-left: auto; transition: transform 0.2s; color: var(--muted); }
.add-toggle .chev.open { transform: rotate(180deg); }
.add-body { padding: 4px 20px 20px; }
.toolbar .search { flex: 1; min-width: 220px; max-width: 380px; }
.gesture-hint { display: flex; align-items: center; gap: 6px; margin-bottom: 14px; }
.card-cell { position: relative; }
.card-cell.selecting .flip-card { opacity: 0.75; transition: opacity 0.2s; }
.card-cell.selected .flip-card { opacity: 1; }
.card-cell.selected .flip-face { box-shadow: 0 0 0 2px var(--violet-2), var(--shadow); }
.select-overlay { position: absolute; inset: 0; border: 0; border-radius: 20px; background: transparent; cursor: pointer; display: flex; align-items: flex-start; justify-content: flex-end; padding: 12px; color: #fff; }
.move-select { width: auto; height: 40px; max-width: 220px; }
@media (max-width: 860px) { .modes { grid-template-columns: 1fr; } }
</style>
