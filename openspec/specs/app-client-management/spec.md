# App Client Management Specification

## Status

- [ ] Draft
- [ ] Proposed
- [x] Approved
- [ ] Implementation In Progress
- [x] Implementation Complete (Core CRUD + Dynamic Routing)
- [ ] Multi-App SSO (Planned)

## Objective

Migrate static, configmap-based OAuth2 client and gateway mappings (CORS, JWT, Frontend redirect URLs) into a dynamic, database-backed configuration system. This allows the DIVER platform to seamlessly register new multi-tenant applications (clients) without requiring Kubernetes redeployments.

## Problem Statement

In the origin implementation, multi-tenant app routing logic and CORS/JWT mappings were hardcoded in `configmap-config.yaml` (`frontend_url_mappings`, `login_path_mappings`, `cors`, `jwt_frontend_mappings`).
As the platform scales to handle dynamic applications (e.g. NMS Checkout, 3rd party DApps), configuring these statically is no longer viable.

## Proposed Architecture

### 1. Database Layer (PostgreSQL)

Introduce a new table `app_clients` in the `account` database:

- `id` (UUID, Primary Key)
- `name` (String, e.g., "DIVER Exchange")
- `oauth2_client_id` (String, links to Hydra)
- `frontend_url` (String)
- `login_path` (String)
- `logout_url` (String)
- `allowed_cors_origins` (JSONB)
- `jwt_secret` (String, optional)
- `created_at` / `updated_at`

### 2. Caching Layer (Redis)

To prevent constant DB lookups during high-traffic authentication flows, configurations map to Redis:

- **Key**: `web3:client:{oauth2_client_id}`
- **Value**: JSON containing frontend mapping & cors configs.
- Cache is invalidated/updated via the App Client Management API on any DB write.

### 3. API Layer

Add a new RESTful API domain `/api/v1/clients`:

- `POST /api/v1/clients` - Create a new app client
- `PUT /api/v1/clients/:id` - Update existing client
- `GET /api/v1/clients` - List clients
- `DELETE /api/v1/clients/:id` - Delete client

**Lifecycle hooks**:

- **Creation**: When `POST /api/v1/clients` is called, the API uses the Go Hydra Client (`hydra_client_service.go`) to call Hydra's Admin API (`POST /admin/clients`). This auto-provisions an OAuth2 Client with the given `redirect_uris` and `cors` rules, linking the generated `client_id` back to our `oauth2_client_id` column.
- **Update**: When `PUT /api/v1/clients/:id` is called (e.g. to change the frontend URL or allowed origins), the API calls Hydra's Admin API (`PUT /admin/clients/:id`) to sync the new OAuth2 settings (like `redirect_uris` or `allowed_cors_origins`) to Hydra.
- **Deletion**: When `DELETE /api/v1/clients/:id` is called, it deletes the Postgres record, the Redis cache, and invokes Hydra Admin API to delete the OAuth2 Client.

### 4. Integration Points

- **OAuth2 Login Redirect**: The `oauth2_handler.go` (`GET /oauth2/login`) will read the `client_id` from Hydra's login challenge, query the App Client configuration from Redis, and redirect the user to the dynamically resolved `frontend_url` + `login_path`.
- **CORS Middleware**: The API CORS middleware will dynamically fetch the aggregated `allowed_cors_origins` from Redis rather than a static config map.
- **JWT Middleware / APISIX**: For JWT validation, the system can expose an internal endpoint or sync these JWT secrets to APISIX's `jwt-auth` consumer configurations via APISIX Admin API.

## Security Considerations

- The API to manage clients (`/api/v1/clients/*`) must be strictly protected by SpiceDB, restricted to `admin` level access only.
- Redis cache must have proper TTL or strict invalidation patterns to prevent routing discrepancies during client config updates.
- Auto-generated Hydra OAuth2 Secrets must only be returned once upon creation and subsequently hashed/hidden.
