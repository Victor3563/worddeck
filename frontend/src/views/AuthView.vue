<template>
  <div class="auth-page container">
    <form class="auth-card panel fade-up" @submit.prevent="submit" novalidate>
      <div class="auth-head">
        <img src="/favicon.svg" alt="" width="44" height="44" />
        <h1>{{ isLogin ? 'С возвращением' : 'Создать аккаунт' }}</h1>
        <p class="muted">{{ isLogin ? 'Войдите, чтобы продолжить обучение' : 'Пара секунд — и можно добавлять слова' }}</p>
      </div>

      <div v-if="error" class="form-error">{{ error }}</div>

      <div v-if="!isLogin" class="field">
        <label for="name">Имя</label>
        <input id="name" v-model="form.display_name" class="input" :class="{ invalid: field === 'display_name' }"
               autocomplete="name" :maxlength="limits.display_name" placeholder="Как к вам обращаться" />
      </div>
      <div class="field">
        <label for="email">Email</label>
        <input id="email" v-model="form.email" type="email" class="input" :class="{ invalid: field === 'email' }"
               autocomplete="email" placeholder="you@example.com" autofocus />
      </div>
      <div class="field">
        <label for="password">Пароль</label>
        <div class="password-wrap">
          <input id="password" v-model="form.password" :type="showPwd ? 'text' : 'password'" class="input"
                 :class="{ invalid: field === 'password' }" :autocomplete="isLogin ? 'current-password' : 'new-password'"
                 :placeholder="isLogin ? 'Ваш пароль' : `Не короче ${limits.password_min} символов`" />
          <button type="button" class="btn ghost icon sm pwd-toggle" @click="showPwd = !showPwd" :aria-label="showPwd ? 'Скрыть пароль' : 'Показать пароль'">
            <Icon name="eye" :size="16" />
          </button>
        </div>
        <span v-if="!isLogin" class="hint">Минимум {{ limits.password_min }} символов</span>
      </div>

      <button class="btn primary lg block" type="submit" :disabled="busy">
        <Spinner v-if="busy" :size="18" />
        {{ isLogin ? 'Войти' : 'Зарегистрироваться' }}
      </button>

      <p class="center muted small">
        <template v-if="isLogin">Нет аккаунта? <RouterLink :to="{ name: 'register', query: $route.query }">Зарегистрируйтесь</RouterLink></template>
        <template v-else>Уже есть аккаунт? <RouterLink :to="{ name: 'login', query: $route.query }">Войдите</RouterLink></template>
      </p>
    </form>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { auth } from '../stores/auth.js'
import { limits } from '../stores/limits.js'
import { ui } from '../stores/ui.js'

const props = defineProps({ mode: { type: String, default: 'login' } })

const route = useRoute()
const router = useRouter()
const isLogin = computed(() => props.mode === 'login')
const form = reactive({ email: '', password: '', display_name: '' })
const error = ref('')
const field = ref('')
const busy = ref(false)
const showPwd = ref(false)

watch(() => props.mode, () => { error.value = ''; field.value = '' })

const submit = async () => {
  error.value = ''
  field.value = ''
  if (!form.email.trim() || !form.password) {
    error.value = 'Заполните email и пароль'
    return
  }
  if (!isLogin.value && !form.display_name.trim()) {
    error.value = 'Укажите имя'
    field.value = 'display_name'
    return
  }
  busy.value = true
  try {
    if (isLogin.value) await auth.login(form.email.trim(), form.password)
    else await auth.register(form.email.trim(), form.display_name.trim(), form.password)
    ui.success(isLogin.value ? `Привет, ${auth.user.display_name}!` : 'Аккаунт создан — добро пожаловать!')
    const r = route.query.redirect
    router.replace(typeof r === 'string' && r.startsWith('/') ? r : '/app')
  } catch (e) {
    error.value = e.message
    field.value = e.field || ''
  } finally {
    busy.value = false
  }
}
</script>

<style>
.auth-page { min-height: calc(100vh - var(--header-h)); display: grid; place-items: center; padding-top: 30px; padding-bottom: 60px; }
.auth-card { width: 100%; max-width: 420px; display: flex; flex-direction: column; gap: 16px; padding: 30px; box-shadow: var(--shadow-lg); }
.auth-head { display: flex; flex-direction: column; align-items: center; text-align: center; gap: 8px; margin-bottom: 6px; }
.auth-head h1 { font-size: 26px; }
.password-wrap { position: relative; }
.password-wrap .input { padding-right: 46px; }
.pwd-toggle { position: absolute; right: 6px; top: 6px; }
</style>
