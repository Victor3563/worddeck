// Использование: <div v-bind="press.handlers">…</div>
export const LONG_PRESS_MS = 450
const MOVE_TOLERANCE = 10

export function usePress({ onTap, onLongPress, delay = LONG_PRESS_MS } = {}) {
  let timer = null
  let startX = 0
  let startY = 0
  let active = false
  let longFired = false

  const clear = () => { if (timer) { clearTimeout(timer); timer = null } }

  const handlers = {
    onPointerdown(e) {
      if (e.button !== undefined && e.button !== 0) return
      if (e.target.closest && e.target.closest('[data-no-press]')) return
      active = true
      longFired = false
      startX = e.clientX
      startY = e.clientY
      clear()
      timer = setTimeout(() => {
        if (!active) return
        longFired = true
        if (navigator.vibrate) navigator.vibrate(15)
        onLongPress && onLongPress(e)
      }, delay)
    },
    onPointermove(e) {
      if (!active) return
      if (Math.abs(e.clientX - startX) > MOVE_TOLERANCE || Math.abs(e.clientY - startY) > MOVE_TOLERANCE) {
        active = false
        clear()
      }
    },
    onPointerup(e) {
      if (!active) return
      active = false
      clear()
      if (!longFired) onTap && onTap(e)
    },
    onPointercancel() { active = false; clear() },
    onPointerleave() { active = false; clear() },
    onContextmenu(e) { e.preventDefault() },
    onKeydown(e) {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onTap && onTap(e) }
      if (e.key.toLowerCase() === 'e' || e.key === 'ArrowUp') { e.preventDefault(); onLongPress && onLongPress(e) }
    },
  }
  return { handlers }
}
