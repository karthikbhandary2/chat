# real-time chat application

A real-time chat application built with Go, PostgreSQL, Redis, and React —
WebSocket-based messaging, JWT authentication, message persistence, typing
indicators, online presence, unread counts, and full-text search.

---

## Features

- User registration and JWT-based authentication
- Real-time messaging over WebSockets (direct and group conversations)
- Message persistence with keyset-paginated history
- Typing indicators (ephemeral, not persisted)
- Online/offline presence via Redis, with TTL-based expiry
- Unread message counts, computed per conversation per user
- Full-text message search (Postgres `tsvector` + GIN index), scoped to the
  requester's own conversations
- A React frontend with login/registration, a conversation sidebar, live
  messaging, and a presence checker

---

## Architecture

```
┌─────────────┐        WebSocket (/ws)        ┌──────────────┐
│   React      │ ─────────────────────────────▶│              │
│   frontend   │                                │   Go server  │
│  (Vite,      │◀───────── REST (/api/v1) ─────│   (chi)      │
│  localhost   │                                │              │
│  :5173)      │                                └──────┬───────┘
└─────────────┘                                        │
                                                          │
                                   ┌──────────────────────┼──────────────────────┐
                                   │                       │                      │
                             ┌─────▼─────┐         ┌───────▼──────┐      ┌────────▼───────┐
                             │ PostgreSQL │         │     Redis     │      │  In-memory hub  │
                             │ (sqlc)     │         │  (presence)   │      │ (goroutine +    │
                             │            │         │               │      │  channels)      │
                             └────────────┘         └───────────────┘      └─────────────────┘
```

**Concurrency model**: each WebSocket connection is backed by a `Client`
struct running two goroutines — a `readPump` (reads incoming frames) and a
`writePump` (owns all writes to that connection, fed by a buffered channel).
A single `Hub` goroutine owns the map of connected clients and is the only
thing that ever touches it, so no mutexes are needed anywhere in the
connection-tracking logic — all coordination happens through channels
(`register`, `unregister`, `broadcast`).

**Message flow**: an incoming WebSocket message is authorized (participant
check), persisted to Postgres, and only then broadcast to currently-connected
participants — so live delivery and history are always consistent with each
other.

---

## Tech stack

| Layer | Choice |
|---|---|
| Language | Go 1.22+ |
| HTTP router | [chi](https://github.com/go-chi/chi) |
| WebSockets | [gorilla/websocket](https://github.com/gorilla/websocket) |
| Database | PostgreSQL 16, via [pgx](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev) |
| Cache / pub-sub | Redis 7, via [go-redis](https://github.com/redis/go-redis) |
| Auth | JWT ([golang-jwt](https://github.com/golang-jwt/jwt)), bcrypt |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Frontend | React 18 + Vite |

---

## Project structure

```
chat/
├── cmd/server/            # entrypoint (main.go)
├── internal/
│   ├── auth/                # JWT generation/validation
│   ├── db/sqlc/              # generated queries + hand-written store/tx logic
│   ├── handlers/             # HTTP handlers
│   ├── middleware/            # auth middleware
│   ├── models/                 # response DTOs
│   ├── presence/                # shared Redis key helpers
│   └── ws/                       # Client, Hub, WebSocket handler
├── migrations/               # SQL migrations (golang-migrate)
├── query/                    # sqlc source .sql query files
├── web/                      # legacy static test client
├── frontend/            # React app (Vite)
├── docker-compose.yml
├── sqlc.yaml
└── .env
```

---

## Setup

### 1. Prerequisites
- Go 1.22+
- Node 20+ (for the frontend)
- Docker + Docker Compose

### 2. Start Postgres and Redis
```bash
docker compose up -d
```

### 3. Configure environment
Create `.env` in the project root:
```
DB_URL=postgres://chat:secret@localhost:25433/chat?sslmode=disable
REDIS_URL=redis://localhost:6379/0
JWT_SECRET=<generate with: openssl rand -base64 32>
```

### 4. Run migrations
```bash
make migrateup
```

### 5. Generate sqlc code (after any query/schema change)
```bash
make sqlc
```

### 6. Run the backend
```bash
make run
```
Server starts on `http://localhost:8082`.

### 7. Run the frontend
```bash
cd chat-frontend
npm install
npm run dev
```
Opens on `http://localhost:5173`. The backend must have CORS enabled for
this origin (see `main.go` — `cors.Handler`).

---

## REST API

All endpoints are prefixed with `/api/v1`. Endpoints marked 🔒 require
`Authorization: Bearer <token>`.

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness check |
| POST | `/register` | Create an account |
| POST | `/login` | Authenticate, returns a JWT + user object |
| 🔒 GET | `/me` | Current user's profile |
| 🔒 POST | `/conversations` | Create a conversation (`{"type": "direct", "usernames": [...]}`) |
| 🔒 GET | `/conversations` | List the caller's conversations |
| 🔒 GET | `/conversations/{id}/messages` | Paginated message history (`?before=<RFC3339>&limit=<n>`) |
| 🔒 GET | `/conversations/{id}/unread` | Unread message count for the caller |
| 🔒 POST | `/conversations/{id}/read` | Mark a conversation as read |
| 🔒 GET | `/messages/search` | Full-text search (`?q=<query>&limit=<n>`), scoped to the caller's conversations |
| 🔒 GET | `/users/{id}/presence` | Check whether a user is currently online |
| 🔒 GET | `/users/by-username/{username}` | Look up a user's id by username |

---

## WebSocket protocol

Connect to `ws://localhost:8082/ws?token=<jwt>`. The token is passed as a
query parameter since browsers can't set custom headers on the WebSocket
handshake.

**Outgoing (client → server)**, JSON text frames:
```json
{"type": "message", "conversation_id": "<uuid>", "content": "hello"}
{"type": "typing", "conversation_id": "<uuid>"}
```

**Incoming (server → client)**:
```json
{"type": "message", "id": "<uuid>", "conversation_id": "<uuid>", "sender_id": "<uuid>", "content": "hello", "created_at": "<RFC3339>"}
{"type": "typing", "conversation_id": "<uuid>", "user_id": "<uuid>"}
```

A `message` sent by a client is also echoed back to that same client's own
connection (useful for multi-device sync); the frontend deduplicates this
against its own optimistic UI update by checking `sender_id`.

---

## Known limitations

- Single-server only — presence and the in-memory hub are not shared across
  multiple server instances (no Redis pub/sub  for cross-server message
  routing).
- No refresh tokens — JWTs are valid for 12 hours with no renewal flow.
- Group conversations can be created via the API, but there's no endpoint to
  add/remove members after creation, or to rename a group.
- No rate limiting is enabled by default (see `internal/ws/client.go` for
  where a per-client limiter can be added).
