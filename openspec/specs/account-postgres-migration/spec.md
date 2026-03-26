# OpenSpec: Account-System Postgres Migration & API Refactor

## Status

Implementation Complete (Core) ✅ — Remaining: `sqlc generate` + repository wrapper, registration webhook

## Context

The `Account-System` previously used Google Cloud Spanner as its primary datastore. As part of moving the identity and authorization stack to a local/Kubernetes-first deployment (`web3-lab`), we are migrating the `Account-System` datastore from Spanner to PostgreSQL. The SpiceDB datastore schema and configuration remain as is, operating via its own PostgreSQL engine.

Simultaneously, we are refactoring the API to use robust Go application patterns (`cobra` and `viper`) and explicitly defining the core identity integration flows.

This migration ensures the API seamlessly integrates with Hydra, Kratos, Oathkeeper, and SpiceDB, while natively supporting wallet connection features (nonce generation, signature verification). Nonces are stored temporarily in Redis, while the final identity mapping (e.g., wallet address) is stored in the Postgres database.

## Documentation References

For detailed API architecture, database schema, and endpoint documentation, refer to the dedicated documentation directory (`documents/api/`):
- [Architecture & Database Schema](../../../documents/api/architecture.md)
- [API Endpoints & Flows](../../../documents/api/endpoints.md)
- [Configuration & Operations](../../../documents/api/operations.md)

## Architecture & Tooling

- **Database**: PostgreSQL 16 (shared Auth DB)
- **Migration Tool**: `golang-migrate/migrate`
- **Go DB Integration**: `sqlc` to generate type-safe Go code, and `pgx/v5` as the underlying driver for superior performance.
- **Go CLI & Configuration**: `cobra` for command-line structure (e.g., `api serve`, `api migrate`) and `viper` for robust configuration management via `config.yaml` and environment variables.
- **Identifiers**: Native `UUID` type in PostgreSQL

## API Required Core Functions

The `Account-System` API serves as the central control plane between Web3 wallets, Auth providers, and OAuth2. Its primary functions are:

1. **Wallet Authentication Flow (Web3 native)**
   - `POST /auth/challenge`: Receive an EOA (wallet address), generate a unique cryptographically-secure nonce, store it in Redis with a TTL, and return it to the client.
   - `POST /auth/verify`: Receive the EOA, signature, and nonce. Verify the signature against the EOA. If valid, locate or provision the user in `account_identities`.
   - **Kratos Bridge**: Upon successful verify, issue a Kratos Session API call to establish a session cookie/token so Kratos recognizes the user as logged in.

2. **OAuth2 Flow Bridge (Hydra Integration)**
   - **Login Endpoint**: Implement the Hydra Login Webhook. Verify if the incoming request already has a valid Kratos session. If yes, automatically accept the login request in Hydra.
   - **Consent Endpoint**: Implement the Hydra Consent Webhook. Present requested scopes to the user (or auto-accept for 1st party clients) and accept the consent request, linking the Kratos identity ID to the Hydra `subject`.

3. **Account & Identity Management (CRUD)**
   - Provide endpoints to link additional Web3 EOAs or Web2 accounts (email/Google) to the same underlying `account_id`.
   - Maintain the `account_identities` table to serve as the single source of truth connecting Kratos UUIDs to wallet addresses or OAuth profiles.

4. **Authorization Checks (SpiceDB Integration)**
   - Intercept and manage relationships (e.g., "Account A owns Resource B") when resources are created, writing them to SpiceDB.
   - (Oathkeeper typically enforces these rules at the ingress level, but the API may have direct gRPC checks for complex logic.)

## Schema Adjustments

The existing 8 Spanner incremental migrations will be consolidated into a single pair of migration files (`000001_init.up.sql` and `000001_init.down.sql`) to establish the clean baseline for PostgreSQL.

**Type Mapping Strategy**:
- `STRING(36)` → `UUID` (used for `account_id`, `identity_id`)
- `STRING(x)` → `VARCHAR(x)`
- `TIMESTAMP OPTIONS(allow_commit_timestamp = true)` → `TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP`
- `JSON` → `JSONB` for better indexing and querying
- `BOOL` → `BOOLEAN`

**Wallet Connect Support**:
- Nonces for signatures are stored in Redis (managed by separate infrastructure).
- External identities (like Ethereum EOAs) are securely mapped inside `account_identities`, using the `JSONB` `attributes` field to store Web3-specific metadata (e.g., `{"eoa_address": "0x123..."}`).

## Implementation Requirements

### 1. Database Migrations
- Create a new directory `migrations/postgres`.
- Write `000001_init.up.sql` containing the consolidated initialization for tables: `accounts`, `identity_providers`, `account_identities`, `account_sessions`, `account_merge_history`, and `account_audit_logs`.
- Setup foreign keys correctly (since Spanner manages relationships differently, PostgreSQL will use standard `REFERENCES`).

### 2. Identity Provider Seed Data
- Seed data for `identity_providers` SHALL be in a dedicated migration (`000003_seed_identity_providers`).
- The seed SHALL use `INSERT ... ON CONFLICT (provider_id) DO NOTHING` so it can be executed multiple times safely (idempotent).
- Default providers: `google` (oidc), `github` (oauth2), `facebook` (oauth2), `apple` (oidc), `email` (email), `eoa` (web3).

### 3. Account Creation on Registration
- When a user registers via **email/password** or **Google OIDC**, a Kratos `after` registration webhook SHALL call the backend API.
- The webhook handler (`POST /api/v1/oauth2/registration-webhook`) SHALL:
  1. Extract the Kratos identity ID and traits from the webhook payload
  2. Determine the `provider_id` (`email` or `google`) from the identity's credentials
  3. Call `AccountService.CreateAccountWithIdentity` to create `accounts` + `account_identities` rows
- When a user registers via **SIWE (Ethereum wallet)**, the existing `auth/verify` flow SHALL continue to create account records directly.

### 4. Database Code Generation (`sqlc`)
- Create `sqlc.yaml` configured for PostgreSQL and `pgx/v5`.
- Write `internal/database/query.sql` holding all necessary read/write queries.
- Generate type-safe Go functions to replace manual DB operations.

### 5. API Refactoring (`cobra` + `viper`)
- **CLI Structure**: Implement `cmd/api/main.go` using `cobra` allowing execution of subcommands like `serve` (start API) and `migrate` (run golang-migrate).
- **Configuration**: Implement `config` package utilizing `viper` to load from `config.yaml` or `viper.AutomaticEnv()`.
- **Repository Implementation**: Replace Spanner SDK references with functions generated by `sqlc`.
- Ensure all Web3 nonce/sign verify flows correctly validate against the unified Postgres store and Redis.
