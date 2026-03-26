## 1. OpenSpec & Planning

- [x] 1.1 Design OpenAPI spec (`openspec/specs/siwe/openapi.yaml`)
- [x] 1.2 Design spec document (`openspec/specs/siwe/spec.md`)
- [x] 1.3 Create `openspec/changes/siwe/tasks.md` (this file)

## 2. Database Migration

- [x] 2.1 Create `migrations/postgres/000004_message_templates.up.sql` (table + FK + seed data)
- [x] 2.2 Create `migrations/postgres/000004_message_templates.down.sql` (rollback)
- [x] 2.3 Create sqlc queries (`backend/internal/database/query/message_templates.sql`)
- [x] 2.4 Run `sqlc generate`
- [x] 2.5 Apply migration to PostgreSQL
- [x] 2.6 Fix SIWE domain in seed data (`web3-local-dev.com` → `app.web3-local-dev.com`)

## 3. Backend — Domain & Repository

- [x] 3.1 Add `MessageTemplate` domain model to `backend/internal/database/repository.go`
- [x] 3.2 Implement `MessageTemplateRepository` in `backend/internal/database/postgres.go`

## 4. Backend — Services

- [x] 4.1 Create `backend/internal/services/message_template_service.go` (CRUD + Redis cache)
- [x] 4.2 Create `backend/internal/services/siwe_service.go` (message gen, parse, verify, authenticate)
- [x] 4.3 Implement `Authenticate()` — Hydra login+consent auto-accept
- [x] 4.4 Implement `autoAcceptConsent()` — follow redirect → extract consent_challenge → accept
- [x] 4.5 Implement `rewriteToInternalURL()` — rewrite external gateway URL to internal Hydra service
- [x] 4.6 Add `PublicURL()` getter to `backend/internal/services/hydra_client_service.go`
- [x] 4.7 Remove Kratos session creation from Authenticate (not supported in Kratos v1.2.0)

## 5. Backend — Handlers & Routing

- [x] 5.1 Create `backend/internal/handlers/siwe_handler.go` (nonce / verify / authenticate)
- [x] 5.2 Create `backend/internal/handlers/message_template_handler.go` (Admin CRUD)
- [x] 5.3 Wire services + handlers in `backend/internal/server/server.go`
- [x] 5.4 Add routes in `backend/internal/server/router.go`

## 6. Backend — Config

- [x] 6.1 Add SIWE fallback config fields in `backend/internal/config/config.go`
- [x] 6.2 Fix default SIWE domain to `app.web3-local-dev.com`

## 7. Kratos Identity Schema

- [x] 7.1 Add `wallet_address` trait (pattern: `^0x[a-fA-F0-9]{40}$`) to Kratos identity schema
- [x] 7.2 Remove `email` from required traits (wallet users don't have email)
- [x] 7.3 Redeploy Kratos with updated ConfigMap

## 8. Frontend — LoginPage

- [x] 8.1 Add SIWE button with MetaMask connection + signing flow
- [x] 8.2 Replace page-load `login/browser?login_challenge=xxx` with `/sessions/whoami` check
- [x] 8.3 Add stale challenge detection → auto-redirect to fresh OAuth2 flow
- [x] 8.4 Persist wallet address in `localStorage('siwe_wallet_address')` after SIWE auth

## 9. Frontend — ProfilePage

- [x] 9.1 Show wallet address, auth method (SIWE/EIP-4361), identity_id for wallet users
- [x] 9.2 Ethereum-themed purple gradient card for wallet users
- [x] 9.3 "Disconnect & Sign Out" button (revokes MetaMask permissions + clears localStorage)
- [x] 9.4 Read wallet address from localStorage (MetaMask loses connection after OAuth2 redirects)

## 10. Build & Deploy

- [x] 10.1 Build + deploy API Docker image
- [x] 10.2 Build + deploy Frontend Docker image
- [x] 10.3 Load images into Minikube, rollout restart

## 11. Documentation

- [x] 11.1 Update `openspec/specs/siwe/spec.md` (status, flow diagram, requirements, backend changes)
- [x] 11.2 Add SIWE + OAuth2 flow to `documents/identity-authorization-stack/authentication-flow.md`
- [x] 11.3 Create `openspec/changes/siwe/tasks.md` (this file)

## 12. Remaining

- [x] 12.1 Unit tests: SIWE message generation, parsing, URL rewriting (`siwe_service_test.go`)
- [x] 12.2 Add "Switch Wallet" feature on profile page
- [x] 12.3 Implement proper EIP-712 typed data verification (currently uses EIP-191 fallback)
- [x] 12.4 Handle EIP-712 typed data in nonce endpoint (`protocol=eip712`)
- [x] 12.5 Add Kratos session creation for SIWE users (Skipped: Not required for Web3 flows)
- [x] 12.6 Admin API for message templates: E2E test
- [x] 12.7 Integration test: full nonce → sign → verify → redirect flow

## Status: ✅ ARCHIVED
