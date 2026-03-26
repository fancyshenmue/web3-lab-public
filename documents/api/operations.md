# Account API — Configuration & Operations

## CLI Structure (cobra)

The API binary uses `cobra` for subcommands:

```bash
# Start the HTTP server
api serve [--port 8080] [--config ./config.yaml]

# Run database migrations
api migrate up    [--dsn postgres://...] [--path ./migrations/postgres]
api migrate down  [--dsn postgres://...] [--steps 1]
api migrate status [--dsn postgres://...]
```

## Configuration (viper)

Configuration is loaded from `config.yaml` with environment variable overrides via `viper.AutomaticEnv()`. Environment variables take precedence over the YAML file.

### config.yaml

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  gin_mode: "release" # debug | release | test
  environment: "minikube"

database:
  postgres_dsn: "${POSTGRES_DSN}"

redis:
  host: "${REDIS_HOST:-redis-service}"
  port: 6379
  password: ""
  database: 0
  pool_size: 10
  key_prefix: "web3"
  nonce_key_prefix: "nonce"
  nonce_ttl_minutes: 5

auth:
  kratos_public_url: "http://kratos-public:4433"
  kratos_admin_url: "http://kratos-admin:4434"
  hydra_public_url: "http://hydra-public:4444"
  hydra_admin_url: "http://hydra-admin:4445"
  wallet_auth_domain: "gateway.web3-local-dev.com"
  wallet_auth_version: "1"
  wallet_auth_chain_id: 1
  cors:
    allowed_origins:
      - "https://gateway.web3-local-dev.com"
      - "http://localhost:3000"
    allow_all: false

spicedb:
  endpoint: "spicedb-grpc:50051"
  token: "${SPICEDB_GRPC_PRESHARED_KEY}"
  insecure: true
```

### Required Environment Variables

| Variable                     | Description                  | Example                                                                   |
| ---------------------------- | ---------------------------- | ------------------------------------------------------------------------- |
| `POSTGRES_DSN`               | PostgreSQL connection string | `postgres://postgres:postgres@auth-postgres:5432/account?sslmode=disable` |
| `SPICEDB_GRPC_PRESHARED_KEY` | SpiceDB pre-shared key       | `web3-spicedb-key`                                                        |

### Optional Environment Variables

| Variable         | Default         | Description        |
| ---------------- | --------------- | ------------------ |
| `REDIS_HOST`     | `redis-service` | Redis hostname     |
| `REDIS_PASSWORD` | _(empty)_       | Redis password     |
| `SERVER_PORT`    | `8080`          | HTTP listen port   |
| `GIN_MODE`       | `release`       | Gin framework mode |

## Project Layout

```
web3-lab/
├── backend/
│   ├── cmd/api/main.go                    # Cobra root + serve/migrate commands
│   ├── internal/
│   │   ├── config/config.go               # Viper config loader
│   │   ├── database/
│   │   │   ├── repository.go              # AccountRepository interface + domain models
│   │   │   ├── query/                     # SQLC query definitions (5 .sql files)
│   │   │   └── sqlc/                      # Auto-generated (sqlc generate)
│   │   ├── handlers/                      # Gin route handlers (6 files)
│   │   ├── services/                      # Business logic (7 files)
│   │   └── server/                        # HTTP server, router, middleware
│   ├── pkg/logs/logger.go                 # Zap structured logger
│   ├── sqlc.yaml
│   ├── go.mod
│   └── go.sum
├── migrations/postgres/                   # golang-migrate files
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── deployments/
│   ├── build/api/Dockerfile               # Multi-stage Docker build
│   └── kustomize/api/                     # K8s manifests (base + minikube overlay)
└── documents/api/                         # API documentation
```

## Kubernetes Deployment

The API is deployed via Kustomize in the `web3` namespace. See the [identity-authorization-stack spec](../../openspec/specs/identity-authorization-stack/spec.md) for full Kustomize structure.

### Build & Load Image

```bash
# Build the Docker image
make build-api

# Load into Minikube
make load-api
```

### Deploy

```bash
make deploy-api
```

### Full Build + Deploy Pipeline

```bash
make bump-patch && \
make build-api && \
make load-api && \
make deploy-api
```

### ConfigMap Environment Variables

The Minikube overlay injects these via `configmap-env.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: web3-api-env
  namespace: web3
data:
  POSTGRES_DSN: "postgres://postgres:postgres@auth-postgres.web3.svc.cluster.local:5432/account?sslmode=disable"
  REDIS_HOST: "redis-service.web3.svc.cluster.local"
  GIN_MODE: "debug"
```

Secrets (SpiceDB key) are applied separately via `kubectl create secret`.

## Database Migrations

Migrations run via Makefile targets or the `api migrate` command.

### Makefile Targets (recommended)

```bash
# Requires: make port-forward-auth-postgres (port 5434)
make migrate-up         # Run all pending migrations
make migrate-down       # Rollback last migration
make migrate-status     # Show current version
make migrate-create NAME=add_foo  # Create new migration files
```

Default DSN: `postgres://postgres:postgres@127.0.0.1:5434/account?sslmode=disable`
Override: `MIGRATE_DSN=... make migrate-up`

### CLI (in-cluster / Docker)

```bash
api migrate up --dsn "$POSTGRES_DSN" --path ./migrations/postgres
api migrate down --dsn "$POSTGRES_DSN" --steps 1
api migrate status --dsn "$POSTGRES_DSN"
```

In Kubernetes, the API deployment includes an init container that runs migrations before the main container starts:

```yaml
initContainers:
  - name: migrate
    image: web3-account-api:latest
    command: ["api", "migrate", "up"]
    env:
      - name: POSTGRES_DSN
        valueFrom:
          secretKeyRef:
            name: account-db-secret
            key: dsn
```

## Health Checks

| Endpoint            | Method | Description                               |
| ------------------- | ------ | ----------------------------------------- |
| `/api/health`       | `GET`  | Returns `200 OK` if the server is running |
| `/api/health/ready` | `GET`  | Checks Postgres + Redis connectivity      |

### Kubernetes Probes

Both probes point to `/api/health` (not `/api/health/ready` for readiness to avoid dependency failures blocking startup):

```yaml
readinessProbe:
  httpGet:
    path: /api/health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
livenessProbe:
  httpGet:
    path: /api/health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 30
```

## Verification Commands

```bash
# In-pod (most reliable, no port-forward needed)
kubectl -n web3 exec deploy/web3-api -- wget -qO- http://localhost:8080/api/health
kubectl -n web3 exec deploy/web3-api -- wget -qO- http://localhost:8080/api/health/ready
kubectl -n web3 exec deploy/web3-api -- wget -qO- "http://localhost:8080/api/v1/auth/challenge?address=0x742d35Cc6634C0532925a3b844Bc9e7595f2bD18"

# Via port-forward
kubectl port-forward svc/web3-api 8080:8080 -n web3
curl http://localhost:8080/api/health

# Via APISIX gateway (requires ApisixRoute for /api/* → web3-api:8080)
curl -sk https://gateway.web3-local-dev.com/api/health

# App Client Administration (requires X-Admin-Key)
curl -sk -H "X-Admin-Key: web3-admin-secret-key" https://gateway.web3-local-dev.com/api/v1/admin/clients
```

### IDE REST Client Extension

A fully interactive HTTP test suite is located at: `documents/api/app_clients.http`

To use it:

1. Ensure your IDE supports `.http` files. (In VSCode, install the **REST Client** extension by Huachao Mao. JetBrains IDEs support it natively.)
2. Open the file and click the hover-over `Send Request` links to dynamically test all API routes against your live cluster.

### Tested Results (v0.0.5)

| Endpoint                                   | Status | Response                                                                    |
| ------------------------------------------ | ------ | --------------------------------------------------------------------------- |
| `GET /api/health`                          | ✅ 200 | `{"status":"ok","service":"web3-account-api","environment":"minikube"}`     |
| `GET /api/health/ready`                    | ✅ 200 | `{"components":{"postgres":"connected","redis":"connected"},"status":"ok"}` |
| `GET /api/v1/auth/challenge?address=0x...` | ✅ 200 | Nonce + SIWE message                                                        |
| `GET /api/v1/authz/health`                 | 404    | Expected — SpiceDB not configured                                           |

## Dependencies

| Dependency                             | Purpose                              |
| -------------------------------------- | ------------------------------------ |
| `github.com/jackc/pgx/v5`              | PostgreSQL driver                    |
| `github.com/golang-migrate/migrate/v4` | Database migrations                  |
| `github.com/sqlc-dev/sqlc`             | SQL → Go code generation (dev tool)  |
| `github.com/spf13/cobra`               | CLI framework                        |
| `github.com/spf13/viper`               | Configuration management             |
| `github.com/gin-gonic/gin`             | HTTP framework                       |
| `github.com/redis/go-redis/v9`         | Redis client                         |
| `github.com/authzed/authzed-go`        | SpiceDB gRPC client                  |
| `github.com/ory/client-go`             | Kratos admin client                  |
| `github.com/ory/hydra-client-go/v2`    | Hydra admin client                   |
| `github.com/ethereum/go-ethereum`      | Signature verification (`ecrecover`) |
