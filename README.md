# ReadyGeneration Backend

Go backend API for the ReadyGeneration grant intelligence platform.

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.24+ |
| HTTP Framework | Gin |
| Database | PostgreSQL 16 + pgvector |
| DB Driver | pgx v5 + pgxpool |
| Migrations | golang-migrate |
| AI (Narrative) | Anthropic Claude |
| AI (Embeddings) | OpenAI text-embedding-3-small |
| Auth | JWT (HS256) |
| Config | Viper (env vars) |
| Docs | Swagger/OpenAPI via swaggo |

## Architecture

Clean Architecture / DDD:
```
cmd/api/          — entrypoint, wires all dependencies
internal/
  config/         — environment configuration
  db/             — pgxpool setup
  migrate/        — golang-migrate runner
  domain/         — domain models and enums
  repository/     — interfaces + pgx implementations
  scoring/        — grant compatibility scoring engine
  ai/
    claude/       — Anthropic Claude client + narrative generation
    embedding/    — OpenAI embedding service
    rag/          — NOFO chunking + vector retrieval engine
  service/        — application services (auth, grant, scoring, narrative)
  middleware/     — JWT auth, RBAC
  handler/        — Gin HTTP handlers
  router/         — route registration
migrations/       — SQL migration files
scripts/seed/     — database seed data
deploy/k8s/       — Kubernetes manifests (Kustomize)
```

## Quick Start

```bash
# 1. Copy and configure environment
cp .env.example .env
# Edit .env — set DATABASE_URL, JWT_SECRET, ANTHROPIC_API_KEY, OPENAI_API_KEY

# 2. Start dependencies
make docker-up

# 3. Run the API
make run

# 4. View Swagger docs
open http://localhost:8080/swagger/index.html
```

## Docker (full stack)

```bash
make stack-up
```

## Testing

```bash
# Unit tests only (no Docker required)
make test
# or: go test -short ./...

# Integration tests (requires Docker — spins up a real Postgres via testcontainers)
go test ./... -run Integration -v

# All tests (unit + integration)
go test ./...
```

Unit test coverage:
- `internal/scoring` — 6 tests (disqualifiers, tier mapping)
- `internal/ai/rag` — 4 tests (chunking, token estimation)
- `pkg/jwt` — 4 tests (sign, verify, wrong secret, TTL)
- `internal/handler` — 8 tests (input validation, 400 paths)

Integration tests (`internal/repository/pgx/*_integration_test.go`) use
[testcontainers-go](https://github.com/testcontainers/testcontainers-go) and are
skipped automatically in short mode (`-short`).

## Database Migrations

```bash
# Apply
make migrate-up

# Rollback last
make migrate-down

# Create new migration
make migration NAME=add_something
```

## Seed Data

```bash
make seed
```

## Environment Variables

See `.env.example` for all required and optional variables.

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | ✅ | PostgreSQL connection string |
| `JWT_SECRET` | ✅ | HMAC signing secret (min 32 chars) |
| `ANTHROPIC_API_KEY` | For AI narratives | Claude API key |
| `OPENAI_API_KEY` | For embeddings | OpenAI API key |
| `PORT` | No (default 8080) | HTTP listen port |
| `APP_ENV` | No | `development` / `production` |

## API Endpoints

### Public
| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/auth/signup` | Register |
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/reset-password` | Request password reset |
| POST | `/api/v1/leads` | Capture lead (quiz / landing page) |

### Authenticated (JWT required)
| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/auth/me` | Current user |
| POST | `/api/v1/auth/change-password` | Change password |
| GET | `/api/v1/grants` | List active grants |
| GET | `/api/v1/grants/search` | Keyword search |
| GET | `/api/v1/grants/semantic-search` | Vector similarity search |
| GET | `/api/v1/grants/:id` | Get grant |
| POST | `/api/v1/orgs` | Create organization |
| GET | `/api/v1/orgs/:id` | Get organization |
| PATCH | `/api/v1/orgs/:id` | Update organization |
| GET | `/api/v1/orgs/:id/profile` | Get eligibility profile |
| PUT | `/api/v1/orgs/:id/profile` | Create/update profile |
| POST | `/api/v1/orgs/:org_id/applications` | Start tracking an application |
| GET | `/api/v1/orgs/:org_id/applications` | List applications |
| GET | `/api/v1/orgs/:org_id/top-grants` | Top-scored grants |
| POST | `/api/v1/orgs/:org_id/score-all` | Bulk score all grants |
| GET | `/api/v1/orgs/:org_id/grants/:grant_id/score` | Get compatibility score |
| POST | `/api/v1/orgs/:org_id/grants/:grant_id/score` | Compute compatibility score |
| POST | `/api/v1/orgs/:org_id/grants/:grant_id/narratives` | Generate AI narrative |
| GET | `/api/v1/applications/:id` | Get application |
| PATCH | `/api/v1/applications/:id` | Update application |
| PATCH | `/api/v1/applications/:id/status` | Advance pipeline status |
| GET | `/api/v1/applications/:id/activities` | Activity log |
| GET | `/api/v1/applications/:id/narratives` | AI narratives |

### Admin (admin/super_admin role)
| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/admin/grants` | Create grant |
| PATCH | `/api/v1/admin/grants/:id` | Update grant |
| DELETE | `/api/v1/admin/grants/:id` | Archive grant |
| POST | `/api/v1/admin/grants/:id/nofo` | Ingest NOFO document |
| GET | `/api/v1/admin/leads` | List all leads |
| GET | `/api/v1/admin/leads/:id` | Get lead |
| PATCH | `/api/v1/admin/leads/:id` | Update lead |
| POST | `/api/v1/admin/leads/:id/convert` | Convert lead to org |
| GET | `/api/v1/admin/leads/:id/activities` | Lead activity timeline |
| GET | `/api/v1/admin/stats` | Platform aggregate stats |
