## 🐳 Запуск из коробки
- [x] Есть docker-compose.yml, поднимающий Postgres + сервис одной командой docker compose up
- [x] Есть .env.example (именно так, не example.env) со всеми переменными
- [x] README: клонируем -> cp .env.example .env -> docker compose up -> сервис работает. Без «допиши вот это» и «подними сам базу»
- [x] Порты и креды совпадают между docker-compose.yml, .env.example, README и config/local.yaml

## 🏗 Архитектура слоёв
- [x] Три слоя: handler -> service -> storage. В видео сделано handler -> storage напрямую — у нас так нельзя
- [x] Бизнес-логика (генерация alias, проверка уникальности, валидация) только в service. В handler — парсинг запроса и формирование ответа, в storage — только SQL
- [x] Все ручки идут через service, включая redirect (частая ошибка: redirect ходит сразу в storage)

## 🌐 HTTP-статусы (а не «всё через 400»)
- [x] Создание URL -> 201 Created, не 200
- [x] Битый JSON -> 400
- [x] Alias уже занят -> 409 Conflict, не 400
- [x] Нет авторизации -> 401
- [x] Alias не найден на редирект -> 404, не «internal error»
- [x] Внутренняя ошибка БД -> 500
- [x] Везде render.Status(...) перед render.JSON(...) — иначе клиент получит 200 на ошибку

## ✅ Тесты
- [x] Отдельные тест-функции на каждый кейс: TestSave_Success, TestSave_BadRequest, TestSave_Conflict — не один общий
- [x] BadRequest проверяет реально битый JSON, а не валидный JSON с пустым полем
- [x] Есть тесты на delete и update (не только save и redirect)
- [x] Basic Auth в тестах берёт креды из того же .env, что и сервис — без хардкода

## 🔒 Конфиги
- [x] Никаких секретов в коде, всё через env
- [x] Креды в тестах = креды в .env