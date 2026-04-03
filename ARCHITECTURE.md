# Архитектура проекта Messenger

## Цель
Сервис отправки сообщений через браузер с шифрованием данных (AES-256-GCM).

---

## Визуальная схема слоёв

```
┌─────────────────────────────────────────────────────────────┐
│                     КЛИЕНТ (Браузер)                        │
│              frontend/index.html + app.js                   │
│         HTTP-запросы (fetch) + WebSocket соединение         │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   СЛОЙ МАРШРУТИЗАЦИИ                        │
│                   cmd/server/main.go                        │
│                                                             │
│  POST /api/auth/register  →  AuthHandler.Register          │
│  POST /api/auth/login     →  AuthHandler.Login             │
│  GET  /api/chats          →  ChatHandler.GetChatHandler     │
│  POST /api/chats          →  ChatHandler.CreateChatHandler  │
│  GET  /api/messages       →  MessageHandler.GetMessages     │
│  GET  /api/users          →  UserHandler.SearchUsers        │
│  GET  /ws                 →  websocket.ServeWs              │
│                                                             │
│  jwtMiddleware — проверяет Bearer токен на каждом запросе   │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    СЛОЙ ХЕНДЛЕРОВ                           │
│               internal/handlers/                            │
│                                                             │
│  auth_handler.go     — Register, Login                      │
│  chat_handler.go     — GetChatHandler, CreateChatHandler    │
│  message_handler.go  — GetMessagesHandler                   │
│  user_handler.go     — SearchUsersHandler                   │
│                                                             │
│  Хендлеры читают HTTP запрос, вызывают Store, возвращают    │
│  JSON ответ. Не знают ничего о БД напрямую.                 │
└──────────┬─────────────────────────────────┬────────────────┘
           │                                 │
           ▼                                 ▼
┌──────────────────────┐       ┌─────────────────────────────┐
│  СЛОЙ WEBSOCKET      │       │      СЛОЙ ХРАНИЛИЩА         │
│  internal/websocket/ │       │      internal/store/        │
│                      │       │                             │
│  hub.go              │       │  store.go                   │
│  - Hub struct        │       │  - Store (агрегатор)        │
│  - Run()             │──────▶│                             │
│  - broadcast канал   │       │  user_store.go              │
│                      │       │  - CreateUser               │
│  client.go           │       │  - GetUserByEmail           │
│  - Client struct     │       │  - SearchUsers              │
│  - readPump()        │       │                             │
│  - writePump()       │       │  message_store.go           │
│  - ServeWs()         │       │  - GetMessages              │
└──────────────────────┘       │  - CreateMessage            │
                               │                             │
                               │  chat_store.go              │
                               │  - CreateChat               │
                               │  - GetUserChats             │
                               │  - IsMember                 │
                               │  - GetMemberIDs             │
                               └──────────────┬──────────────┘
                                              │
                                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  СЛОЙ ИНФРАСТРУКТУРЫ                        │
│                                                             │
│  internal/auth/jwt.go       — генерация и валидация токенов │
│  internal/crypto/aes.go     — шифрование/расшифровка        │
│  internal/models/           — структуры данных              │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      БАЗА ДАННЫХ                            │
│                  PostgreSQL (pgxpool)                       │
│                                                             │
│  users               — пользователи                        │
│  chats               — чаты                                │
│  chat_participants   — участники чатов                      │
│  messages            — сообщения (encrypted_content, iv)    │
└─────────────────────────────────────────────────────────────┘
```

---

## Слой 1 — Точка входа (`cmd/server/main.go`)

### `loadConfig() (Config, error)`
Читает переменные окружения `DATABASE_URL` и `JWT_SECRET`.
Возвращает ошибку если они не заданы.

### `jwtMiddleware(next http.HandlerFunc, jwtSecret string) http.HandlerFunc`
Обёртка над хендлером. Проверяет заголовок `Authorization: Bearer <token>`.
Валидирует токен через `auth.ValidateJWT`.
Кладёт `*auth.JWTClaims` в контекст запроса под ключом `auth.UserContextKey`.
Все защищённые маршруты проходят через неё.

### `main()`
1. Загружает `.env` через `godotenv`
2. Вызывает `loadConfig()`
3. Создаёт пул соединений с PostgreSQL (`pgxpool.New`)
4. Создаёт `store.NewStore(dbpool)` — хранилище данных
5. Создаёт `websocket.NewHub(mainStore)` и запускает его в горутине
6. Регистрирует все HTTP маршруты
7. Запускает сервер на `:8080`

---

## Слой 2 — Хендлеры (`internal/handlers/`)

### `auth_handler.go`

**Структуры:**
- `AuthHandler` — хранит `store *store.Store` и `jwtSecret string`
- `RegisterRequest` — `{username, email, password}`
- `LoginRequest` — `{email, password}`
- `AuthResponse` — `{token}`
- `ErrorResponse` — `{error}` (используется во всех хендлерах)

**`writeJSON(w, status, v)`**
Вспомогательная функция. Устанавливает `Content-Type: application/json`, статус и кодирует ответ.
Используется во всех хендлерах пакета.

**`Register(w, r)`**
1. Декодирует тело запроса в `RegisterRequest`
2. Валидирует: username 3-50 символов, пароль мин. 6, email содержит `@`
3. Вызывает `store.UserStore.CreateUser`
4. Если ошибка — проверяет `pgconn.PgError` с кодом `23505` (дубль) → возвращает 409
5. Генерирует JWT через `auth.GenerateJWT` → возвращает токен

**`Login(w, r)`**
1. Декодирует тело в `LoginRequest`
2. Ищет пользователя через `store.UserStore.GetUserByEmail`
3. Проверяет пароль через `store.CheckPasswordHash`
4. Генерирует JWT → возвращает токен

---

### `chat_handler.go`

**Структуры:**
- `ChatHandler` — хранит `store *store.Store`
- `CreateChatRequest` — `{name, is_group, member_ids}`

**`GetChatHandler(w, r)`**
1. Достаёт `claims` из контекста (положил `jwtMiddleware`)
2. Вызывает `store.ChatStore.GetUserChats(userID)`
3. Возвращает массив чатов JSON

**`CreateChatHandler(w, r)`**
1. Достаёт `claims` из контекста
2. Декодирует `CreateChatRequest`
3. Вызывает `store.ChatStore.CreateChat(name, isGroup, creatorID, memberIDs)`
   — создатель автоматически добавляется как участник
4. Возвращает созданный чат (HTTP 201)

---

### `message_handler.go`

**`GetMessagesHandler(w, r)`**
1. Достаёт `claims` из контекста
2. Читает `?chat_id=N` из URL параметров, конвертирует в `int64`
3. Проверяет через `store.ChatStore.IsMember` — является ли пользователь участником
4. Вызывает `store.MessageStore.GetMessages(chatID)`
5. Возвращает массив сообщений JSON

---

### `user_handler.go`

**`SearchUsersHandler(w, r)`**
1. Проверяет авторизацию через контекст
2. Читает `?q=текст` из URL параметров
3. Вызывает `store.UserStore.SearchUsers(query)`
4. Возвращает массив пользователей JSON (без паролей)

---

## Слой 3 — WebSocket (`internal/websocket/`)

### `hub.go`

**Структуры:**
- `ClientMessage` — обёртка `{Client *Client, Message []byte}` для передачи в канал
- `Hub` — центральный узел:
  - `clients map[*Client]bool` — все подключённые клиенты
  - `broadcast chan *ClientMessage` — канал входящих сообщений
  - `register chan *Client` — канал регистрации новых клиентов
  - `unregister chan *Client` — канал отключения клиентов
  - `store *store.Store` — доступ к БД

**`NewHub(s *store.Store) *Hub`**
Создаёт и инициализирует Hub. Вызывается в `main()`.

**`Run()`**
Бесконечный цикл с `select` по трём каналам:
- `register` → добавляет клиента в `clients`
- `unregister` → удаляет клиента, закрывает канал `send`
- `broadcast` → обрабатывает входящее сообщение:
  1. Парсит JSON в `models.Message`
  2. Устанавливает `msg.User` из данных клиента
  3. Сохраняет в БД через `store.MessageStore.CreateMessage`
  4. Сериализует обратно в JSON
  5. Получает список участников чата через `store.ChatStore.GetMemberIDs`
  6. Строит `map[int64]bool` для быстрой проверки
  7. Рассылает только тем клиентам, чей `UserID` есть в списке

---

### `client.go`

**Структуры:**
- `Client` — представляет одно WebSocket соединение:
  - `hub *Hub` — ссылка на хаб
  - `conn *websocket.Conn` — само соединение
  - `send chan []byte` — буферизованный канал исходящих сообщений
  - `UserID int64`, `Username string` — данные из JWT

**Константы:**
- `writeWait = 10s` — таймаут на запись
- `maxMessageSize = 32KB` — максимальный размер сообщения
- `pongWait = 60s` — таймаут ожидания pong от клиента
- `pingPeriod` — интервал отправки ping (~54 секунды)

**`readPump()`**
Горутина. Читает сообщения от клиента в бесконечном цикле.
Отправляет каждое сообщение в `hub.broadcast`.
При разрыве соединения — отправляет клиента в `hub.unregister`.

**`writePump()`**
Горутина. Читает из канала `send` и пишет клиенту.
Периодически отправляет ping для поддержания соединения.
При ошибке записи — закрывает соединение.

**`ServeWs(hub, w, r, jwtSecret)`**
HTTP хендлер для апгрейда соединения до WebSocket.
1. Достаёт `token` из URL параметра (`?token=...`)
2. Валидирует токен через `auth.ValidateJWT`
3. Апгрейдит соединение через `upgrader.Upgrade`
4. Создаёт `Client` с `UserID` и `Username` из токена
5. Регистрирует клиента в хабе
6. Запускает `writePump()` и `readPump()` как горутины

---

## Слой 4 — Хранилище (`internal/store/`)

### `store.go`

**`Store`** — агрегатор, объединяет все хранилища:
- `UserStore *UserStore`
- `MessageStore *MessageStore`
- `ChatStore *ChatStore`

**`NewStore(db *pgxpool.Pool) *Store`**
Создаёт все три хранилища и возвращает агрегатор.

---

### `user_store.go`

**`CreateUser(ctx, username, email, password) (*User, error)`**
1. Хеширует пароль через `bcrypt.GenerateFromPassword`
2. Выполняет `INSERT INTO users` с `RETURNING id, created_at, updated_at`
3. Возвращает заполненный `*models.User`
Связана с: `AuthHandler.Register`

**`GetUserByEmail(ctx, email) (*User, error)`**
`SELECT` по email. Если не найден — возвращает `models.ErrUserNotFound`.
Связана с: `AuthHandler.Login`

**`CheckPasswordHash(password, hash) bool`**
Сравнивает пароль с bcrypt хешем. Не метод, обычная функция.
Связана с: `AuthHandler.Login`

**`SearchUsers(ctx, query) ([]*User, error)`**
`SELECT` с `ILIKE '%query%'`, LIMIT 20.
Связана с: `UserHandler.SearchUsersHandler`

---

### `message_store.go`

**`GetMessages(ctx, chatID int64) ([]Message, error)`**
JOIN с таблицей users по `sender_id`.
Фильтр `WHERE m.chat_id = $1`.
Читает `encrypted_content` как байты → конвертирует в строку.
Связана с: `MessageHandler.GetMessagesHandler`

**`CreateMessage(ctx, msg *Message, userID int64) (*Message, error)`**
CTE-запрос: INSERT + SELECT в одном запросе.
Сохраняет `sender_id`, `chat_id`, `encrypted_content`, `iv='temp_iv'`.
Возвращает сохранённое сообщение с username из JOIN.
Связана с: `Hub.Run()` (через broadcast)

---

### `chat_store.go`

**`CreateChat(ctx, name, isGroup, creatorID, memberIDs) (*Chat, error)`**
1. `INSERT INTO chats` → получает `id`
2. `INSERT INTO chat_participants` для создателя
3. `INSERT INTO chat_participants` для каждого из `memberIDs` (с `ON CONFLICT DO NOTHING`)
Связана с: `ChatHandler.CreateChatHandler`

**`GetUserChats(ctx, userID) ([]*Chat, error)`**
JOIN `chats` с `chat_participants` по `user_id`.
Возвращает все чаты где пользователь является участником.
Связана с: `ChatHandler.GetChatHandler`

**`IsMember(ctx, chatID, userID) (bool, error)`**
`SELECT EXISTS(...)` — быстрая проверка участия.
Связана с: `MessageHandler.GetMessagesHandler` (проверка доступа)

**`GetMemberIDs(ctx, chatID) ([]int64, error)`**
Возвращает все `user_id` из `chat_participants` для чата.
Связана с: `Hub.Run()` (фильтрация рассылки)

---

## Слой 5 — Инфраструктура

### `internal/auth/jwt.go`

**`GenerateJWT(user *User, jwtSecret string) (string, error)`**
Создаёт JWT токен с полями `UserID`, `Username`, срок действия 24 часа.
Подписывает алгоритмом HS256.
Связана с: `AuthHandler.Register`, `AuthHandler.Login`

**`ValidateJWT(tokenString, jwtSecret) (*JWTClaims, error)`**
Парсит и проверяет JWT токен.
Связана с: `jwtMiddleware` (main.go), `ServeWs` (client.go)

**`JWTClaims`** — структура данных токена: `UserID int64`, `Username string`

**`UserContextKey`** — типизированный ключ для хранения claims в `context.Context`

---

### `internal/crypto/aes.go`

**`Encrypt(key, plaintext []byte) (ciphertext, iv []byte, error)`**
1. Создаёт AES блок из 32-байтного ключа
2. Создаёт GCM режим (`cipher.NewGCM`)
3. Генерирует случайный nonce (IV) 12 байт через `crypto/rand`
4. Шифрует через `aesGCM.Seal`
Связана с: `MessageStore.CreateMessage` (когда будет подключена)

**`Decrypt(key, ciphertext, iv []byte) ([]byte, error)`**
1. Создаёт AES блок и GCM режим
2. Расшифровывает через `aesGCM.Open`
3. GCM автоматически проверяет целостность данных
Связана с: `MessageStore.GetMessages` (когда будет подключена)

---

## Слой 6 — Модели (`internal/models/`)

### `user.go`
```
User {
    ID           int64     json:"id"
    Username     string    json:"username"
    Email        string    json:"email"
    PasswordHash string    json:"-"   ← не сериализуется в JSON
    CreatedAt    time.Time json:"created_at"
    UpdatedAt    time.Time json:"updated_at"
}
ErrUserNotFound — sentinel ошибка для "пользователь не найден"
```

### `message.go`
```
Message {
    ChatID  int64     json:"chat_id"
    User    string    json:"user"
    Text    string    json:"text"
    SentAt  time.Time json:"sent_at"
}
```

### `chat.go`
```
Chat {
    ID        int64     json:"id"
    Name      string    json:"name"
    IsGroup   bool      json:"is_group"
    CreatedAt time.Time json:"created_at"
}
```

---

## Схема базы данных

```
users
├── id SERIAL PRIMARY KEY
├── username VARCHAR UNIQUE
├── email VARCHAR UNIQUE
├── password_hash VARCHAR
├── created_at TIMESTAMP
└── updated_at TIMESTAMP

chats
├── id SERIAL PRIMARY KEY
├── uuid UUID UNIQUE
├── name VARCHAR
├── is_group BOOLEAN
├── created_at TIMESTAMP
└── updated_at TIMESTAMP  ← обновляется триггером при новом сообщении

chat_participants
├── chat_id → chats.id
├── user_id → users.id
└── joined_at TIMESTAMP

messages
├── id SERIAL PRIMARY KEY
├── uuid UUID UNIQUE
├── chat_id → chats.id
├── sender_id → users.id
├── encrypted_content BYTEA  ← зашифрованный текст
├── iv BYTEA                 ← вектор инициализации для AES-GCM
├── message_type VARCHAR     (text / file)
├── file_name VARCHAR
├── file_size INTEGER
├── mime_type VARCHAR
├── created_at TIMESTAMP
├── updated_at TIMESTAMP
├── is_read BOOLEAN
└── read_at TIMESTAMP
```

---

## Поток данных: отправка сообщения

```
Браузер
  │  WebSocket: {"chat_id": 1, "text": "привет"}
  ▼
client.readPump()
  │  отправляет в hub.broadcast канал
  ▼
hub.Run() — case broadcast:
  │  1. json.Unmarshal → models.Message
  │  2. msg.User = client.Username
  ▼
MessageStore.CreateMessage()
  │  Encrypt(key, text) → ciphertext, iv  [будет в Этапе 2]
  │  INSERT INTO messages
  │  RETURNING → savedMsg
  ▼
ChatStore.GetMemberIDs(chat_id)
  │  SELECT user_id FROM chat_participants
  ▼
hub.Run() — рассылка
  │  для каждого client в h.clients:
  │    если client.UserID в memberSet → отправить
  ▼
client.writePump()
  │  пишет JSON в WebSocket
  ▼
Браузер получает сообщение
```

---

## Поток данных: авторизация

```
Браузер
  │  POST /api/auth/login {"email": "...", "password": "..."}
  ▼
jwtMiddleware — НЕ применяется (публичный маршрут)
  ▼
AuthHandler.Login()
  │
  ├── UserStore.GetUserByEmail(email)
  │     SELECT * FROM users WHERE email = $1
  │
  ├── CheckPasswordHash(password, user.PasswordHash)
  │     bcrypt.CompareHashAndPassword
  │
  └── auth.GenerateJWT(user, secret)
        jwt.NewWithClaims → подписанный токен
  ▼
Браузер получает {"token": "eyJ..."}
  │
  └── Все следующие запросы: Authorization: Bearer eyJ...
```
