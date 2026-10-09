// API на том же origin (/api проксирует nginx): адрес backend в бандле не нужен,
// cookie сессии браузер отправляет сам.
import { errorText } from './errors.js'

export class ApiError extends Error {
  constructor(status, body) {
    const err = (body && body.error) || {}
    super(errorText(err, status))
    this.status = status
    this.code = err.code || `http_${status}`
    this.field = err.field || null
    this.params = err.params || {}
  }
}

let onUnauthenticated = null
export function setUnauthenticatedHandler(fn) { onUnauthenticated = fn }

async function request(method, path, { body, form, query, raw } = {}) {
  let url = '/api' + path
  if (query) {
    const qs = new URLSearchParams()
    Object.entries(query).forEach(([k, v]) => { if (v !== undefined && v !== null && v !== '') qs.set(k, v) })
    const s = qs.toString()
    if (s) url += '?' + s
  }
  const headers = {}
  let payload
  if (form) payload = form
  else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }

  let res
  try {
    res = await fetch(url, { method, headers, body: payload, credentials: 'same-origin' })
  } catch {
    throw new ApiError(0, { error: { code: 'network' } })
  }

  if (raw && res.ok) return res
  const ct = res.headers.get('content-type') || ''
  const data = res.status === 204 ? null : ct.includes('application/json') ? await res.json().catch(() => null) : await res.text()

  if (!res.ok) {
    if (res.status === 401 && !path.startsWith('/auth/') && onUnauthenticated) onUnauthenticated()
    throw new ApiError(res.status, typeof data === 'object' ? data : null)
  }
  return data
}

export const api = {
  limits: () => request('GET', '/limits'),

  register: (d) => request('POST', '/auth/register', { body: d }),
  login: (d) => request('POST', '/auth/login', { body: d }),
  logout: () => request('POST', '/auth/logout'),
  logoutAll: () => request('POST', '/auth/logout-all'),
  me: () => request('GET', '/me'),
  updateMe: (d) => request('PATCH', '/me', { body: d }),
  changePassword: (d) => request('PUT', '/me/password', { body: d }),
  deleteMe: (password) => request('DELETE', '/me', { body: { password } }),
  stats: () => request('GET', '/stats', { query: { tz: Intl.DateTimeFormat().resolvedOptions().timeZone } }),

  workspaces: () => request('GET', '/workspaces'),
  workspace: (id) => request('GET', `/workspaces/${id}`),
  createWorkspace: (d) => request('POST', '/workspaces', { body: d }),
  updateWorkspace: (id, d) => request('PATCH', `/workspaces/${id}`, { body: d }),
  deleteWorkspace: (id) => request('DELETE', `/workspaces/${id}`),
  workspaceCards: (id) => request('GET', `/workspaces/${id}/cards`),

  folders: (wsId) => request('GET', `/workspaces/${wsId}/folders`),
  folder: (id) => request('GET', `/folders/${id}`),
  createFolder: (wsId, d) => request('POST', `/workspaces/${wsId}/folders`, { body: d }),
  updateFolder: (id, d) => request('PATCH', `/folders/${id}`, { body: d }),
  deleteFolder: (id) => request('DELETE', `/folders/${id}`),

  cards: (folderId, q) => request('GET', `/folders/${folderId}/cards`, { query: { q } }),
  createCard: (folderId, d) => request('POST', `/folders/${folderId}/cards`, { body: d }),
  createCards: (folderId, cards) => request('POST', `/folders/${folderId}/cards/bulk`, { body: { cards } }),
  deleteCards: (folderId, ids) => request('POST', `/folders/${folderId}/cards/delete`, { body: { ids } }),
  updateCard: (id, d) => request('PATCH', `/cards/${id}`, { body: d }),
  deleteCard: (id) => request('DELETE', `/cards/${id}`),
  importCards: (folderId, { file, text }, dryRun) => {
    const form = new FormData()
    if (file) form.append('file', file)
    else form.append('text', text || '')
    return request('POST', `/folders/${folderId}/import`, { form, query: { dry_run: dryRun ? 'true' : undefined } })
  },
  exportCards: (folderId, format) => request('GET', `/folders/${folderId}/export`, { query: { format }, raw: true }),

  shares: (filter) => request('GET', '/shares', { query: filter }),
  createShare: (resource_type, resource_id) => request('POST', '/shares', { body: { resource_type, resource_id } }),
  deleteShare: (token) => request('DELETE', `/shares/${encodeURIComponent(token)}`),
  publicShare: (token) => request('GET', `/public/shares/${encodeURIComponent(token)}`),
  copyShare: (token, target_id) => request('POST', `/shares/${encodeURIComponent(token)}/copy`, { body: { target_id } }),

  sessions: (limit, offset) => request('GET', '/study-sessions', { query: { limit, offset } }),
  saveSession: (d) => request('POST', '/study-sessions', { body: d }),
  deleteSession: (id) => request('DELETE', `/study-sessions/${id}`),
  clearSessions: () => request('DELETE', '/study-sessions'),
}

export function shareUrl(token) {
  return `${window.location.origin}/s/${token}`
}
