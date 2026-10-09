import { reactive } from 'vue'

let seq = 0

export const ui = reactive({
  toasts: [],
  confirmState: null,

  toast(message, type = 'info', timeout = 3200) {
    const id = ++seq
    this.toasts.push({ id, message, type })
    setTimeout(() => this.dismiss(id), timeout)
  },
  success(m) { this.toast(m, 'success') },
  error(m) { this.toast(m instanceof Error ? m.message : m, 'error', 4500) },
  dismiss(id) {
    const i = this.toasts.findIndex(t => t.id === id)
    if (i >= 0) this.toasts.splice(i, 1)
  },

  /** confirm({ title, message, confirmText, danger }) → Promise<boolean> */
  confirm(opts) {
    return new Promise(resolve => {
      this.confirmState = { confirmText: 'Подтвердить', ...opts, resolve }
    })
  },
  resolveConfirm(value) {
    if (this.confirmState) this.confirmState.resolve(value)
    this.confirmState = null
  },
})

export async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    ta.remove()
  }
  ui.success('Ссылка скопирована')
}
