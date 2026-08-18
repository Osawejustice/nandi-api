# Nandi API

Go API for Nandi, a multi-tenant customer engagement platform (Africa-first).

Module: [`github.com/Osawejustice/nandi-api`](https://github.com/Osawejustice/nandi-api)

This repository is a modular monolith. The Next.js frontend is a separate product and talks to this API only over HTTP JSON (`/api/v1/...`) and WebSockets.

## What is implemented

The API is past the project skeleton. Auth, domain models, providers, inbox, campaigns, analytics, realtime, and optional AI are in this repo.

- Multi-tenant auth: register (creates tenant + owner), login, refresh, logout, JWT access tokens, hashed API keys
- Roles: `owner`, `admin`, `agent`
- Tenant-scoped domain: tenants, users, contacts, conversations, messages, campaigns, settings, provider logs
- ChannelProvider routing with Africa's Talking (SMS), Evolution (WhatsApp), and a local stub failover
- Inbound webhooks plus a `POST /api/v1/dev/inbound` simulator
- WebSocket hub with Redis Pub/Sub fan-out (`conversation_created`, `new_message`, presence, campaign updates)
- Optional Groq / OpenAI-compatible sentiment and conversation summary (AI failure never blocks persistence)
- Analytics overview, agent presence, and tenant settings
- Swagger UI and a Postman collection

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
internal/models     tenant-scoped domain models
internal/providers  ChannelProvider + Africa's Talking + Evolution + stub
internal/realtime   WebSocket hub + Redis Pub/Sub
internal/ai         sentiment and summary
internal/utils
pkg/                public libraries if needed
migrations/         schema snapshot (AutoMigrate still runs on boot)
docs/               OpenAPI + Postman collection
scripts/
```

Do not add new top-level folders.

## Authentication

Protected routes accept either:

- `Authorization: Bearer <access_token>`
- `X-API-Key: <key>` (owner/admin can mint keys)

Tenant scope always comes from the JWT or API key, never from a client-supplied tenant id.

| Method | Path | Auth | Notes |
| --- | --- | --- | --- |
| POST | `/api/v1/auth/register` | public | Creates tenant + owner |
| POST | `/api/v1/auth/login` | public | Optional `tenant_slug` when email exists in more than one tenant |
| POST | `/api/v1/auth/refresh` | public | Rotates refresh token |
| POST | `/api/v1/auth/logout` | public | Revokes refresh token |
| GET | `/api/v1/me` or `/api/v1/auth/me` | JWT or API key | Current principal |
| POST | `/api/v1/users` | owner/admin | Invite/create a user |
| POST/GET/DELETE | `/api/v1/api-keys` | owner/admin | Create, list, revoke |

## HTTP API

Base path: `/api/v1`. Full request/response shapes live in Swagger.

| Area | Routes |
| --- | --- |
| Health | `GET /health` |
| Contacts | `GET/POST /contacts`, `GET/PATCH/DELETE /contacts/:id` |
| Inbox | `GET /conversations`, `GET/PATCH /conversations/:id`, `POST /conversations/:id/messages`, `POST /conversations/:id/summary` |
| Campaigns | `GET/POST /campaigns`, `GET /campaigns/:id`, `POST /campaigns/:id/start` |
| Analytics | `GET /analytics/overview` |
| Settings | `GET /settings`, `PUT /settings` (write: owner/admin) |
| Agents | `GET /agents`, `POST /agents/me/status` |
| Webhooks | `POST /webhooks/:tenant_slug/sms/africastalking`, `POST /webhooks/:tenant_slug/whatsapp/evolution` |
| Dev | `POST /dev/inbound` (authenticated simulator) |
| Realtime | `GET /ws?access_token=<JWT>` |

## Providers

Core services depend on `ChannelProvider` only. They never import a concrete vendor package.

| Adapter | Channel | Role |
| --- | --- | --- |
| Africa's Talking | SMS | Primary SMS when `AT_USERNAME` / `AT_API_KEY` are set |
| Evolution | WhatsApp | Primary WhatsApp when Evolution env is set |
| Stub | SMS + WhatsApp | Local/demo adapter and configured failover |

Every outbound attempt is written to `provider_logs`.

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
