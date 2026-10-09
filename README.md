# WordDeck

Веб-приложение для изучения английских слов по карточкам.

## Запуск

```bash
cp .env.example .env
docker compose up --build
```

Приложение откроется на http://localhost:8080 (порт задаётся в `FRONTEND_PORT`). Первая сборка занимает пару минут. Пример файла для импорта карточек: http://localhost:8080/sample-cards.csv.

Контейнеры запускаются по порядку: сначала `postgres`, потом `migrate` применяет миграции и завершается, после него стартуют `backend` и `frontend`.

Логи backend:

```bash
docker compose logs -f backend
```

Запуск нескольких копий backend:

```bash
docker compose up -d --scale backend=3
```

Миграции вручную: применить, посмотреть состояние, откатить последнюю.

```bash
docker compose run --rm migrate
docker compose run --rm migrate migrate status
docker compose run --rm migrate migrate down
```

Остановка. Данные сохраняются в томе `pgdata`, а с флагом `-v` удаляются.

```bash
docker compose down
docker compose down -v
```

## Запуск в minikube

Нужен рантайм docker, иначе `minikube docker-env` не работает

```bash
minikube start --driver=docker --container-runtime=docker --memory=3072 --cpus=4
eval $(minikube docker-env)
docker build -t worddeck-backend:1.0.0 --build-arg VERSION=1.0.0 backend
docker build -t worddeck-frontend:1.0.0 frontend
kubectl apply -k k8s
kubectl -n worddeck get pods
minikube service frontend -n worddeck --url
```

Образы, собранные локальным docker, можно загрузить в minikube так: `minikube image load worddeck-backend:1.0.0`.

Если `registry.npmjs.org` недоступен, frontend собирается через зеркало: `--build-arg NPM_REGISTRY=https://registry.npmmirror.com/`.

Масштабирование и удаление:

```bash
kubectl -n worddeck scale deployment backend --replicas=4
kubectl -n worddeck scale deployment frontend --replicas=3
kubectl delete -k k8s
```

## Проверки

Сквозная проверка API на запущенном стенде. С флагом `--rate-limit` дополнительно проверяется лимит попыток входа.

```bash
python3 scripts/smoke_test.py http://localhost:8080
python3 scripts/smoke_test.py http://localhost:8080 --rate-limit
```

Тесты и линтер backend. Тестам нужен Docker: они сами поднимают PostgreSQL. С флагом `-short` тесты с базой пропускаются.

```bash
cd backend
go test ./...
go test -short ./...
docker run --rm -v "$PWD":/src -w /src golangci/golangci-lint:v2.14.0 golangci-lint run
```

Линтер frontend (нужен Node 22):

```bash
cd frontend
npm run lint
```

Браузерный сценарий на Playwright описан в [scripts/e2e/README.md](scripts/e2e/README.md).

## Запуск без Docker

Нужен запущенный PostgreSQL.

```bash
cd backend
export DATABASE_URL="postgres://worddeck:worddeck@localhost:5432/worddeck?sslmode=disable"
export COOKIE_SECURE=false
go run ./cmd/server migrate
go run ./cmd/server serve
```

В другом терминале:

```bash
cd frontend
npm ci
npm run dev
```

## Как устроено

Браузер обращается только к nginx. Он отдаёт собранный frontend и пересылает запросы `/api` на backend. Поэтому backend можно вынести на другой сервер или запустить в нескольких копиях, а frontend об этом знать не нужно.

Backend разбит на слои, каждый вызывает только те, что ниже:

| Пакет | Что делает |
|---|---|
| `cmd/server` | точка входа: команды `serve` и `migrate`, конфигурация, логгер, сигналы |
| `internal/app` | собирает зависимости, запускает HTTP-сервер и корректно его останавливает |
| `internal/httpapi` | маршруты, middleware, разбор запросов и ответов, cookie |
| `internal/service` | бизнес-логика: проверки, сессии, права, транзакции, статистика |
| `internal/store` | SQL-запросы к PostgreSQL |
| `internal/domain` | сущности, ограничения и ошибки |

Вспомогательные пакеты: `config` читает переменные окружения, `logger` пишет логи в stdout, `db` подключается к базе и выполняет миграции, `auth` хэширует пароли и создаёт токены сессий, `cardio` разбирает файлы импорта и формирует экспорт, `testdb` поднимает PostgreSQL для тестов.

После входа сервер ставит cookie `wd_session`. В ней лежит случайный токен, а в таблице `user_sessions` хранится только его хэш. Выход завершает текущую сессию, «выйти на всех устройствах» завершает все, а смена пароля — все, кроме текущей.

Ошибки API возвращаются кодом, например `{"error": {"code": "too_long", "field": "name", "params": {"max": 100}}}`. Статус 422 означает ошибку проверки данных. Текст для пользователя frontend подбирает по коду. Ограничения на длину полей и размер импорта frontend получает из `GET /api/limits`, поэтому на сервере и в формах одни и те же числа.

## Переменные окружения

Backend:

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `DATABASE_URL` | обязательна | строка подключения к PostgreSQL |
| `PORT` | `8080` | порт HTTP-сервера |
| `SESSION_TTL` | `720h` | срок жизни сессии |
| `COOKIE_SECURE` | `true` | флаг `Secure` у cookie сессии; `false` только для локального HTTP |
| `LOG_LEVEL` / `LOG_FORMAT` | `info` / `json` | уровень и формат логов |
| `SHUTDOWN_TIMEOUT` | `15s` | сколько ждать завершения запросов при остановке |
| `HTTP_READ_HEADER_TIMEOUT` / `HTTP_READ_TIMEOUT` | `5s` / `30s` | таймауты чтения заголовков и запроса |
| `HTTP_WRITE_TIMEOUT` / `HTTP_IDLE_TIMEOUT` | `60s` / `120s` | таймаут ответа и простоя соединения |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` | `10` / `5` | размер пула соединений |
| `DB_CONN_MAX_IDLE_TIME` / `DB_CONN_MAX_LIFETIME` | `5m` / `30m` | время жизни соединений в пуле |
| `DB_CONNECT_TIMEOUT` | `30s` | сколько ждать базу при старте |
| `MAX_IMPORT_BYTES` | `1048576` | максимальный размер файла импорта |

Frontend (nginx):

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `BACKEND_UPSTREAM` | `backend:8080` | адрес backend |
| `NGINX_CLIENT_MAX_BODY_SIZE` | `2m` | максимальный размер запроса, должен быть больше `MAX_IMPORT_BYTES` |
| `NGINX_AUTH_RATE` / `NGINX_AUTH_BURST` | `10r/m` / `20` | лимит попыток входа и регистрации с одного IP |
