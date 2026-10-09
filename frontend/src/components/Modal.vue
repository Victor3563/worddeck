<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-backdrop" @mousedown.self="onBackdrop">
        <div class="modal" :style="{ maxWidth: width }" role="dialog" aria-modal="true" :aria-label="title" ref="box">
          <header v-if="title || $slots.header" class="modal-head">
            <slot name="header"><h3>{{ title }}</h3></slot>
            <button class="btn ghost icon sm" type="button" @click="$emit('close')" aria-label="Закрыть">
              <Icon name="x" />
            </button>
          </header>
          <div class="modal-body"><slot /></div>
          <footer v-if="$slots.footer" class="modal-foot"><slot name="footer" /></footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script>
// Общий для всех модальных окон: прокрутка страницы заблокирована, пока открыта хотя бы одна.
let openCount = 0
</script>

<script setup>
import { watch, ref, nextTick, onBeforeUnmount } from 'vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '' },
  width: { type: String, default: '520px' },
  closeOnBackdrop: { type: Boolean, default: true },
})
const emit = defineEmits(['close'])

const box = ref(null)
let counted = false

const onKey = (e) => { if (e.key === 'Escape') emit('close') }
const onBackdrop = () => { if (props.closeOnBackdrop) emit('close') }

const lock = (on) => {
  if (on && !counted) { openCount++; counted = true; window.addEventListener('keydown', onKey) }
  if (!on && counted) { openCount--; counted = false; window.removeEventListener('keydown', onKey) }
  document.body.style.overflow = openCount > 0 ? 'hidden' : ''
}

watch(() => props.open, async (v) => {
  lock(v)
  if (v) {
    await nextTick()
    const el = box.value && box.value.querySelector('[autofocus], input, textarea, select')
    if (el) el.focus()
  }
}, { immediate: true })

onBeforeUnmount(() => lock(false))
</script>

<style>
.modal-backdrop {
  position: fixed; inset: 0; z-index: 100;
  display: flex; align-items: center; justify-content: center;
  padding: 20px;
  background: rgba(5, 4, 18, 0.72);
  backdrop-filter: blur(6px);
}
.modal {
  width: 100%;
  max-height: calc(100vh - 40px);
  display: flex; flex-direction: column;
  background: linear-gradient(180deg, #1b1745 0%, #141138 100%);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}
.modal-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 18px 20px 6px; }
.modal-body { padding: 14px 20px 20px; overflow-y: auto; }
.modal-foot { display: flex; justify-content: flex-end; gap: 10px; padding: 14px 20px; border-top: 1px solid var(--border); background: rgba(8, 6, 26, 0.35); flex-wrap: wrap; }
.modal-enter-active, .modal-leave-active { transition: opacity 0.2s ease; }
.modal-enter-active .modal, .modal-leave-active .modal { transition: transform 0.22s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
.modal-enter-from .modal { transform: translateY(12px) scale(0.98); }
.modal-leave-to .modal { transform: translateY(6px); }
@media (max-width: 640px) {
  .modal-backdrop { align-items: flex-end; padding: 0; }
  .modal { max-height: 92vh; border-radius: 22px 22px 0 0; }
}
</style>
