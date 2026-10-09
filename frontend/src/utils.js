export function shuffle(arr) {
  const a = arr.slice()
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[a[i], a[j]] = [a[j], a[i]]
  }
  return a
}

// Не на позицию 0, чтобы незнакомая карточка не появилась сразу снова.
export function insertRandom(arr, item) {
  const min = arr.length > 0 ? 1 : 0
  const pos = min + Math.floor(Math.random() * (arr.length - min + 1))
  arr.splice(pos, 0, item)
}

// Без учёта регистра, ё/е, лишних пробелов и пунктуации по краям.
export function normalizeAnswer(s) {
  return String(s || '')
    .toLowerCase()
    .replace(/ё/g, 'е')
    .replace(/[’`´]/g, "'")
    .replace(/\s+/g, ' ')
    .replace(/^[\s.,!?;:"«»()-]+|[\s.,!?;:"«»()]+$/g, '')
    .trim()
}

// "house, home / dwelling" → 3 варианта; "to " у английских глаголов необязателен.
export function answerVariants(target) {
  const parts = String(target || '').split(/[,;/]/).map(normalizeAnswer).filter(Boolean)
  const out = new Set(parts)
  parts.forEach(p => { if (p.startsWith('to ')) out.add(p.slice(3)) })
  if (out.size === 0) out.add(normalizeAnswer(target))
  return [...out]
}

export function isCorrectAnswer(input, target) {
  const n = normalizeAnswer(input)
  if (!n) return false
  const variants = answerVariants(target)
  return variants.includes(n) || variants.includes(n.replace(/^to /, ''))
}

// Посимвольный diff (LCS): [{ ch, type: 'ok'|'missing'|'extra' }]
export function diffChars(input, target) {
  const a = [...String(input || '')]
  const b = [...String(target || '')]
  const m = a.length, n = b.length
  const dp = Array.from({ length: m + 1 }, () => new Array(n + 1).fill(0))
  for (let i = m - 1; i >= 0; i--)
    for (let j = n - 1; j >= 0; j--)
      dp[i][j] = a[i].toLowerCase() === b[j].toLowerCase() ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1])
  const out = []
  let i = 0, j = 0
  while (i < m && j < n) {
    if (a[i].toLowerCase() === b[j].toLowerCase()) { out.push({ ch: b[j], type: 'ok' }); i++; j++ }
    else if (dp[i + 1][j] >= dp[i][j + 1]) { out.push({ ch: a[i], type: 'extra' }); i++ }
    else { out.push({ ch: b[j], type: 'missing' }); j++ }
  }
  while (i < m) out.push({ ch: a[i++], type: 'extra' })
  while (j < n) out.push({ ch: b[j++], type: 'missing' })
  return out
}

export function formatDuration(ms) {
  const s = Math.max(0, Math.round(ms / 1000))
  const m = Math.floor(s / 60)
  const r = s % 60
  return m ? `${m} мин ${r.toString().padStart(2, '0')} с` : `${r} с`
}

export function plural(n, one, few, many) {
  const n10 = n % 10, n100 = n % 100
  if (n10 === 1 && n100 !== 11) return one
  if (n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14)) return few
  return many
}

export function cardsWord(n) { return `${n} ${plural(n, 'карточка', 'карточки', 'карточек')}` }

export function formatDate(iso) {
  const d = new Date(iso)
  return d.toLocaleString('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}

export const MODE_LABELS = { learn: 'Учить', dictation: 'Диктант', match: 'Сопоставление' }

export const WORKSPACE_COLORS = {
  violet: '#8b5cf6', indigo: '#6366f1', blue: '#3b82f6', cyan: '#06b6d4',
  magenta: '#d946ef', emerald: '#10b981', amber: '#f59e0b', rose: '#f43f5e',
}
