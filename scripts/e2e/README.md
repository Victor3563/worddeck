# Браузерный сценарий (Playwright)

`walkthrough.js` проходит по приложению в настоящем браузере Chromium. Сценарий проверяет регистрацию с ошибкой в форме, все страницы, три режима обучения, импорт с ошибками в строках, выход на всех устройствах, вход с неверным паролем и удаление аккаунта. Если в консоли браузера появится неожиданная ошибка, тест падает.

Запускается на поднятом `docker compose`. Браузер берётся из образа Playwright той же версии, что указана в `package.json`.

```bash
docker run --rm --network host -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$PWD/scripts/e2e":/e2e -w /e2e \
  mcr.microsoft.com/playwright:v1.56.1-noble \
  sh -c "npm ci --no-audit --no-fund && node walkthrough.js http://localhost:8080"
```

Сценарий делает 4 попытки входа и регистрации, это укладывается в лимит nginx. Если перед ним запускали `smoke_test.py --rate-limit`, подождите минуту.
