## 1. Planning & Spec
- [x] 1.1 Create `app-client-management` specification
- [ ] 1.2 Update implementation plan

## 2. PostgreSQL Migrations
- [x] 2.1 Create up/down migration for `app_clients` table

## 3. SQLC Integration
- [x] 3.1 Write CRUD queries in `backend/internal/database/query/clients.sql`
- [x] 3.2 Run `make sqlc-generate`
- [x] 3.3 Add `AppClientRepository` methods

## 4. Services & Caching
- [x] 4.1 Create `AppClientService` (CRUD logic + Redis sync)
- [x] 4.2 Update `HydraClientService` to support OAuth2 Client Creation via Hydra Admin API
- [x] 4.3 Ensure transaction/rollback if Hydra creation fails during DB creation

## 5. API Handlers
- [x] 5.1 Create `client_handler.go` (POST, PUT, GET, DELETE)
- [x] 5.2 Mount endpoints in `server/router.go` at `/admin/clients`

## 6. Middleware Refactoring
- [x] 6.1 Inject Redis lookup into `oauth2_handler.go` for dynamic redirect URLs
- [x] 6.2 Inject Redis lookup into `CORSMiddleware` for dynamic allowed origins
- [x] 6.3 Cleanup static config definitions in `configmap-config.yaml` and `viper` mappings

## 7. Multi-App SSO (Cross-Domain)

Enable SSO across multiple apps on different domains (e.g. `app.web3-local-dev.com`, `app-2.web3-local-dev.com`) without ConfigMap changes or redeployments.

- [ ] 7.1 Add `frontend_url`, `login_path`, `logout_url` columns to `app_clients` table (migration)
- [ ] 7.2 Register additional Hydra OAuth2 clients per app (via Admin API or `POST /admin/clients`)
- [ ] 7.3 Update `HandleLogin` to resolve `frontend_url` + `login_path` from `app_clients` by `client_id`
- [ ] 7.4 Update `HandleLogout` to resolve `logout_url` from `app_clients` by `client_id`
- [ ] 7.5 Update consent handler to resolve `redirect_uri` from `app_clients` by `client_id`
- [ ] 7.6 Dynamic CORS: aggregate `allowed_cors_origins` from all `app_clients` records (Redis-cached)
- [ ] 7.7 Ingress: add TLS + routing for new app domains (cert-manager, NGINX)
- [ ] 7.8 E2E test: login on app-1 → skip login on app-2 (same Hydra session, different domain)

## 8. Frontend App Configuration

- [ ] 8.1 Each app needs its own runtime config (`gatewayUrl`, `clientId`, `authDomain`)
- [ ] 8.2 Admin API: expose `/api/v1/clients/:id/config` to return frontend runtime config
- [ ] 8.3 Frontend: load config from API instead of hardcoded `__RUNTIME_CONFIG__`
