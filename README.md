# README — Кеширующий веб-сервер документов на Go

## Описание

Данный проект реализует веб-сервер для хранения, выдачи и управления электронными документами с поддержкой авторизации, кэширования (Redis), логирования (slog), работы с PostgreSQL (pgx) и маршрутизации (gorilla/mux).

## Архитектура

- **Единая точка входа:** `cmd/main.go` — запуск HTTP-сервера.
- **Конфигурация:** `config.yaml` — параметры сервера, БД, Redis, токен администратора.
- **Docker:** `Dockerfile`, `docker-compose.yml` — контейнеризация, запуск с Redis и PostgreSQL.
- **Миграции:** `migrations/001_init.sql` — создание таблиц в БД.
- **Кэш:** `internal/cache/cache.go` — работа с Redis.
- **Логирование:** `internal/logger/logger.go` — централизованный логгер на slog.
- **Маршрутизация:** `gorilla/mux` — маршруты API.
- **Слои:**
  - `internal/handler/` — HTTP-обработчики (auth, docs).
  - `internal/service/` — бизнес-логика.
  - `internal/repository/` — работа с БД.
  - `internal/util/` — утилиты (пароли, UUID и др.).

## Запуск

### 1. Docker

```sh
docker-compose up --build
```
Запустит сервер, Redis и PostgreSQL.

### 2. Локально

- Установите Go >= 1.18, PostgreSQL, Redis.
- Настройте `config.yaml`.
- Запустите:
  ```sh
  go run cmd/main.go
  ```

## Конфигурация

Пример `config.yaml`:
```yaml
server:
  addr: ":8080"
  admin_token: "secret123"

postgres:
  dsn: "postgres://postgres:postgres@postgres:5432/webdb?sslmode=disable"

redis:
  addr: "redis:6379"
  password: ""
  db: 0

security:
  token_ttl_seconds: 3600
```

## REST API

### 1. Регистрация пользователя

**POST** `/api/register`

**Вход:**
```json
{
  "token": "secret123",
  "login": "testUser1",
  "pswd": "StrongP@ssw0rd"
}
```
**Выход:**
```json
{
  "response": { "login": "testUser1" }
}
```

### 2. Аутентификация

**POST** `/api/auth`

**Вход:**
```json
{
  "login": "testUser1",
  "pswd": "StrongP@ssw0rd"
}
```
**Выход:**
```json
{
  "response": { "token": "<token_uuid_generated>" }
}
```

### 3. Загрузка нового документа

**POST** `/api/docs`

**Вход (multipart form):**
- `meta` — JSON с параметрами:
  ```json
  {
    "name": "photo.jpg",
    "file": true,
    "public": false,
    "token": "jwt_or_random_token",
    "mime": "image/jpg",
    "grant": ["login1", "login2"]
  }
  ```
- `json` — дополнительные данные (опционально)
- `file` — файл документа

**Выход:**
```json
{
  "data": {
    "json": { ... },
    "file": "photo.jpg"
  }
}
```

### 4. Получение списка документов

**GET/HEAD** `/api/docs?token=...&login=...&key=...&value=...&limit=...`

**Выход:**
```json
{
  "data": {
    "docs": [
      {
        "id": "qwdj1q4o34u34ih759ou1",
        "name": "photo.jpg",
        "mime": "image/jpg",
        "file": true,
        "public": false,
        "created": "2018-12-24 10:30:56",
        "grant": ["login1", "login2"]
      }
    ]
  }
}
```

### 5. Получение одного документа

**GET/HEAD** `/api/docs/<id>`

- Если файл: возвращается файл с нужным mime.
- Если JSON:
  ```json
  {
    "data": { ... }
  }
  ```

### 6. Удаление документа

**DELETE** `/api/docs/<id>`

Доступ осуществялеться через поле Authorization: Bearer <token_uuid_generated>

**Выход:**
```json
{
  "response": { "qwdj1q4o34u34ih759ou1": true }
}
```

### 7. Завершение сессии

**DELETE** `/api/auth`

Доступ осуществялеться через поле Authorization: Bearer <token_uuid_generated>

**Выход:**
```json
{
  "response": { "qwdj1q4o34u34ih759ou1": true }
}
```

## Шаблон ответа

```json
{
  "error": { "code": 123, "text": "so sad" },
  "response": { ... },
  "data": { ... }
}
```
- Поля присутствуют только если заполнены.

## Кэширование

- **GET/HEAD** запросы к `/api/docs` и `/api/docs/<id>` — выдаются из Redis.
- **POST/DELETE** — инвалидируют кэш (выборочно).
- Кэш ключи: по токену, id документа, параметрам фильтрации.

## Валидация

- Логин: минимум 8 символов, латиница и цифры.
- Пароль: минимум 8 символов, 2 буквы разных регистров, 1 цифра, 1 спецсимвол.

## Логирование

- Все действия логируются через slog.
- Логи пишутся в stdout.

## Тестирование

- Использовал Postman для тестирования API.
- Примеры запросов выше.

## Используемые библиотеки

- `github.com/gorilla/mux` — маршрутизация
- `github.com/jackc/pgx/v5` — PostgreSQL
- `github.com/go-redis/redis/v9` — Redis
- `golang.org/x/exp/slog` — логирование
- `github.com/google/uuid` — UUID
- `golang.org/x/crypto/bcrypt` — хеширование паролей
- `gopkg.in/yaml.v3` — парсинг .yaml файлов

## Примеры ошибок

- Некорректные параметры — 400
- Не авторизован — 401
- Нет прав доступа — 403
- Неверный метод — 405
- Внутренняя ошибка — 500
- Не реализовано — 501

## Контакты

Автор: Ovsyannikov Alexandr
