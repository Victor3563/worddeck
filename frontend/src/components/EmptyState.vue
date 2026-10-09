<template>
  <div class="empty-state">
    <div class="empty-icon"><Icon :name="icon" :size="30" /></div>
    <h3>{{ title }}</h3>
    <p v-if="text" class="muted">{{ text }}</p>
    <div v-if="$slots.default" class="empty-actions"><slot /></div>
  </div>
</template>

<script setup>
defineProps({
  icon: { type: String, default: 'sparkles' },
  title: { type: String, required: true },
  text: { type: String, default: '' },
})
</script>

<style>
.empty-state { display: flex; flex-direction: column; align-items: center; text-align: center; gap: 10px; padding: 48px 20px; border: 1px dashed var(--border-strong); border-radius: var(--radius-lg); background: rgba(255, 255, 255, 0.02); }
.empty-icon { width: 64px; height: 64px; display: grid; place-items: center; border-radius: 20px; background: var(--grad-soft); color: #c4b5fd; margin-bottom: 4px; }
.empty-state p { max-width: 420px; }
.empty-actions { display: flex; gap: 10px; margin-top: 8px; flex-wrap: wrap; justify-content: center; }
</style>
