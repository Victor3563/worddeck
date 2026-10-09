<template>
  <span class="spinner" :style="{ width: size + 'px', height: size + 'px' }" role="status" aria-label="Загрузка"></span>
</template>

<script setup>
defineProps({ size: { type: Number, default: 22 } })
</script>

<style>
.spinner { display: inline-block; border-radius: 50%; border: 2.5px solid rgba(167, 139, 250, 0.25); border-top-color: var(--violet-2); animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.page-loader { display: grid; place-items: center; padding: 80px 0; }
</style>
