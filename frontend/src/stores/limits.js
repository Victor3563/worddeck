import { reactive } from 'vue'
import { api } from '../api.js'

export const limits = reactive({})

export async function loadLimits() {
  try {
    Object.assign(limits, await api.limits())
  } catch {
    // Без лимитов формы лишь теряют подсказки, сервер всё равно проверяет.
  }
}

export function formatBytes(n) {
  if (!n) return ''
  return n >= 1 << 20 ? `${+(n / (1 << 20)).toFixed(1)} МБ` : `${Math.round(n / 1024)} КБ`
}
