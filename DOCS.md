# Messenger — Документация проекта

## Содержание

1. [Обзор проекта](#1-обзор-проекта)
2. [Архитектура](#2-архитектура)
3. [Структура директорий](#3-структура-директорий)
4. [База данных](#4-база-данных)
5. [Модели данных](#5-модели-данных)
6. [Слой хранилища (Store)](#6-слой-хранилища-store)
7. [Аутентификация (JWT)](#7-аутентификация-jwt)
8. [HTTP Handlers](#8-http-handlers)
9. [WebSocket](#9-websocket)
10. [Точка входа (main.go)](#10-точка-входа-maingo)
11. [Конфигурация и окружение](#11-конфигурация-и-окружение)
12. [Зависимости](#12-зависимости)
13. [API Reference](#13-api-reference)
14. [Текущее состояние и известные проблемы](#14-текущее-состояние-и-известные-проблемы)

---

## 1. Обзор проекта

**Messenger** — бэкенд для мессенджера реального времени, написанный на Go.

**Возможности:**
- Регистрация и вход пользователей с JWT-аутентификацией
- Real-time обмен сообщениями через WebSocket
- Поддержка личных и групповых чатов
- Хранение сообщений в PostgreSQL (с поддержкой шифрования на уровне БД)
- REST API для управления пользователями, чатами и историей сообщений

**Стек:**
- Язык: Go
- БД: PostgreSQL (драйвер `pgx/v5` с пулом соединений)
- WebSocket: `gorilla/websocket`
- Auth: JWT (HS256, `golang-jwt/jwt/v5`)
- Пароли: bcrypt (`golang.org/x/crypto`)

---

## 2. Архитектура

```
┌─────────────────────────────────────────────────────────┐
│                        Browser                          │
└───────────────┬──────────────────────┬──────────────────┘
                │ HTTP/REST            │ WebSocket
                ▼                      ▼
┌───────────────────────┐   ┌─────────────────────────────┐
│     HTTP Handlers     │   │       WebSocket Layer       │
│  - AuthHandler        │   │  ServeWs() → Client         │
│  - MessageHandler     │   │  readPump / writePump        │
│  - ChatHandler (WIP)  │   │  Hub (центральный брокер)   │
└──────────┬────────────┘   └──────────────┬──────────────┘
           │                               │
           ▼                               ▼
┌──────────────────────────────────────────────────────────┐
│                     Store Layer                          │
│         UserStore │ MessageStore │ ChatStore             │
└──────────────────────────────┬───────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────┐
│                      PostgreSQL                          │
│          users │ chats │ chat_participants │ messages     │
└──────────────────────────────────────────────────────────┘
```

### Поток данных

**HTTP (авторизация):**
```
POST /api/auth/register → AuthHandler.Register() → UserStore.CreateUser() → БД → JWT
POST /api/auth/login    → AuthHandler.Login()    → UserStore.GetUserByEmail() → bcrypt → JWT
```

**HTTP (сообщения):**
```
GET /api/messages → jwtMiddleware → MessageHandler.GetMessagesHandler() → MessageStore.GetMessages() → БД
```

**WebSocket (real-time):**
```
WS /ws?token=<JWT>
  → ServeWs(): валидация JWT → создание Client → hub.register
  → Client.readPump(): читает сообщения → hub.broadcast
  → Hub.Run(): сохраняет в БД → рассылает всем клиентам
  → Client.writePump(): отправляет клиенту
```

---

## 3. Структура директорий

```
messenger/
├── cmd/
│   └── server/
│       └── main.go             # Точка входа, инициализация, маршруты
├── internal/
│   ├── auth/
│   │   └── jwt.go              # Генерация и валидация JWT
│   ├── handlers/
│   │   ├── auth_handler.go     # Регистрация и вход
│   │   ├── message_handler.go  # История сообщений
│   │   └── chat_handler.go     # Управление чатами (WIP)
│   ├── models/
│   │   ├── user.go             # Модель пользователя
│   │   ├── message.go          # Модель сообщения
│   │   └── chat.go             # Модель чата
│   ├── store/
│   │   ├── store.go            # Агрегатор хранилищ
│   │   ├── user_store.go       # CRUD для пользователей
│   │   ├── message_store.go    # CRUD для сообщений
│   │   └── chat_store.go       # CRUD для чатов
│   └── websocket/
│       ├── hub.go              # Центральный брокер сообщений
│       └── client.go           # WebSocket-клиент
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   ├── 000001_create_users_table.down.sql
│   ├── 000002_create_chats_tables.up.sql
│   ├── 000002_create_chats_tables.down.sql
│   ├── 000003_create_messages_table.up.sql
│   └── 000003_create_messages_table.down.sql
├── frontend/                   # Статика (раздаётся сервером)
├── docker-compose.yaml         # PostgreSQL в Docker
├── .env                        # Переменные окружения
├── go.mod
└── go.sum
```

---

## 4. База данных

### Схема

#### Таблица `users`
```sql
CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(50)  UNIQUE NOT NULL,
    email         VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,                          -- bcrypt хеш
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email    ON users(email);
```

#### Таблица `chats`
```sql
CREATE TABLE chats (
    id         SERIAL PRIMARY KEY,
    uuid       UUID DEFAULT gen_random_uuid() UNIQUE NOT NULL,
    name       VARCHAR(100),
    is_group   BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()   -- обновляется триггером при новом сообщении
);
```

#### Таблица `chat_participants`
```sql
CREATE TABLE chat_participants (
    chat_id   INTEGER NOT NULL REFERENCES chats(id)  ON DELETE CASCADE,
    user_id   INTEGER NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (chat_id, user_id)
);
```
Связующая таблица «многие ко многим» между чатами и пользователями.

#### Таблица `messages`
```sql
CREATE TABLE messages (
    id                SERIAL PRIMARY KEY,
    uuid              UUID DEFAULT gen_random_uuid() UNIQUE NOT NULL,
    chat_id           INTEGER NOT NULL REFERENCES chats(id)  ON DELETE CASCADE,
    sender_id         INTEGER NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    encrypted_content BYTEA NOT NULL,   -- зашифрованное тело сообщения
    iv                BYTEA NOT NULL,   -- вектор инициализации для шифрования
    message_type      VARCHAR(20) DEFAULT 'text',
    file_name         VARCHAR(255),
    file_size         INTEGER,
    mime_type         VARCHAR(100),
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_read           BOOLEAN DEFAULT FALSE,
    read_at           TIMESTAMP WITH TIME ZONE
);
```

#### Триггер `trigger_update_chat_on_message`
После каждой вставки в `messages` автоматически обновляет `chats.updated_at`:
```sql
CREATE OR REPLACE FUNCTION update_chat_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE chats SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.chat_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_chat_on_message
AFTER INSERT ON messages
FOR EACH ROW EXECUTE FUNCTION update_chat_updated_at();
```

### ER-диаграмма

```
users ──────────────────────── chat_participants ──── chats
  id PK                          chat_id FK               id PK
  username                        user_id FK               uuid
  email                           joined_at                name
  password_hash                                            is_group
  created_at                                               created_at
  updated_at                                               updated_at
                                                               │
                                                               │
                                                          messages
                                                            id PK
                                                            chat_id FK → chats.id
                                                            sender_id FK → users.id
                                                            encrypted_content
                                                            iv
                                                            created_at
                                                            ...
```

---

## 5. Модели данных

### `models.User` — [internal/models/user.go](internal/models/user.go)

```go
type User struct {
    ID           int64     `json:"id"`
    Username     string    `json:"username"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"`        // НЕ сериализуется в JSON
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

var ErrUserNotFound = errors.New("Пользователь не найден")
```

**Важно:** `PasswordHash` помечен `json:"-"` — он никогда не попадает в HTTP-ответы.

---

### `models.Chat` — [internal/models/chat.go](internal/models/chat.go)

```go
type Chat struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    IsGroup   bool      `json:"is_group"`
    CreatedAt time.Time `json:"created_at"`
}
```

---

### `models.Message` — [internal/models/message.go](internal/models/message.go)

```go
type Message struct {
    ChatID int64     `json:"chat_id"`
    User   string    `json:"user"`    // username отправителя (не ID)
    Text   string    `json:"text"`    // текст (в БД хранится как BYTEA)
    SentAt time.Time `json:"sent_at"`
}
```

**Примечание:** В БД хранится `encrypted_content BYTEA`, но при чтении приводится к `string` через `string(contentBytes)`. Реальное шифрование на стороне приложения не реализовано — поле `iv` заполняется заглушкой `'temp_iv'`.

---

## 6. Слой хранилища (Store)

### `Store` — [internal/store/store.go](internal/store/store.go)

Центральный агрегатор всех хранилищ:

```go
type Store struct {
    UserStore    *UserStore
    MessageStore *MessageStore
    ChatStore    *ChatStore
}

func NewStore(db *pgxpool.Pool) *Store
```

Создаётся один раз в `main.go` и передаётся во все хендлеры и Hub.

---

### `UserStore` — [internal/store/user_store.go](internal/store/user_store.go)

| Метод | Сигнатура | Описание |
|-------|-----------|----------|
| `CreateUser` | `(ctx, username, email, password string) (*User, error)` | Хеширует пароль bcrypt, вставляет в БД, возвращает созданного пользователя с `id`, `created_at`, `updated_at` |
| `GetUserByEmail` | `(ctx, email string) (*User, error)` | Ищет пользователя по email; возвращает `ErrUserNotFound` если не найден |
| `SearchUsers` | `(ctx, query string) ([]*User, error)` | Поиск пользователей по `username` (ILIKE, лимит 20) |
| `CheckPasswordHash` | `(password, hash string) bool` | Функция-утилита (не метод): сравнивает пароль с bcrypt-хешем |

**SQL — CreateUser:**
```sql
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, created_at, updated_at
```

**SQL — GetUserByEmail:**
```sql
SELECT id, username, email, password_hash, created_at, updated_at
FROM users WHERE email = $1
```

**SQL — SearchUsers:**
```sql
SELECT id, username, email, created_at, updated_at
FROM users WHERE username ILIKE $1
LIMIT 20
```

---

### `MessageStore` — [internal/store/message_store.go](internal/store/message_store.go)

| Метод | Сигнатура | Описание |
|-------|-----------|----------|
| `GetMessages` | `(ctx, chatID int64) ([]Message, error)` | Возвращает все сообщения чата, отсортированные по времени (ASC) |
| `CreateMessage` | `(ctx, msg *Message, userID int64) (*Message, error)` | Сохраняет сообщение, возвращает сохранённое с username отправителя |

**SQL — GetMessages:**
```sql
SELECT u.username, m.chat_id, m.encrypted_content, m.created_at
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.chat_id = $1
ORDER BY m.created_at ASC
```

**SQL — CreateMessage (CTE):**
```sql
WITH new_msg AS (
    INSERT INTO messages (sender_id, chat_id, encrypted_content, iv)
    VALUES ($1, $2, $3, 'temp_iv')
    RETURNING id, sender_id, encrypted_content, created_at
)
SELECT u.username, nm.encrypted_content, nm.created_at
FROM new_msg nm
JOIN users u ON nm.sender_id = u.id
```

---

### `ChatStore` — [internal/store/chat_store.go](internal/store/chat_store.go)

| Метод | Сигнатура | Описание |
|-------|-----------|----------|
| `GetUserChats` | `(ctx, userID int64) ([]*Chat, error)` | Возвращает все чаты пользователя через `chat_participants` |
| `CreateChat` | `(ctx, name string, isGroup bool, creatorID int64, memberIDs []int64) (*Chat, error)` | Создаёт чат и добавляет всех участников |
| `IsMember` | `(ctx, chatID, userID int64) (bool, error)` | Проверяет, является ли пользователь участником чата |

**SQL — GetUserChats:**
```sql
SELECT c.id, c.name, c.is_group, c.created_at
FROM chats c
JOIN chat_participants cp ON cp.chat_id = c.id
WHERE cp.user_id = $1
ORDER BY c.created_at DESC
```

**SQL — CreateChat:**
```sql
INSERT INTO chats (name, is_group) VALUES ($1, $2)
RETURNING id, name, is_group, created_at
-- затем для каждого участника:
INSERT INTO chat_participants (chat_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
```

**SQL — IsMember:**
```sql
SELECT EXISTS(
    SELECT 1 FROM chat_participants WHERE chat_id=$1 AND user_id=$2
)
```

---

## 7. Аутентификация (JWT)

### Файл: [internal/auth/jwt.go](internal/auth/jwt.go)

#### Структура Claims

```go
type JWTClaims struct {
    UserID   int64  `json:"user_id"`
    Username string `json:"username"`
    jwt.RegisteredClaims
}
```

Содержит `UserID` и `Username` пользователя + стандартные поля JWT (`ExpiresAt` и др.).

#### Константы

| Параметр | Значение |
|----------|----------|
| Алгоритм подписи | HS256 |
| Время жизни токена | 24 часа |
| Ключ контекста | `auth.UserContextKey` |

#### Функции

**`GenerateJWT(user *models.User, jwtSecret string) (string, error)`**

Создаёт JWT с `UserID`, `Username`, временем истечения через 24 часа. Подписывает секретом из конфига.

**`ValidateJWT(tokenString string, jwtSecret string) (*JWTClaims, error)`**

Парсит и валидирует токен. Проверяет алгоритм подписи (защита от `alg: none`). Возвращает claims или ошибку.

#### JWT Middleware (в main.go)

```go
func jwtMiddleware(next http.HandlerFunc, jwtSecret string) http.HandlerFunc
```

Обёртка для защищённых маршрутов:
1. Читает заголовок `Authorization: Bearer <token>`
2. Парсит и валидирует токен
3. Кладёт `*JWTClaims` в контекст запроса под ключом `auth.UserContextKey`
4. При ошибке → 401 Unauthorized

---

## 8. HTTP Handlers

### `AuthHandler` — [internal/handlers/auth_handler.go](internal/handlers/auth_handler.go)

#### `POST /api/auth/register`

**Запрос:**
```json
{
    "username": "john",
    "email": "john@example.com",
    "password": "secret123"
}
```

**Ответ (201 Created):**
```json
{
    "token": "<JWT>"
}
```

**Поток выполнения:**
1. Декодировать JSON тело
2. `UserStore.CreateUser()` — bcrypt пароль + INSERT
3. `auth.GenerateJWT()` — создать токен
4. Вернуть токен

**Ошибки:**
- `400` — неверный JSON
- `500` — ошибка создания пользователя или генерации токена

---

#### `POST /api/auth/login`

**Запрос:**
```json
{
    "email": "john@example.com",
    "password": "secret123"
}
```

**Ответ (200 OK):**
```json
{
    "token": "<JWT>"
}
```

**Поток выполнения:**
1. Декодировать JSON тело
2. `UserStore.GetUserByEmail()` — найти пользователя
3. `CheckPasswordHash()` — bcrypt проверка
4. `auth.GenerateJWT()` — создать токен
5. Вернуть токен

**Ошибки:**
- `400` — неверный JSON
- `401` — неверный email или пароль
- `500` — ошибка сервера

---

### `MessageHandler` — [internal/handlers/message_handler.go](internal/handlers/message_handler.go)

#### `GET /api/messages` *(требует JWT)*

**Ответ (200 OK):**
```json
[
    {
        "chat_id": 1,
        "user": "john",
        "text": "Привет!",
        "sent_at": "2026-03-23T10:00:00Z"
    }
]
```

**⚠️ Текущий баг:** вызывает `GetMessages(ctx)` без параметра `chatID`, тогда как метод ожидает `GetMessages(ctx, chatID int64)` — код не компилируется.

---

### `ChatHandler` — [internal/handlers/chat_handler.go](internal/handlers/chat_handler.go)

**⚠️ Статус: незаконченный (WIP)**

```go
type CreateChatRequest struct {
    Name      string  `json:"name"`
    IsGroup   bool    `json:"is_group"`
    MemberIDs []int64 `json:"member_ids"`
}
```

`GetChatHandler` — вызывает `ChatStore.GetUserChats()`, но не возвращает ответ клиенту. Не зарегистрирован в маршрутах.

---

## 9. WebSocket

### `Client` — [internal/websocket/client.go](internal/websocket/client.go)

Представляет одно WebSocket-соединение.

```go
type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte   // буфер исходящих сообщений (256)
    UserID   int64
    Username string
}
```

#### Таймауты и лимиты

| Константа | Значение | Назначение |
|-----------|----------|------------|
| `writeWait` | 10s | Дедлайн на запись одного сообщения |
| `pongWait` | 60s | Максимальное ожидание Pong от клиента |
| `pingPeriod` | 54s | Интервал отправки Ping (= pongWait * 9/10) |
| `maxMessageSize` | 512 байт | Максимальный размер входящего сообщения |

#### `readPump()`

Горутина чтения входящих сообщений:

```
conn.ReadMessage()
    → hub.broadcast <- &ClientMessage{client, bytes}
```

- При разрыве соединения: `hub.unregister <- client` → закрывает соединение
- Pong-хендлер сбрасывает дедлайн чтения каждый раз при получении Pong

#### `writePump()`

Горутина отправки исходящих сообщений:

```
client.send channel → conn.Write(message)
ticker (54s)        → conn.WriteMessage(Ping)
```

- Пишет все накопленные в буфере сообщения за один `NextWriter`
- При закрытом `send` канале: шлёт `CloseMessage` и завершается

#### `ServeWs(hub, w, r, jwtSecret)`

Точка входа для WebSocket:

1. Читает `?token=<JWT>` из query string
2. Валидирует JWT → получает `UserID`, `Username`
3. Апгрейдит HTTP → WebSocket (`gorilla/websocket`)
4. Создаёт `Client`
5. `hub.register <- client`
6. Запускает `go writePump()` и `go readPump()`

**Важно:** `CheckOrigin` всегда возвращает `true` — CORS не проверяется (TODO).

---

### `Hub` — [internal/websocket/hub.go](internal/websocket/hub.go)

Центральный брокер. Работает в **одной горутине** (`go hub.Run()`).

```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan *ClientMessage
    register   chan *Client
    unregister chan *Client
    store      *store.Store
}
```

#### `Run()`

Бесконечный `select` на 3 канала:

```
register   → clients[client] = true
unregister → delete(clients, client); close(client.send)
broadcast  →
    1. JSON.Unmarshal → models.Message
    2. msg.User = client.Username
    3. MessageStore.CreateMessage() → сохранить в БД
    4. JSON.Marshal(savedMsg)
    5. for client := range clients { client.send <- data }
```

**⚠️ Текущее ограничение:** рассылка идёт **всем** подключённым клиентам, без фильтрации по `chat_id`. Это нужно исправить в рамках ветки `chats/backend`.

---

## 10. Точка входа (main.go)

### Файл: [cmd/server/main.go](cmd/server/main.go)

#### Последовательность запуска

```
1. godotenv.Load()           — загрузить .env
2. loadConfig()              — прочитать DATABASE_URL, JWT_SECRET
3. pgxpool.New()             — создать пул соединений с PostgreSQL
4. store.NewStore(pool)      — создать агрегатор хранилищ
5. websocket.NewHub(store)   — создать Hub
6. go hub.Run()              — запустить брокер в горутине
7. Регистрация маршрутов     — (см. ниже)
8. http.ListenAndServe(:8080)
```

#### Маршруты

| Метод | Путь | Хендлер | Auth |
|-------|------|---------|------|
| `GET` | `/` | Static files (`./frontend`) | — |
| `POST` | `/api/auth/register` | `AuthHandler.Register` | — |
| `POST` | `/api/auth/login` | `AuthHandler.Login` | — |
| `GET` | `/api/messages` | `MessageHandler.GetMessagesHandler` | JWT header |
| `GET/WS` | `/ws` | `websocket.ServeWs` | JWT query param |

---

## 11. Конфигурация и окружение

Файл `.env` в корне проекта:

```env
DATABASE_URL=postgres://user:password@localhost:5432/messenger
JWT_SECRET=your_secret_key_here
```

| Переменная | Назначение | Обязательна |
|------------|-----------|-------------|
| `DATABASE_URL` | Строка подключения к PostgreSQL | Да |
| `JWT_SECRET` | Секрет для подписи JWT токенов | Да |

При отсутствии `.env` используются переменные окружения системы.

**Запуск PostgreSQL через Docker:**
```bash
docker-compose up -d
```

---

## 12. Зависимости

```
github.com/golang-jwt/jwt/v5  v5.3.1   — JWT токены
github.com/gorilla/websocket  v1.5.3   — WebSocket
github.com/jackc/pgx/v5       v5.8.0   — PostgreSQL драйвер + пул
github.com/joho/godotenv      v1.5.1   — загрузка .env
golang.org/x/crypto           v0.48.0  — bcrypt
```

---

## 13. API Reference

### Аутентификация

#### `POST /api/auth/register`
Регистрация нового пользователя.

```
Request:  { "username": string, "email": string, "password": string }
Response: { "token": string }
Errors:   400 Bad Request | 500 Internal Server Error
```

#### `POST /api/auth/login`
Вход существующего пользователя.

```
Request:  { "email": string, "password": string }
Response: { "token": string }
Errors:   400 Bad Request | 401 Unauthorized | 500 Internal Server Error
```

### Сообщения

#### `GET /api/messages`
Получение истории сообщений. Требует заголовок `Authorization: Bearer <token>`.

```
Response: [{ "chat_id": int, "user": string, "text": string, "sent_at": datetime }]
Errors:   401 Unauthorized | 500 Internal Server Error
```

### WebSocket

#### `WS /ws?token=<JWT>`
Установка WebSocket соединения.

**Входящее сообщение (от клиента к серверу):**
```json
{
    "chat_id": 1,
    "text": "Привет!"
}
```

**Исходящее сообщение (от сервера к клиенту):**
```json
{
    "user": "john",
    "text": "Привет!",
    "sent_at": "2026-03-23T10:00:00Z"
}
```

---

## 14. Текущее состояние и известные проблемы

### Ветка `chats/backend`

Текущая работа — добавление полноценной поддержки нескольких чатов.

**Что сделано:**
- ✅ Модель `Chat` ([internal/models/chat.go](internal/models/chat.go))
- ✅ `ChatStore` с методами `GetUserChats`, `CreateChat`, `IsMember` ([internal/store/chat_store.go](internal/store/chat_store.go))
- ✅ `Message.ChatID` добавлен в модель
- ✅ `MessageStore.GetMessages` принимает `chatID` и фильтрует по нему

**Что нужно доделать:**

| Проблема | Файл | Описание |
|----------|------|---------|
| 🔴 Баг компиляции | [handlers/message_handler.go](internal/handlers/message_handler.go) | `GetMessages()` вызывается без `chatID` |
| 🟡 Незакончен | [handlers/chat_handler.go](internal/handlers/chat_handler.go) | `GetChatHandler` не пишет ответ, хендлер не зарегистрирован в маршрутах |
| 🟡 Hub не фильтрует | [websocket/hub.go](internal/websocket/hub.go) | Рассылка всем клиентам, а не участникам чата |
| 🟡 CORS не настроен | [websocket/client.go](internal/websocket/client.go) | `CheckOrigin` возвращает `true` для всех |
| 🟡 Нет валидации | [handlers/auth_handler.go](internal/handlers/auth_handler.go) | Нет проверки длины пароля, формата email |
| 🟡 Нет обработки дублей | [handlers/auth_handler.go](internal/handlers/auth_handler.go) | Ошибка уникальности при регистрации не различается от других ошибок |
| 🟡 IV-заглушка | [store/message_store.go](internal/store/message_store.go) | `iv` заполняется `'temp_iv'` — реальное шифрование не реализовано |
| 🟡 Нет маршрута | [cmd/server/main.go](cmd/server/main.go) | `ChatHandler` не подключён к маршрутам |
