<template>
  <div class="container page">
    <div v-if="loading" class="page-loader"><Spinner :size="30" /></div>
    <EmptyState v-else-if="error" icon="link" title="Ссылка недействительна" :text="error">
      <RouterLink :to="auth.loggedIn ? '/app' : '/'" class="btn primary">На главную</RouterLink>
    </EmptyState>

    <template v-else>
      <div class="shared-banner panel tight">
        <div class="share-icon"><Icon name="link" :size="18" /></div>
        <div class="grow">
          <div class="strong">{{ data.is_owner ? 'Это ваша ссылка' : `${data.owner_name} поделился(ась) с вами` }}</div>
          <div class="muted small">{{ typeText }} · только для просмотра и тренировки</div>
        </div>
        <template v-if="!data.is_owner">
          <button v-if="auth.loggedIn" class="btn primary" type="button" @click="openCopy"><Icon name="bookmark" :size="16" /> Сохранить себе</button>
          <RouterLink v-else class="btn primary" :to="{ name: 'login', query: { redirect: $route.fullPath } }"><Icon name="login" :size="16" /> Войти, чтобы сохранить</RouterLink>
        </template>
        <RouterLink v-else class="btn" to="/shares">Мои ссылки</RouterLink>
      </div>

      <template v-if="data.workspace">
        <div class="page-head">
          <div>
            <h1>{{ data.workspace.name }}</h1>
            <p v-if="data.workspace.description" class="sub">{{ data.workspace.description }}</p>
            <p class="sub small">{{ data.workspace.folders.length }} {{ plural(data.workspace.folders.length, 'папка', 'папки', 'папок') }} · {{ cardsWord(totalCards) }}</p>
          </div>
          <StudyButtons :query="{ source: 'share', token }" :count="totalCards" />
        </div>
        <EmptyState v-if="!data.workspace.folders.length" icon="folder" title="В области пока нет папок" />
        <div v-for="f in data.workspace.folders" :key="f.id" class="shared-folder panel">
          <button class="shared-folder-head" type="button" @click="toggle(f.id)">
            <div class="folder-icon"><Icon name="folder" :size="20" /></div>
            <div class="grow">
              <div class="strong">{{ f.name }}</div>
              <div class="muted small">{{ cardsWord(f.cards.length) }}<template v-if="f.description"> · {{ f.description }}</template></div>
            </div>
            <Icon name="chevron-down" class="chev" :class="{ open: expanded.has(f.id) }" />
          </button>
          <div v-if="expanded.has(f.id)" class="shared-folder-body">
            <StudyButtons :query="{ source: 'share', token, folder: f.id }" :count="f.cards.length" small />
            <div class="card-grid">
              <FlipCard v-for="c in f.cards" :key="c.id" :card="c" :side="side" />
            </div>
          </div>
        </div>
      </template>

      <template v-else-if="data.folder">
        <div class="page-head">
          <div>
            <h1>{{ data.folder.name }}</h1>
            <p v-if="data.folder.description" class="sub">{{ data.folder.description }}</p>
            <p class="sub small">{{ cardsWord(data.folder.cards.length) }}</p>
          </div>
          <StudyButtons :query="{ source: 'share', token }" :count="data.folder.cards.length" />
        </div>
        <div class="toolbar">
          <div class="segmented">
            <button type="button" :class="{ active: side === 'ru' }" @click="side = 'ru'">RU → EN</button>
            <button type="button" :class="{ active: side === 'en' }" @click="side = 'en'">EN → RU</button>
          </div>
        </div>
        <EmptyState v-if="!data.folder.cards.length" icon="cards" title="В папке пока нет карточек" />
        <div v-else class="card-grid">
          <FlipCard v-for="c in data.folder.cards" :key="c.id" :card="c" :side="side" />
        </div>
      </template>

      <template v-else-if="data.card">
        <div class="single-card">
          <FlipCard :card="data.card" :side="side" size="study" />
          <p class="muted small center">Нажмите, чтобы перевернуть · удерживайте, чтобы увидеть пример</p>
        </div>
      </template>
    </template>

    <Modal :open="copy.open" title="Сохранить себе" width="460px" @close="copy.open = false">
      <div class="stack">
        <p v-if="data && data.resource_type === 'workspace'" class="muted">
          Будет создана новая область «{{ data.workspace.name }}» со всеми папками и карточками. Дальше это будет ваша копия — изменения автора на неё не повлияют.
        </p>
        <template v-else>
          <div class="field">
            <label for="copy-ws">Область</label>
            <select id="copy-ws" v-model="copy.workspace" class="select" @change="loadTargetFolders">
              <option v-for="w in copy.workspaces" :key="w.id" :value="w.id">{{ w.name }}</option>
            </select>
          </div>
          <div v-if="data && data.resource_type === 'card'" class="field">
            <label for="copy-folder">Папка</label>
            <select id="copy-folder" v-model="copy.folder" class="select" :disabled="!copy.folders.length">
              <option v-for="f in copy.folders" :key="f.id" :value="f.id">{{ f.name }}</option>
            </select>
            <span v-if="!copy.folders.length" class="hint">В этой области нет папок — сначала создайте папку.</span>
          </div>
        </template>
      </div>
      <template #footer>
        <button class="btn ghost" type="button" @click="copy.open = false">Отмена</button>
        <button class="btn primary" type="button" :disabled="copy.busy || !copyTarget" @click="doCopy">
          <Spinner v-if="copy.busy" :size="16" /> Сохранить
        </button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'
import { auth } from '../stores/auth.js'
import { ui } from '../stores/ui.js'
import { cardsWord, plural } from '../utils.js'
import FlipCard from '../components/FlipCard.vue'
import StudyButtons from '../components/StudyButtons.vue'

const TYPE_TEXT = { workspace: 'Область', folder: 'Папка', card: 'Карточка' }

const props = defineProps({ token: { type: String, required: true } })

const router = useRouter()
const data = ref(null)
const loading = ref(true)
const error = ref('')
const side = ref('ru')
const expanded = reactive(new Set())
const copy = reactive({ open: false, busy: false, workspaces: [], workspace: null, folders: [], folder: null })

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    data.value = await api.publicShare(props.token)
    const name = data.value.workspace ? data.value.workspace.name : data.value.folder ? data.value.folder.name : data.value.card.en_word
    document.title = name + ' · WordDeck'
    if (data.value.workspace && data.value.workspace.folders[0]) expanded.add(data.value.workspace.folders[0].id)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
watch(() => props.token, load, { immediate: true })

const typeText = computed(() => (data.value ? TYPE_TEXT[data.value.resource_type] : ''))
const totalCards = computed(() => (data.value && data.value.workspace ? data.value.workspace.folders.reduce((s, f) => s + f.cards.length, 0) : 0))
const toggle = (id) => { expanded.has(id) ? expanded.delete(id) : expanded.add(id) }

const loadTargetFolders = async () => {
  copy.folders = []
  copy.folder = null
  if (data.value.resource_type !== 'card' || !copy.workspace) return
  copy.folders = await api.folders(copy.workspace)
  copy.folder = copy.folders[0] ? copy.folders[0].id : null
}
const openCopy = async () => {
  copy.open = true
  if (data.value.resource_type === 'workspace') return
  try {
    copy.workspaces = await api.workspaces()
    copy.workspace = copy.workspaces[0] ? copy.workspaces[0].id : null
    await loadTargetFolders()
  } catch (e) { ui.error(e) }
}
const copyTarget = computed(() => {
  if (!data.value) return null
  if (data.value.resource_type === 'workspace') return -1
  if (data.value.resource_type === 'folder') return copy.workspace
  return copy.folder
})
const doCopy = async () => {
  copy.busy = true
  try {
    const res = await api.copyShare(props.token, copyTarget.value > 0 ? copyTarget.value : 0)
    copy.open = false
    ui.success(`Сохранено: ${cardsWord(res.cards_copied)}`)
    if (res.resource_type === 'workspace') router.push(`/workspaces/${res.id}`)
    else if (res.resource_type === 'folder') router.push(`/folders/${res.id}`)
    else router.push(`/folders/${copy.folder}`)
  } catch (e) {
    ui.error(e)
  } finally {
    copy.busy = false
  }
}

  
</script>

<style>
.shared-banner { display: flex; align-items: center; gap: 14px; flex-wrap: wrap; margin-bottom: 26px; background: var(--grad-soft); border-color: rgba(139, 92, 246, 0.3); }
.shared-folder { padding: 0; margin-bottom: 12px; overflow: hidden; }
.shared-folder-head { width: 100%; display: flex; align-items: center; gap: 14px; padding: 14px 18px; background: none; border: 0; color: var(--text); font: inherit; text-align: left; cursor: pointer; }
.shared-folder-head:hover { background: var(--surface); }
.shared-folder-head .chev { color: var(--muted); transition: transform 0.2s; }
.shared-folder-head .chev.open { transform: rotate(180deg); }
.shared-folder-body { padding: 4px 18px 18px; display: flex; flex-direction: column; gap: 14px; }
.single-card { width: min(520px, 100%); height: 340px; margin: 10px auto 0; display: flex; flex-direction: column; gap: 14px; }
.single-card .flip-card { flex: 1; height: auto; }
.study-buttons { gap: 8px; }
</style>
