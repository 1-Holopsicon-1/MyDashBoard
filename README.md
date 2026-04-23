# !!! DISCLAIMER !!!
## Да параноиков шизов и дедов которые "аа вайбкодеры атакуют так плохо ыы, хочу мидла за 50к без вайбкода с обязаностями сеньёра" - да в данном проекте присутствует много элементов вайб кодинга, если вы такое не приемлите, немедленно закройти данный репозиторий!

Для тех кому интересно что и почему вот:

Данный репо мне нужен был для мониторинга своих сервисов и для такой простой задачи решил я поэксперементировать и посмотреть на что способны текущие модели китайские в сравнению с лидером (имхо) Claude Opus 4.6 (на момент написанная данного репо это было последняя версия)

Эксперемент был на 4 моделях:
- Claude Opus 4.6 (Накидал базовый функционал на который я ориентировался и сравнивал)
- MinMax M2.5 (Код ушёл в топку мне не понравился - ни как он писал, ни что выбирал, ни скорость)
- Qwen3.5-Qwen3-coder-next (Впринципе резульат меня устроил, но у QwenCode больше нет бесплатной версии пришлось костылить через Ollama, а там лимиты быстро вгонялись)
- Z.ai GLM 5.1 и в данной версии репо представлен именно его код. Мне более чем понравилася как он организован, меньше правок, думающий режим конено медлительный особенно на фоне с Opus но за туже цену с 2х лимитом меня более чем устроил. Но в плане удобства вне opencode/claude code - качество оставляет желать лучшего.

Вот такой эксперимент. 

Там местами захардкожен мой домен, но если вам сильно нужен можете спокойно брать и для себя разворачивать заменив имена

---
# holopsicon.ru Dashboard

Мониторинг-дашборд для сети Tailscale, Docker-контейнеров и проверки сервисов. Авторизация через Passkey (WebAuthn), регистрация при первом запуске.

## Возможности

- Мониторинг устройств Tailscale (IP скрыты до авторизации)
- Статус Docker-контейнеров
- Проверка доступности сервисов
- Извлечение ссылок SimpleX из логов контейнеров
- Авторизация по Passkey/WebAuthn (поддержка нескольких ключей)
- Три темы: terminal, minimal, glass

## Стек

- **Backend**: Go + [chi](https://github.com/go-chi/chi) router
- **Frontend**: SvelteKit 5 (runes mode) + TypeScript
- **Auth**: WebAuthn через [go-webauthn](https://github.com/go-webauthn/webauthn), хранение в SQLite
- **Деплой**: Docker multi-stage + Caddy reverse proxy, GitHub Actions CI/CD

## Быстрый старт

### Docker (продакшен)

```bash
cp .env.example .env
# Заполните .env своими значениями (укажите WEBAUTHN_RP_ID и WEBAUTHN_ORIGIN — ваш домен)
docker compose up --build -d
```

Дашборд доступен на `http://localhost:8090`.

Первый визит: нажмите на название сайта 5 раз в течение 10 секунд — появится панель регистрации. Зарегистрируйте passkey, затем авторизуйтесь для доступа к полным данным включая IP устройств.

### Локальная разработка

**Backend:**

```bash
cd backend
cp .env.example .env
# Заполните .env
go run ./cmd/server
```

**Frontend:**

```bash
cd frontend
yarn install
yarn dev
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `LISTEN_ADDR` | `:8081` | Адрес сервера |
| `TAILSCALE_API_KEY` | — | API-ключ Tailscale |
| `TAILSCALE_TAILNET` | — | ID tailnet'а Tailscale |
| `VAULTWARDEN_URL` | — | URL Vaultwarden для проверки |
| `DOCKER_SOCKET` | `/var/run/docker.sock` | Путь к Docker-сокету |
| `CONTAINER_FILTERS` | `amnezia,simplex` | Фильтры контейнеров (через запятую) |
| `SESSION_SECRET` | автогенерация | HMAC-ключ для подписи сессионных cookie |
| `WEBAUTHN_RP_ID` | `localhost` | WebAuthn Relying Party ID (ваш домен) |
| `WEBAUTHN_ORIGIN` | `http://localhost:5173` | WebAuthn origin URL |
| `DB_PATH` | `dashboard.db` | Путь к файлу SQLite |

### Для Docker-деплоя

Укажите в `.env`:
- `WEBAUTHN_RP_ID=holopsicon.ru`
- `WEBAUTHN_ORIGIN=https://holopsicon.ru`
- `DB_PATH=/app/data/dashboard.db`

## Авторизация

Однопользовательская авторизация через passkey. При первом запуске панель регистрации скрыта — активируйте её, нажав на название сайта 5 раз за 10 секунд. После входа можно добавить дополнительные ключи через кнопку "Add Key".

- Credentials WebAuthn хранятся в SQLite (персистентность через Docker volume)
- Сессии — HMAC-подписанные cookie (7 дней, HttpOnly, Secure, SameSite=Strict)
- IP устройств Tailscale видны только авторизованным пользователям

## Структура проекта

```
backend/
  cmd/server/          # Точка входа + DI
  internal/
    handler/           # HTTP-обработчики (chi-маршруты)
    service/           # Бизнес-логика
    client/            # Клиенты внешних API
    model/             # Модели данных
    middleware/         # HTTP-middleware
    auth/              # WebAuthn + сессии + SQLite-хранилище
  .env.example         # Шаблон для локальной разработки

frontend/
  src/
    lib/
      components/      # Svelte-компоненты
      themes/          # CSS-темы
      stores.ts        # Svelte-сторы
      api.ts           # API-клиент
      types.ts         # TypeScript-типы
    routes/            # Страницы SvelteKit

.env.example           # Шаблон для Docker-деплоя
docker-compose.yml
deploy/Caddyfile - (у меня нативная версия на сервере чтобы не моричится с реверсом в контейнер через нжинкс поэтому, если вам надо то добавте в контейнер сами)

Dockerfile
.github/workflows/     # CI/CD
```
