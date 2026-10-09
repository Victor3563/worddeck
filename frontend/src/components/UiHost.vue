<template>
  <div class="toast-host" aria-live="polite">
    <TransitionGroup name="toast">
      <div v-for="t in ui.toasts" :key="t.id" class="toast" :class="t.type" @click="ui.dismiss(t.id)">
        <Icon :name="t.type === 'success' ? 'check-circle' : t.type === 'error' ? 'alert' : 'info'" />
        <span>{{ t.message }}</span>
      </div>
    </TransitionGroup>
  </div>

  <Modal :open="!!ui.confirmState" :title="ui.confirmState ? ui.confirmState.title : ''" width="440px" @close="ui.resolveConfirm(false)">
    <p v-if="ui.confirmState" class="muted confirm-text">{{ ui.confirmState.message }}</p>
    <template #footer>
      <button class="btn ghost" type="button" @click="ui.resolveConfirm(false)">Отмена</button>
      <button class="btn" :class="ui.confirmState && ui.confirmState.danger ? 'danger' : 'primary'" type="button" autofocus
              @click="ui.resolveConfirm(true)">
        {{ ui.confirmState ? ui.confirmState.confirmText : '' }}
      </button>
    </template>
  </Modal>
</template>

<script setup>
import { ui } from '../stores/ui.js'
</script>

<style>
.toast-host { position: fixed; right: 20px; bottom: 20px; z-index: 200; display: flex; flex-direction: column; gap: 10px; max-width: min(380px, calc(100vw - 40px)); }
.toast {
  display: flex; align-items: flex-start; gap: 10px; padding: 12px 14px; border-radius: 14px; cursor: pointer;
  background: #1d1948; border: 1px solid var(--border-strong); box-shadow: var(--shadow); font-size: 14px;
}
.toast.success { border-color: rgba(52, 211, 153, 0.45); }
.toast.success .icon { color: var(--success); }
.toast.error { border-color: rgba(251, 113, 133, 0.5); }
.toast.error .icon { color: var(--danger); }
.toast .icon { margin-top: 2px; color: var(--violet-2); }
.toast-enter-active, .toast-leave-active { transition: all 0.25s ease; }
.toast-enter-from { opacity: 0; transform: translateY(10px); }
.toast-leave-to { opacity: 0; transform: translateX(20px); }
.confirm-text { white-space: pre-line; }
@media (max-width: 640px) { .toast-host { left: 14px; right: 14px; bottom: 14px; max-width: none; } }
</style>
