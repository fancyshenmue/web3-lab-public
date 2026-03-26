# OpenSpec: Cross-Domain Single Sign-On (SSO) with App-2

## Status

In Progress 🔧

## Context

The backend API and Ory ecosystem (Hydra/Kratos) have been updated to support dynamic Application Client Management. To prove the effectiveness of this architecture, we need to deploy a second, distinct frontend application (`app-2`) under a different origin domain (`app.web3-local-dev-2.com`). 

This will demonstrate true **Cross-Domain Single Sign-On (SSO)**:
1. User logs into `app.web3-local-dev.com` (App-1).
2. User navigates to `app.web3-local-dev-2.com` (App-2).
3. App-2 initiates an OAuth2 flow.
4. Hydra recognizes the existing Ory session (established via Oathkeeper/Kratos) and automatically completes the login sequence without requiring the user to re-enter credentials or sign a wallet message again.

## Proposed Architecture

### 1. Frontend App-2 (`frontend/app-2`)

Create a second frontend application. For simplicity and to clearly distinguish it from App-1, duplicate the logic from `frontend/app` but change the styling (e.g., color scheme) and header text to indicate "App 2 / Sub-domain Web3 DApp".

The application must dynamically load its runtime config or use a hardcoded fallback pointing matching its domain but keeping the same gateway:
- `gatewayUrl`: `https://gateway.web3-local-dev.com` (Authentication Gateway)
- `redirectUri`: `https://app.web3-local-dev-2.com/`

### 2. Ingress & TLS

- Add `app.web3-local-dev-2.com` to `/etc/hosts`.
- Update the Kubernetes Ingress configurations (NGINX/APISIX ingress manifests) and `cert-manager` TLS certificates to support the new host `app.web3-local-dev-2.com`.

### 3. Application Client Registration (Database)

We need to register App-2 in the `app_clients` database using the newly built Admin API (`POST /api/v1/admin/clients` or direct SQL seed insert). This action auto-provisions an OAuth2 client in Hydra with the correct `redirect_uris` and CORS origins.

**App-2 Configuration Matrix:**
- **Name**: "Web3 Lab App 2"
- **Frontend URL**: `https://app.web3-local-dev-2.com`
- **Login Path**: `/?login_challenge=`
- **Logout URL**: `https://app.web3-local-dev-2.com/logout`
- **Allowed CORS Origins**: `["https://app.web3-local-dev-2.com"]`

### 4. Verification Flow

- Visit `https://app.web3-local-dev-2.com`.
- Not logged in -> Click "Login via SSO".
- User is bounced to `gateway.web3-local-dev.com/oauth2/auth...`
- Because the browser holds Hydras/Kratos session cookies on `.web3-local-dev.com`, Hydra detects the active session.
- Hydra redirects back to `https://app.web3-local-dev-2.com/?code=...` instantly.
- App-2 consumes the authorization code and completes login.

## Implementation Steps

See [tasks.md](../../changes/sso-cross-domain/tasks.md) for the detailed step-by-step checklist.

## 5. Lessons Learned & Edge Cases

* **SPA OAuth2 Public Clients**: Frontend Single-Page Applications (React, Vite) must be registered in Hydra with `token_endpoint_auth_method: none`. Crucially, the frontend must **not** transmit a `client_secret` parameter during the PKCE `/oauth2/token` exchange (otherwise Hydra throws a `passwords do not match` or `invalid_client` mismatch error).
* **OIDC RP-Initiated Single Logout (SLO)**: To correctly return the user to the initiating application after session destruction, the frontend must attach `id_token_hint` and `post_logout_redirect_uri` to the Hydra `/oauth2/sessions/logout` request. 
* **Auto-Login Edge Case**: The frontend router must implement an explicit `/logout` catch route (redirecting to a signed-out view) to prevent a fallback to the index path (`/`), which could inadvertently auto-trigger a fresh OAuth2 login sequence loop.

## 6. Cross-TLD Expansion (.net Deployment)

To conclusively prove the flexibility of our architecture, the system was expanded to support cross-top-level-domain integrations via `App-3` and `App-4` mapped to `.net` equivalents.

1. **App-3**: `https://app.web3-local-dev.net`
2. **App-4**: `https://app.web3-local-dev-2.net`

**Key configuration requirements for Cross-TLD:**
- Both `.net` and `.com` domains must share the central, identical `gateway.web3-local-dev.com` for their `gatewayUrl` configurations.
- APISIX's `base/apisix-route-public.yaml` CORS origins whitelist MUST include every independent origin URI (e.g. `https://app.web3-local-dev.net`).
- **Database Consistency:** Every new application **must** be registered not just inside the Hydra OAuth2 framework, but natively within the central API's `app_clients` PostgreSQL database so the user-login routing path (`login_path` redirect URL mapping) can be correctly resolved, avoiding fallback JSON returns.
