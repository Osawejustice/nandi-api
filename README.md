# Nandi API

Go API for Nandi, a multi-tenant customer engagement platform (Africa-first).

This repository is a modular monolith. The Next.js frontend is a separate product and talks to this API only over HTTP JSON (`/api/v1/...`) and WebSockets.

## Stack

- Go 1.22+
- Gin
- GORM + PostgreSQL 16
- Redis 7 (Pub/Sub, presence)
- Viper + `.env`
- zerolog
- JWT (access + refresh) and hashed API keys
- ChannelProvider (Africa's Talking, Evolution, stub failover)
- Groq / OpenAI-compatible sentiment + summary
- swaggo/swag

## Layout

```
cmd/api/            process entrypoint
internal/config     environment-based config
internal/database   Postgres + Redis + AutoMigrate
internal/middleware request ID, logging, recovery, auth, CORS, limits
internal/handlers   HTTP handlers (no business logic)
internal/services   business logic
internal/repositories
internal/models
internal/providers  ChannelProvider + Africa's Talking + Evolution + stub
internal/realtime   WebSocket hub + Redis Pub/Sub
internal/ai         sentiment and summary
internal/utils
pkg/                public libraries if needed
migrations/         schema snapshot
docs/               OpenAPI + Postman collection
scripts/
```

Do not add new top-level folders.

## Quick start

```bash
cp .env.example .env
docker compose up -d postgres redis
go mod tidy
go build ./...
go test ./...
go run ./cmd/api
```

Or run the full stack (API + Postgres + Redis):

```bash
docker compose up --build
```

Then:

- Health: http://localhost:8080/health
- Swagger UI: http://localhost:8080/swagger/index.html
- Postman: import `docs/Nandi_API.postman_collection.json`
- WebSocket: `ws://localhost:8080/api/v1/ws?access_token=<JWT>`

The API starts even if Postgres or Redis is down (it logs a warning). Auth, inbox, and webhooks require Postgres. Realtime fan-out across processes requires Redis.

## Commands

| Action | Command |
| --- | --- |
| Start infra | `docker compose up -d postgres redis` |
| Build | `go build ./...` |
| Run API | `go run ./cmd/api` |
| Tests | `go test ./...` |
| Generate Swagger | `swag init -g cmd/api/main.go -o docs` |
| Tidy modules | `go mod tidy` |
| Format | `gofmt -w .` |

## Golden demo

1. `POST /api/v1/auth/register` — creates tenant + owner.
2. Connect WebSocket with the access token.
3. `POST /api/v1/webhooks/{tenant_slug}/sms/africastalking` (or `POST /api/v1/dev/inbound`) — inbound SMS.
4. Agent receives `conversation_created` / `new_message`.
5. `GET /api/v1/conversations` then `POST /api/v1/conversations/:id/messages` — reply goes through ChannelProvider.
6. Without Africa's Talking credentials, outbound fails over to the stub provider. ProviderLog records every attempt.
7. `GET /api/v1/analytics/overview` reflects the interaction.

## Environment

See `.env.example`. Provider keys, AI keys, and `JWT_SECRET` live only in the backend environment.

## Rules

- Every business table has `tenant_id`. Queries are always tenant-scoped from the JWT / API key, never from a client-supplied tenant id.
- Provider traffic goes through `ChannelProvider`. Core services never import a concrete vendor package.
- AI failure never blocks message persistence.
- Backend is the source of truth. Keep business logic in services, not handlers.
- Swagger annotations stay in sync with handlers.
