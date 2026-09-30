# URL Shortener

HTTP-сервис для создания коротких ссылок, перенаправления по alias, обновления и удаления ссылок. Данные хранятся в PostgreSQL.

## Запуск через Docker Compose

Нужны Docker и Docker Compose.

```sh
cp .env.example .env
docker compose up --build
```

Compose поднимет PostgreSQL, дождется его готовности, применит миграции и запустит API на `http://localhost:8082`. Данные PostgreSQL сохраняются в Docker volume `pgdata`.

Остановить сервисы можно командой `docker compose down`. Чтобы удалить также базу данных `docker compose down -v`.


## API

- `POST /url` — сохранить ссылку. JSON: `{"url":"https://example.com","alias":"optional"}`. При пропуске alias он генерируется автоматически. Ответ — `201 Created`.
- `PUT /url` — изменить ссылку. JSON: `{"alias":"abc123","url":"https://new.example.com"}`.
- `DELETE /url/{alias}` — удалить ссылку.
- `GET /{alias}` — перенаправить на исходный URL.

Пример создания ссылки:

```sh
. ./.env

curl -u "$AUTH_USER:$AUTH_PASSWORD" \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}' \
  http://localhost:8082/url
```

## Миграции

Версионные миграции находятся в `migrations/` и используют пару `up.sql` / `down.sql`. Compose применяет `up` перед запуском API. Локально используйте те же миграции через CLI.

Откатить последнюю миграцию локально можно командой:

```sh
migrate -path migrations -database "$DATABASE_URL" down 1
```

следующие миграции последовательными версиями, например: `migrate create -ext sql -dir migrations -seq add_description`.

## Проверка

```sh
go test ./internal/...
```

Интеграционные тесты из `tests/` требуют запущенный сервис и PostgreSQL.

## Чек-лист требований

### 🐳 Запуск из коробки

- [x] Есть `docker-compose.yml`, поднимающий Postgres и сервис одной командой `docker compose up`.
- [x] Есть `.env.example` со всеми переменными.
- [x] README описывает запуск: клонировать проект, выполнить `cp .env.example .env`, затем `docker compose up`.
- [x] Порты и креды совпадают между `docker-compose.yml`, `.env.example`, README и `config/local.yaml`.

### 🏗 Архитектура слоёв

- [x] Три слоя: handler → service → storage.
- [x] Бизнес-логика (генерация alias, проверка уникальности, валидация) находится в service. Handler разбирает запрос и формирует ответ, storage выполняет SQL.
- [x] Все ручки идут через service, включая redirect.

### 🌐 HTTP-статусы

- [x] Создание URL → `201 Created`.
- [x] Некорректный JSON → `400 Bad Request`.
- [x] Занятый alias → `409 Conflict`.
- [x] Нет авторизации → `401 Unauthorized`.
- [x] Alias не найден при redirect → `404 Not Found`.
- [x] Внутренняя ошибка БД → `500 Internal Server Error`.
- [x] При ошибках HTTP-статус задаётся до отправки JSON-ответа.

### ✅ Тесты

- [x] Для основных сценариев есть отдельные тесты, в том числе `TestSave_Success`, `TestSave_BadRequest` и `TestSave_Conflict`.
- [x] `BadRequest` проверяет некорректный JSON, а не только валидный JSON с пустым полем.
- [x] Есть тесты на delete и update, а также на save и redirect.
- [x] Basic Auth в интеграционных тестах использует значения из `.env`.

### 🔒 Конфиги

- [x] Секреты не хранятся в коде, настройки передаются через переменные окружения.
- [x] Креды в тестах берутся из `.env`.


