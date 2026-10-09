<template>
  <div class="container page">
    <div class="page-head">
      <div>
        <h1>Мои ссылки</h1>
        <p class="sub">Всё, чем вы поделились. Отзовите ссылку, и она перестанет работать у всех.</p>
      </div>
    </div>

    <div v-if="loading" class="page-loader"><Spinner :size="30" /></div>
    <EmptyState v-else-if="!links.length" icon="link" title="Вы ещё ничем не делились"
                text="Откройте область, папку или карточку и нажмите «Поделиться» — появится ссылка, которую можно отправить кому угодно.">
      <RouterLink to="/app" class="btn primary">К моим областям</RouterLink>
    </EmptyState>
    <div v-else class="panel tight">
      <div class="list">
        <div v-for="l in links" :key="l.token" class="list-item share-row">
          <span class="share-icon"><Icon :name="iconOf(l.resource_type)" :size="18" /></span>
          <div class="grow">
            <RouterLink :to="linkTo(l)" class="strong share-name">{{ l.resource_name }}</RouterLink>
            <div class="muted small">{{ typeOf(l.resource_type) }} · создана {{ formatDate(l.created_at) }}</div>
          </div>
          <code class="share-token hide-sm">/s/{{ l.token }}</code>
          <button class="btn sm" type="button" @click="copyText(shareUrl(l.token))"><Icon name="copy" :size="14" /> <span class="hide-sm">Копировать</span></button>
          <a class="btn sm icon" :href="shareUrl(l.token)" target="_blank" rel="noopener" title="Открыть"><Icon name="external" :size="14" /></a>
          <button class="btn sm icon danger" type="button" title="Отозвать" @click="revoke(l)"><Icon name="trash" :size="14" /></button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { api, shareUrl } from '../api.js'
import { ui, copyText } from '../stores/ui.js'
import { formatDate } from '../utils.js'

const links = ref([])
const loading = ref(true)
onMounted(async () => {
  try { links.value = await api.shares() } catch (e) { ui.error(e) } finally { loading.value = false }
})
const iconOf = (t) => ({ workspace: 'layers', folder: 'folder', card: 'cards' }[t])
const typeOf = (t) => ({ workspace: 'Область', folder: 'Папка', card: 'Карточка' }[t])
const linkTo = (l) => (l.resource_type === 'workspace' ? `/workspaces/${l.resource_id}` : l.resource_type === 'folder' ? `/folders/${l.resource_id}` : `/s/${l.token}`)
const revoke = async (l) => {
  const ok = await ui.confirm({ title: 'Отозвать ссылку?', message: `Ссылка на «${l.resource_name}» перестанет работать.`, confirmText: 'Отозвать', danger: true })
  if (!ok) return
  try {
    await api.deleteShare(l.token)
    links.value = links.value.filter(x => x.token !== l.token)
    ui.success('Ссылка отозвана')
  } catch (e) { ui.error(e) }
}
</script>

<style>
.share-row { gap: 10px; }
.share-name { color: var(--text); }
.share-name:hover { color: #c4b5fd; }
.share-token { font-size: 12px; color: var(--muted); background: var(--surface); padding: 4px 8px; border-radius: 8px; border: 1px solid var(--border); }
</style>
