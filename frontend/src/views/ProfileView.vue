<template>
  <div v-if="auth.user" class="container page profile">
    <div class="page-head">
      <div>
        <h1>Профиль</h1>
        <p class="sub">Аккаунт создан {{ created }}</p>
      </div>
    </div>

    <div class="profile-grid">
      <form class="panel stack" @submit.prevent="saveName">
        <h3>Основное</h3>
        <div class="field">
          <label for="p-name">Имя</label>
          <input id="p-name" v-model="name" class="input" :maxlength="limits.display_name" />
        </div>
        <div class="field">
          <label>Email</label>
          <input class="input" :value="auth.user.email" disabled />
          <span class="hint">Email используется для входа и не меняется.</span>
        </div>
        <div><button class="btn primary" type="submit" :disabled="!name.trim() || name.trim() === auth.user.display_name">Сохранить</button></div>
      </form>

      <form class="panel stack" @submit.prevent="changePassword">
        <h3>Смена пароля</h3>
        <div class="field">
          <label for="p-cur">Текущий пароль</label>
          <input id="p-cur" v-model="pwd.current" type="password" class="input" autocomplete="current-password" />
        </div>
        <div class="field">
          <label for="p-new">Новый пароль</label>
          <input id="p-new" v-model="pwd.next" type="password" class="input" autocomplete="new-password" />
          <span class="hint">Минимум {{ limits.password_min }} символов</span>
        </div>
        <div v-if="pwd.error" class="form-error">{{ pwd.error }}</div>
        <div><button class="btn" type="submit" :disabled="!pwd.current || pwd.next.length < (limits.password_min || 1)">Сменить пароль</button></div>
      </form>

      <div class="panel stack">
        <h3>Сессии</h3>
        <p class="muted">Если вы входили с чужого устройства или потеряли его, завершите все сессии. Смена пароля завершает все сессии, кроме текущей.</p>
        <div><button class="btn" type="button" @click="logoutEverywhere"><Icon name="logout" :size="16" /> Выйти на всех устройствах</button></div>
      </div>

      <div class="panel stack danger-zone">
        <h3>Удаление аккаунта</h3>
        <p class="muted">Будут безвозвратно удалены все ваши области, папки, карточки, ссылки и статистика.</p>
        <div><button class="btn danger" type="button" @click="del.open = true"><Icon name="trash" :size="16" /> Удалить аккаунт</button></div>
      </div>
    </div>

    <Modal :open="del.open" title="Удалить аккаунт?" width="440px" @close="del.open = false">
      <form class="stack" id="del-form" @submit.prevent="deleteAccount">
        <p class="muted">Это действие нельзя отменить. Для подтверждения введите пароль.</p>
        <input v-model="del.password" type="password" class="input" placeholder="Пароль" autocomplete="current-password" autofocus />
        <div v-if="del.error" class="form-error">{{ del.error }}</div>
      </form>
      <template #footer>
        <button class="btn ghost" type="button" @click="del.open = false">Отмена</button>
        <button class="btn danger" type="submit" form="del-form" :disabled="!del.password">Удалить навсегда</button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'
import { auth } from '../stores/auth.js'
import { limits } from '../stores/limits.js'
import { ui } from '../stores/ui.js'

const router = useRouter()
const name = ref(auth.user ? auth.user.display_name : '')
const pwd = reactive({ current: '', next: '', error: '' })
const del = reactive({ open: false, password: '', error: '' })
// После выхода или удаления аккаунта user обнуляется ещё до ухода со страницы.
const created = computed(() => (auth.user ? new Date(auth.user.created_at).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' }) : ''))

const saveName = async () => {
  try {
    auth.user = await api.updateMe({ display_name: name.value.trim() })
    ui.success('Имя сохранено')
  } catch (e) { ui.error(e) }
}
const changePassword = async () => {
  pwd.error = ''
  try {
    await api.changePassword({ current_password: pwd.current, new_password: pwd.next })
    pwd.current = ''
    pwd.next = ''
    ui.success('Пароль изменён. Остальные устройства вышли из аккаунта.')
  } catch (e) { pwd.error = e.message }
}
const logoutEverywhere = async () => {
  if (!(await ui.confirm({ title: 'Выйти на всех устройствах?', message: 'Все сессии, включая текущую, будут завершены.', confirmText: 'Выйти везде' }))) return
  try {
    await auth.logoutEverywhere()
    ui.toast('Вы вышли на всех устройствах')
    router.push('/login')
  } catch (e) { ui.error(e) }
}
const deleteAccount = async () => {
  del.error = ''
  try {
    await api.deleteMe(del.password)
    auth.clear()
    ui.toast('Аккаунт удалён')
    router.push('/')
  } catch (e) { del.error = e.message }
}
</script>

<style>
.profile-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 16px; align-items: start; }
.danger-zone { border-color: rgba(251, 113, 133, 0.25); }
.input:disabled { opacity: 0.6; }
</style>
