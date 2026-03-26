# Account API — Architecture & Database Schema

The Account API is a Go/Gin backend service that acts as the central control plane between Web3 wallets, the Ory identity stack (Kratos, Hydra, Oathkeeper), AuthZed SpiceDB, and the shared PostgreSQL datastore. It is deployed as `web3-account-api` in the `web3` namespace.

## High-Level Architecture

```mermaid
graph TB
    subgraph "External"
        GW["APISIX Gateway<br>gateway.web3-local-dev.com/api/*"]
    end

    subgraph "Account API (:8080)"
        direction TB
        R["Gin Router"]
        WA["Wallet Auth<br>/auth/challenge<br>/auth/verify"]
        OA["OAuth2 Hooks<br>/oauth2/login<br>/oauth2/consent"]
        AM["Account CRUD<br>/accounts/*<br>/identities/*"]
        AZ["AuthZ Middleware<br>SpiceDB checks"]
    end

    subgraph "Ory Stack"
        KR["Kratos<br>(Identity)"]
        HY["Hydra<br>(OAuth2)"]
        SP["SpiceDB<br>(Authorization)"]
    end

    subgraph "Datastores"
        PG[(PostgreSQL<br>account_db)]
        RD[(Redis<br>nonce store)]
    end

    GW --> R
    R --> WA
    R --> OA
    R --> AM
    R --> AZ

    WA --> RD
    WA --> KR
    WA --> PG
    OA --> HY
    OA --> KR
    AM --> PG
    AZ --> SP
```

## Database Schema (PostgreSQL)

The Account API owns the `account_db` database on the shared PostgreSQL instance. Schema is managed via `golang-migrate/migrate` with migration files in `Account-System/migrations/postgres/`.

### Entity Relationship Diagram

```mermaid
erDiagram
    accounts ||--o{ account_identities : "has many"
    accounts ||--o{ account_sessions : "has many"
    accounts ||--o{ account_merge_history : "source"
    accounts ||--o{ account_merge_history : "target"
    accounts ||--o{ account_audit_logs : "logged for"
    identity_providers ||--o{ account_identities : "provides"

    accounts {
        UUID account_id PK "grouping for linked identities"
        TIMESTAMPTZ created_at "DEFAULT CURRENT_TIMESTAMP"
        TIMESTAMPTZ updated_at "DEFAULT CURRENT_TIMESTAMP"
        TIMESTAMPTZ last_login_at "nullable"
        VARCHAR(20) status "active | suspended | deleted"
        JSONB metadata "nullable"
    }

    identity_providers {
        VARCHAR(50) provider_id PK
        VARCHAR(100) provider_name
        VARCHAR(50) provider_type "oauth2 | oidc | web3 | email"
        BOOLEAN enabled
        JSONB configuration "nullable"
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    app_clients {
        UUID id PK
        VARCHAR(255) name
        VARCHAR(255) oauth2_client_id
        VARCHAR(255) frontend_url
        VARCHAR(255) login_path
        VARCHAR(255) logout_url
        JSONB allowed_cors_origins
        VARCHAR(255) jwt_secret "nullable"
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    account_identities {
        UUID identity_id PK "SCW derivation salt"
        UUID account_id FK "grouping FK, changed on link"
        UUID kratos_identity_id "nullable, unique"
        VARCHAR(50) provider_id FK
        VARCHAR(255) provider_user_id
        VARCHAR(255) display_name "nullable"
        VARCHAR(500) avatar_url "nullable"
        JSONB attributes "nullable, e.g. eoa_address, email"
        JSONB raw_data "nullable"
        BOOLEAN verified
        BOOLEAN is_primary
        TIMESTAMPTZ linked_at
        TIMESTAMPTZ last_used_at "nullable"
        TIMESTAMPTZ updated_at
        TIMESTAMPTZ unlinked_at "nullable, soft delete"
    }

    account_sessions {
        UUID session_id PK
        UUID account_id FK
        UUID identity_id FK
        UUID kratos_session_id "nullable"
        VARCHAR(45) ip_address "nullable"
        VARCHAR(500) user_agent "nullable"
        TIMESTAMPTZ created_at
        TIMESTAMPTZ expires_at
        TIMESTAMPTZ revoked_at "nullable"
        TIMESTAMPTZ last_activity_at "nullable"
    }

    account_merge_history {
        UUID merge_id PK
        UUID source_account_id FK
        UUID target_account_id FK
        TIMESTAMPTZ merged_at
        UUID merged_by "nullable"
        VARCHAR(500) reason "nullable"
        INTEGER identities_transferred "nullable"
        JSONB metadata "nullable"
    }

    account_audit_logs {
        UUID log_id PK
        UUID account_id "nullable FK"
        VARCHAR(255) identity_id "nullable"
        VARCHAR(50) event_type "LOGIN | LOGOUT | REGISTRATION | ..."
        VARCHAR(20) event_status "SUCCESS | FAILURE | PENDING"
        TEXT event_message "nullable"
        UUID session_id "nullable"
        VARCHAR(255) kratos_session_id "nullable"
        VARCHAR(45) ip_address "nullable"
        VARCHAR(500) user_agent "nullable"
        VARCHAR(50) provider_id "nullable"
        JSONB event_data "nullable"
        TIMESTAMPTZ created_at
    }
```

### Type Mapping (Spanner → PostgreSQL)

| Spanner Type | PostgreSQL Type | Notes |
|---|---|---|
| `STRING(36)` (UUIDs) | `UUID` | Native UUID type with `gen_random_uuid()` |
| `STRING(x)` | `VARCHAR(x)` | Length-constrained strings |
| `STRING(MAX)` | `TEXT` | Unbounded text |
| `TIMESTAMP OPTIONS(allow_commit_timestamp)` | `TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP` | Auto-set on insert |
| `JSON` | `JSONB` | Binary JSON for indexing |
| `BOOL` | `BOOLEAN` | Direct mapping |
| `INT64` | `INTEGER` | Standard integer |

### Key Indexes

| Table | Index | Columns | Notes |
|---|---|---|---|
| `accounts` | `idx_accounts_status` | `status` | Filter by status |
| `account_identities` | `idx_account_identities_provider_user` | `provider_id, provider_user_id` | **UNIQUE** — prevents duplicate provider accounts |
| `account_identities` | `idx_account_identities_kratos_id` | `kratos_identity_id` | **UNIQUE** — one Kratos identity per mapping |
| `account_sessions` | `idx_account_sessions_kratos_session_id` | `kratos_session_id` | Lookup by Kratos session |
| `account_audit_logs` | `idx_account_audit_logs_failed_attempts` | `event_status, event_type, created_at DESC` | Security analysis with `INCLUDE (ip_address, identity_id)` |

## Code Generation (SQLC)

All database queries are defined in `internal/database/query.sql` and compiled by [`sqlc`](https://sqlc.dev/) into type-safe Go code using `pgx/v5`:

```
Account-System/
├── sqlc.yaml                      # SQLC config
├── internal/database/
│   ├── query.sql                  # SQL query definitions
│   ├── postgres/                  # Auto-generated by sqlc
│   │   ├── db.go                  # DBTX interface
│   │   ├── models.go              # Go structs from schema
│   │   └── query.sql.go           # Type-safe query functions
│   ├── repository.go              # AccountRepository interface
│   ├── redis/                     # Nonce storage (unchanged)
│   └── spicedb/                   # Authorization client (unchanged)
```

The generated `Queries` struct is wrapped by a `Repository` implementation that satisfies the existing `AccountRepository` interface, ensuring handler code requires minimal changes.
