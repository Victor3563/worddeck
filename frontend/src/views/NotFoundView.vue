<template>
  <div class="container page">
    <EmptyState icon="alert" title="Страница не найдена" text="Такой страницы нет — возможно, ссылка устарела.">
      <RouterLink :to="auth.loggedIn ? '/app' : '/'" class="btn primary"><Icon name="home" :size="16" /> На главную</RouterLink>
    </EmptyState>
  </div>
</template>

<script setup>
import { auth } from '../stores/auth.js'
</script>
