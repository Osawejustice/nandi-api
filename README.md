# Nandi API

Go API for Nandi, a multi-tenant customer engagement platform (Africa-first).

This repository is a modular monolith. Day 1 is the project skeleton: Gin, Viper, zerolog, GORM/Postgres, Redis, Swagger, and Docker Compose. Auth, domain models, and providers come in later prompts.

## Stack

- Go 1.22+
- Gin
- GORM + PostgreSQL 16
- Redis 7
- Viper + `.env`
- zerolog
- swaggo/swag

## Layout

```
cmd/api/            process entrypoint
internal/config     environment-based config
internal/database   Postgres + Redis connection helpers
internal/middleware request ID, logging, recovery
internal/handlers   HTTP handlers (no business logic)
internal/services   business logic (empty until later)
internal/repositories
internal/models
internal/providers  ChannelProvider adapters (later)
internal/realtime   WebSocket hub (later)
internal/ai         sentiment (later)
internal/utils
pkg/                public libraries if needed
migrations/
docs/               OpenAPI + Postman collection
scripts/
```

Do not add new top-level folders.

## Quick start

```bash
cp .env.example .env
docker compose up -d
go mod tidy
go build ./...
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/api/main.go -o docs
go run ./cmd/api
```

Then:

- Health: http://localhost:8080/health
- Swagger UI: http://localhost:8080/swagger/index.html
- Postman: import `docs/Nandi_API.postman_collection.json`

The API starts even if Postgres or Redis is down (it logs a warning). Bring compose up first for a full local stack.

## Commands

| Action | Command |
| --- | --- |
| Start infra | `docker compose up -d` |
| Build | `go build ./...` |
| Run API | `go run ./cmd/api` |
| Generate Swagger | `swag init -g cmd/api/main.go -o docs` |
| Tidy modules | `go mod tidy` |
| Format | `gofmt -w .` |

## Rules

- Every future business table has `tenant_id`.
- Provider traffic goes through a `ChannelProvider` interface (no hardcoded vendors).
- Backend is the source of truth. Keep business logic in services, not handlers.
- Structured logging, request IDs, graceful shutdown, and `/health` stay on from day 1.
- Swagger annotations stay in sync with handlers.
- Config is environment-based only. No secrets in code.
