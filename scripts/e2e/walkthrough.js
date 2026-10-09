const { chromium } = require('playwright')

const BASE = (process.argv[2] || 'http://localhost:8080').replace(/\/$/, '')

const check = (cond, msg) => { if (!cond) throw new Error('CHECK FAILED: ' + msg) }

;(async () => {
  const browser = await chromium.launch()
  const page = await (await browser.newContext()).newPage()
  const problems = []
  page.on('console', (m) => { if (m.type() === 'error' || m.type() === 'warning') problems.push(`[${m.type()}] ${page.url()} ${m.text()}`) })
  page.on('pageerror', (e) => problems.push(`[pageerror] ${page.url()} ${e.message}`))
  const step = (s) => console.log('·', s)
  const atPath = (p) => (url) => new URL(url).pathname === p
  const loginOK = async () => {
    try { await page.waitForURL(atPath('/app'), { timeout: 10000 }) } catch (e) {
      const msg = await page.locator('.form-error').textContent().catch(() => '')
      throw new Error('not logged in, form error: ' + msg)
    }
  }

  const email = `e2e${Date.now()}@example.com`
  step('register: limits come from GET /api/limits')
  await page.goto(BASE + '/register')
  const limits = await (await page.request.get(BASE + '/api/limits')).json()
  check(await page.getAttribute('#name', 'maxlength') === String(limits.display_name), 'maxlength of the name field')
  check((await page.textContent('.hint')).includes(`Минимум ${limits.password_min} символов`), 'password hint')

  step('register: validation text comes from the client dictionary')
  await page.fill('#name', 'E2E Tester')
  await page.fill('#email', email)
  await page.fill('#password', 'short')
  await page.click('button[type=submit]')
  await page.waitForSelector('.form-error')
  const err = await page.textContent('.form-error')
  check(err.includes(`Пароль должен быть не короче ${limits.password_min} символов`), 'validation text: ' + err)

  step('register: success, cookie hidden from JS, session survives reload')
  await page.fill('#password', 'password123')
  await page.click('button[type=submit]')
  await loginOK()
  await page.waitForSelector('text=Моя область')
  check(!(await page.evaluate(() => document.cookie)).includes('wd_session'), 'session cookie visible to JavaScript')
  check(!(await page.evaluate(() => JSON.stringify(localStorage))).includes('token'), 'token in localStorage')
  await page.reload()
  await page.waitForSelector('text=Моя область')

  step('seed data through the API with the same cookie')
  const api = async (method, path, data) => {
    const r = await page.request.fetch(BASE + '/api' + path, { method, data })
    check(r.ok(), `${method} ${path} -> ${r.status()}`)
    return r.status() === 204 ? null : r.json()
  }
  const [ws] = await api('GET', '/workspaces')
  const f = await api('POST', `/workspaces/${ws.id}/folders`, { name: 'E2E папка' })
  await api('POST', `/folders/${f.id}/cards/bulk`, { cards: [
    { ru_word: 'дом', en_word: 'house' }, { ru_word: 'кот', en_word: 'cat' }, { ru_word: 'окно', en_word: 'window' }] })
  await api('POST', '/study-sessions', { mode: 'learn', folder_id: f.id, title: 'E2E', total_cards: 3, mistakes: 1, rounds: 1, duration_ms: 60000 })
  const share = await api('POST', '/shares', { resource_type: 'folder', resource_id: f.id })

  const pages = [
    ['/app', 'Моя область'], [`/workspaces/${ws.id}`, 'E2E папка'], [`/folders/${f.id}`, 'house'],
    ['/stats', null], ['/shares', 'E2E папка'], ['/profile', 'Выйти на всех устройствах'],
    [`/s/${share.token}`, 'house'], ['/nope', null],
  ]
  for (const [path, text] of pages) {
    step('page ' + path)
    await page.goto(BASE + path)
    await page.waitForLoadState('networkidle')
    if (text) await page.waitForSelector(`text=${text}`, { timeout: 5000 })
  }

  for (const mode of ['learn', 'dictation', 'match']) {
    step('study ' + mode)
    await page.goto(`${BASE}/study/${mode}?source=folder&id=${f.id}`)
    await page.click('button:has-text("Начать")')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(300)
  }

  step('import preview shows translated line errors')
  await page.goto(`${BASE}/folders/${f.id}`)
  await page.click('button:has-text("Импорт")')
  await page.click('button:has-text("Вставить текст")')
  await page.fill('.modal textarea', 'солнце; sun\nодно поле\n')
  await page.click('.modal button:has-text("Проверить")')
  await page.waitForSelector('text=Строка 2:')
  const preview = await page.textContent('.preview-errors')
  check(preview.includes('Ожидалось 2 колонки'), 'line error text: ' + preview)
  await page.click('.modal button:has-text("Импортировать")')
  await page.waitForSelector('text=sun')

  step('logout everywhere from the profile')
  await page.goto(BASE + '/profile')
  await page.click('button:has-text("Выйти на всех устройствах")')
  await page.click('.modal button:has-text("Выйти везде")')
  await page.waitForURL(atPath('/login'))
  await page.goto(BASE + '/app')
  await page.waitForURL(atPath('/login'))

  step('wrong password shows the dictionary text')
  await page.fill('#email', email)
  await page.fill('#password', 'wrong-password')
  await page.click('button[type=submit]')
  await page.waitForSelector('.form-error')
  check((await page.textContent('.form-error')).includes('Неверный email или пароль'), 'invalid credentials text')

  step('login again and delete the account from the profile')
  await page.fill('#password', 'password123')
  await page.click('button[type=submit]')
  await loginOK()
  await page.goto(BASE + '/profile')
  await page.click('button:has-text("Удалить аккаунт")')
  await page.fill('.modal input[type=password]', 'password123')
  await page.click('.modal button:has-text("Удалить навсегда")')
  await page.waitForURL(atPath('/'))
  const me = await page.request.get(BASE + '/api/me')
  check(me.status() === 401, 'session after account deletion: ' + me.status())

  await browser.close()
  const expected = [/401 \(Unauthorized\)/, /\/register Failed to load resource.*422/] // намеренно вызванные выше ошибки
  const real = problems.filter((p) => !expected.some((re) => re.test(p)))
  if (real.length) { console.log('PROBLEMS:\n' + real.join('\n')); process.exit(1) }
  console.log('E2E OK')
})().catch((e) => { console.error(e.message); process.exit(1) })
