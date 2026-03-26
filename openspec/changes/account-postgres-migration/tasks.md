## 1. OpenSpec Definition
- [x] 1.1 Create `openspec/specs/account-postgres-migration/spec.md`
- [x] 1.2 Create `openspec/changes/account-postgres-migration/tasks.md`
- [x] 1.3 Update spec with Documentation References section

## 2. Documentation
- [x] 2.1 Create `documents/api/architecture.md` (ER diagram, schema, type mappings, SQLC)
- [x] 2.2 Create `documents/api/endpoints.md` (route table, auth flows, error format)
- [x] 2.3 Create `documents/api/operations.md` (CLI, config, K8s deployment, health checks)
- [x] 2.4 Updated `operations.md` with correct project layout, DSN, Makefile targets

## 3. PostgreSQL Schema Initialization
- [x] 3.1 Consolidated Spanner schema into `migrations/postgres/000001_init.up.sql`
- [x] 3.2 Created matching `migrations/postgres/000001_init.down.sql`
- [x] 3.3 Ran `make migrate-up` — 6 tables + seed data applied to Minikube Postgres

## 4. Go Backend - SQLC Integration
- [x] 4.1 Created `backend/sqlc.yaml` for Postgres and `pgx/v5`
- [x] 4.2 Wrote SQL query definitions in `backend/internal/database/query/` (5 files)
- [ ] 4.3 Run `make sqlc-generate` to create the database package
- [ ] 4.4 Implement repository wrapper (`internal/database/postgres.go`)

## 5. Go Backend - Full API
- [x] 5.1 Created `backend/cmd/api/main.go` with `cobra` commands (`serve`, `migrate up/down/status`)
- [x] 5.2 Created `backend/internal/config/config.go` with `viper`
- [x] 5.3 Created services (7): nonce, auth, account, wallet_auth, kratos, hydra, authz
- [x] 5.4 Created handlers (6): auth, account, oauth2, authz, health, helpers
- [x] 5.5 Created server layer: server.go, router.go, middleware.go
- [x] 5.6 Created `backend/pkg/logs/logger.go`
- [x] 5.7 Created `backend/internal/database/repository.go` (interface + domain models)

## 6. Infrastructure & Tooling
- [x] 6.1 Installed `golang-migrate` and `sqlc` via pixi tasks
- [x] 6.2 Added Makefile targets: `migrate-up/down/status/create`, `sqlc-generate`
- [x] 6.3 Fixed Dockerfile CMD to `["/app/api", "serve"]`
- [x] 6.4 Fixed deployment health probes to `/api/health`
- [x] 6.5 Fixed `configmap-env.yaml` env var `DATABASE_DSN` → `POSTGRES_DSN`
- [x] 6.6 Added `kustomize-update-api` target (auto-updates image tag on deploy)
- [x] 6.7 Created `migrations/spicedb/schema.zed` and added `make spicedb-schema` / `make spicedb-verify`

## 7. Deployment & Verification
- [x] 7.1 `go mod tidy` — all dependencies resolved
- [x] 7.2 `CGO_ENABLED=0 go build ./cmd/api/...` — compiles successfully
- [x] 7.3 `make migrate-up` — migration applied to Minikube Postgres
- [x] 7.4 `make build-api && make load-api && make deploy-api` — deployed v0.0.5
- [x] 7.5 `GET /api/health` → 200 `{"status":"ok"}`
- [x] 7.6 `GET /api/health/ready` → 200 (Postgres: connected, Redis: connected)
- [x] 7.7 `GET /api/v1/auth/challenge` → 200 (nonce + SIWE message)
- [x] 7.8 Run `make sqlc-generate`
- [x] 7.9 Add APISIX route for `/api/*` → `web3-api:8080`
- [x] 7.10 Add `WALLET_AUTH_DOMAIN` / `WALLET_AUTH_VERSION` to configmap-env
