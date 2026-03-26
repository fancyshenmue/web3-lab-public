# Web3 Wallet Authentication Flow

Because Web3 applications authenticate users via cryptographic wallet signatures instead of traditional passwords, the backend API must coordinate between the Wallet, Redis (for nonce storage), and Ory Kratos (for session issuance).

## Authentication Sequence Diagram

The following sequence details how a user connects their wallet, signs a specialized challenge message, and obtains a secure HTTP-only session cookie from Ory Kratos.

```mermaid
sequenceDiagram
    autonumber
    actor User as Web Client
    participant API as Web3 Backend API
    participant RD as Redis (Nonce Storage)
    participant KR as Ory Kratos
    participant PG as PostgreSQL (Identity DB)

    Note over User,API: 1. Request Challenge
    User->>API: GET /api/v1/auth/challenge?address=0xABC...
    API->>RD: Store generated Nonce for 0xABC...
    RD-->>API: OK
    API-->>User: Return Nonce Message to Sign

    Note over User,API: 2. Wallet Signature
    User->>User: Wallet (MetaMask) prompts user to sign the Nonce
    User->>API: POST /api/v1/auth/verify (Signature, Address)

    Note over API,RD: 3. Cryptographic Verification
    API->>RD: Retrieve expected Nonce for 0xABC...
    RD-->>API: Nonce details
    API->>API: Verify ecrecover(Signature, Nonce) == Address
    
    alt Signature is Invalid
        API-->>User: 401 Unauthorized
    else Signature is Valid
        Note over API,KR: 4. Identity Management
        API->>KR: Admin API: Get Identity by Traits.wallet_address == 0xABC...
        
        alt Identity does not exist
            KR-->>API: Not Found
            API->>KR: Admin API: Create Identity (wallet_address: 0xABC...)
            KR->>PG: Insert new User
            KR-->>API: New Identity ID
        else Identity exists
            KR-->>API: Existing Identity ID
        end
        
        Note over API,KR: 5. Session Issuance
        API->>KR: Admin API: Create Session for Identity ID
        KR->>PG: Store Session
        KR-->>API: Session Cookie String & Token
        
        API-->>User: 200 OK (Set-Cookie: ory_kratos_session=...)
    end
```

## Email Registration + OAuth2 Auto-Login Flow

When a user signs up via email/password on the LoginPage (which was reached via an OAuth2 login challenge), Kratos creates the identity, establishes a session, fires the registration webhook, and then auto-accepts the Hydra login — redirecting the user directly to the profile page.

```mermaid
sequenceDiagram
    autonumber
    actor User as Browser (LoginPage)
    participant KR as Kratos (Identity)
    participant HY as Hydra (OAuth2)
    participant API as Backend API
    participant PG as PostgreSQL

    Note over User,KR: 1. Login Flow Initialization (on page load)
    User->>KR: GET /self-service/login/browser?login_challenge=xxx
    KR->>KR: Store OAuth2 context in flow
    KR-->>User: Login flow UI (Set-Cookie: csrf, continuity)

    Note over User,KR: 2. User Clicks Sign Up
    User->>KR: GET /self-service/registration/browser
    KR-->>User: Registration flow UI
    User->>KR: POST /self-service/registration (email, password)
    KR->>KR: Validate traits, hash password
    KR->>PG: INSERT identity (Kratos DB)

    Note over KR,HY: 3. After Registration Hooks
    KR->>KR: hook: session → Create Kratos session
    KR->>HY: oauth2_provider: Accept login (subject=identity_id)
    HY-->>KR: Redirect URL to consent endpoint

    Note over KR,API: 4. Registration Webhook
    KR->>API: POST /api/v1/oauth2/registration-webhook
    Note right of KR: Payload: {identity_id, email, provider}
    API->>PG: INSERT accounts + account_identities
    API-->>KR: 200 OK

    Note over User,HY: 5. OAuth2 Completion
    KR-->>User: 422 + redirect_browser_to (Hydra consent URL)
    User->>API: GET /api/v1/oauth2/consent?consent_challenge=yyy
    API->>HY: Accept consent (scopes: openid, offline_access)
    HY-->>User: Redirect → /callback?code=zzz
    User->>HY: POST /oauth2/token (code exchange)
    HY-->>User: {access_token, refresh_token}
    User->>User: Navigate to /profile
```

> [!IMPORTANT]
> **Critical Kratos Registration Hooks Order:**
> ```yaml
> registration.after.password.hooks:
>   - hook: session       # MUST be first — creates ory_kratos_session
>   - hook: web_hook      # Fires registration webhook to backend API
> ```
> Without the `session` hook, Kratos does NOT create a session after registration, the `oauth2_provider` integration cannot auto-accept the Hydra login, and the response is 200 (not 422) with no redirect URL.

> [!NOTE]
> The registration webhook uses **provider-specific Jsonnet templates**: `registration-webhook-email.jsonnet` for password method and `registration-webhook-oidc.jsonnet` for OIDC. This avoids accessing `ctx.identity.credentials` which is not available in the webhook context.

## SIWE (Sign-In with Ethereum) + OAuth2 Flow

When a user authenticates via MetaMask wallet signature, the backend API handles the entire Hydra OAuth2 flow server-side (no Kratos session needed). This is fundamentally different from Email/Google login where Kratos manages the session and auto-accepts the Hydra login.

```mermaid
sequenceDiagram
    autonumber
    actor User as Browser (LoginPage)
    participant API as Backend API
    participant RD as Redis
    participant KR as Kratos (Identity)
    participant HY as Hydra (OAuth2)

    Note over User: Page load: /sessions/whoami check (no Kratos flow binding)
    User->>API: GET /api/v1/siwe/nonce?address=0xABC&protocol=eip712
    API->>RD: Resolve message template (cache/DB)
    API->>RD: SET siwe:nonce:0xabc → nonce
    API-->>User: { nonce, message (EIP-712 TypedData), expires_at }

    User->>User: MetaMask: eth_signTypedData_v4(message)
    User->>API: POST /api/v1/siwe/authenticate { message, signature, login_challenge }

    API->>API: Parse SIWE message, verify signature (ecrecover)
    API->>RD: GETDEL siwe:nonce:0xabc (one-time use)

    alt New wallet user
        API->>KR: POST /admin/identities (wallet_address trait)
        API->>API: Create account + account_identity
    end

    API->>HY: PUT /admin/.../login/accept (login_challenge, subject=identity_id)
    HY-->>API: redirect URL → rewrite to internal Hydra URL
    API->>HY: GET /oauth2/auth (follow redirect, capture consent_challenge)
    API->>HY: PUT /admin/.../consent/accept (grant all scopes)
    HY-->>API: final callback URL (with authorization code)

    API-->>User: { redirect_to: callback URL }
    User->>User: window.location.href = redirect_to
    User->>HY: /callback?code=zzz → exchange for tokens
    User->>User: localStorage: access_token + siwe_wallet_address
    User->>User: Navigate to /profile (shows wallet address)
```

> [!IMPORTANT]
> **Key Differences from Email/Google Login:**
> - SIWE bypasses Kratos session creation — `AcceptLoginRequest` uses identity_id directly as the Hydra subject.
> - The `login_challenge` must NOT be bound to a Kratos flow before SIWE uses it (page load uses `/sessions/whoami` instead of `/login/browser?login_challenge=xxx`).
> - The redirect URL from `AcceptLoginRequest` uses the external gateway hostname, which is rewritten to the internal Hydra service URL via `rewriteToInternalURL()`.

## Email Sign-In + OAuth2 Flow

When a user signs in with email/password on the LoginPage, Kratos validates credentials and auto-accepts the Hydra login via `oauth2_provider`. The response is HTTP 422 with `redirect_browser_to`.

```mermaid
sequenceDiagram
    autonumber
    actor User as Browser (LoginPage)
    participant KR as Kratos (Identity)
    participant HY as Hydra (OAuth2)
    participant API as Backend API

    Note over User,KR: 1. Login Flow (created on page load with login_challenge)
    User->>KR: POST /self-service/login (identifier, password)
    KR->>KR: Validate credentials

    alt Invalid Credentials
        KR-->>User: 400 + error messages
    else Valid Credentials
        Note over KR,HY: 2. OAuth2 Auto-Accept
        KR->>HY: Accept login (subject=identity_id)
        HY-->>KR: Redirect URL
        KR-->>User: 422 + redirect_browser_to

        Note over User,API: 3. Complete OAuth2 Flow
        User->>API: GET /api/v1/oauth2/consent?consent_challenge=yyy
        API->>HY: Accept consent
        HY-->>User: Redirect → /callback?code=zzz
        User->>HY: POST /oauth2/token
        HY-->>User: {access_token, refresh_token}
        User->>User: Navigate to /profile
    end
```

> [!NOTE]
> The frontend checks for `redirect_browser_to` in the response body **BEFORE** checking `submitRes.ok`. Kratos returns HTTP 422 (not 200) for successful auth when `oauth2_provider` is configured, so the `ok` check would fail if checked first.


## API Access Control Flow (Oathkeeper + SpiceDB)

Once the user has a Kratos Session Cookie, they can call protected APIs through Oathkeeper.

```mermaid
sequenceDiagram
    actor User as Web Client
    participant OK as Oathkeeper Proxy
    participant KR as Kratos (Who API)
    participant SP as SpiceDB (AuthZ)
    participant Svc as Protected Service

    User->>OK: Request /protected/resource
    Note right of User: Headers include Kratos Session Cookie

    OK->>KR: Authenticator: Validate Session Cookie
    alt Invalid Session
        KR-->>OK: 401 Unauthorized
        OK-->>User: 401 Unauthorized
    else Valid Session
        KR-->>OK: 200 OK (Returns Subject/User ID)
    end

    OK->>SP: Authorizer: Check Permission (User ID, "read", "resource:123")
    alt Denied
        SP-->>OK: Access Denied
        OK-->>User: 403 Forbidden
    else Allowed
        SP-->>OK: Access Granted
    end

    OK->>Svc: Mutator: Forward Request (Headers injected with X-User-ID)
    Svc-->>OK: Service Response
    OK-->>User: Service Response
```

## Google OIDC + OAuth2 Login Flow

When a user clicks "Sign in via OAuth2" → "Google", the following multi-service redirect chain authenticates them and issues an OAuth2 access token.

```mermaid
sequenceDiagram
    autonumber
    actor User as Browser (app.web3-local-dev.com)
    participant HY as Hydra (OAuth2)
    participant API as Backend API
    participant KR as Kratos (Identity)
    participant Google as Google OIDC

    Note over User,HY: 1. Start OAuth2 Flow
    User->>HY: GET /oauth2/auth?client_id=...&redirect_uri=.../callback&scope=openid
    HY->>API: Redirect → /api/v1/oauth2/login?login_challenge=xxx

    Note over API,KR: 2. Login Challenge → Kratos Login UI
    API->>HY: Get login request (admin API)
    API->>User: Redirect → app.web3-local-dev.com/login?login_challenge=xxx

    Note over User,Google: 3. User Clicks Google
    User->>KR: POST /identity/self-service/login (method=oidc, provider=google)
    KR->>Google: Redirect → accounts.google.com/o/oauth2/v2/auth
    Google->>User: Show "Choose an account" consent
    User->>Google: Select account
    Google->>KR: Redirect → /identity/self-service/methods/oidc/callback/google?code=xxx

    Note over KR,HY: 4. Kratos Accepts Login
    KR->>KR: Create/update identity from Google profile
    KR->>HY: Accept login challenge (admin API, subject=identity_id)
    HY->>API: Redirect → /api/v1/oauth2/consent?consent_challenge=yyy

    Note over API,HY: 5. Auto-Accept Consent
    API->>HY: Get consent request (admin API)
    API->>HY: Accept consent (grant scopes: openid, offline_access)
    HY->>User: Redirect → app.web3-local-dev.com/callback?code=zzz

    Note over User,HY: 6. Token Exchange
    User->>HY: POST /oauth2/token (code=zzz, grant_type=authorization_code)
    HY-->>User: {access_token, refresh_token, id_token}

    Note over User: 7. Profile Page
    User->>User: Store access_token in localStorage
    User->>HY: GET /userinfo (Authorization: Bearer access_token)
    HY-->>User: {sub, email, name, picture}
    User->>User: Navigate to /profile
```

> [!IMPORTANT]
> **Key Configuration Gotchas:**
> - Hydra `URLS_SELF_ISSUER` must be `https://gateway.web3-local-dev.com` (without `/oauth2` suffix) — otherwise Hydra generates doubled paths like `/oauth2/oauth2/auth`
> - Kratos OIDC credentials must use `SELFSERVICE_METHODS_OIDC_CONFIG_PROVIDERS` env var (full JSON array) — the `env://` syntax does NOT work inside nested OIDC provider configs
> - Hydra serves `/userinfo` at root path, NOT under `/oauth2/` — requires a separate APISIX route
> - APISIX CORS `allow_origins` must include `https://app.web3-local-dev.com` for frontend cross-origin requests
> - The backend consent handler auto-accepts all consent for first-party clients (no consent screen)
> - Kratos registration hooks MUST include `- hook: session` before `- hook: web_hook` — without it, no session is created and `oauth2_provider` cannot auto-accept the Hydra login
> - Kratos returns HTTP 422 (not 200) for successful auth when `oauth2_provider` is configured — the frontend must check `redirect_browser_to` before `submitRes.ok`
> - The backend `HandleLogin` checks for Kratos session via `/sessions/whoami` (forwarding cookies) when Hydra `skip=false` — this enables auto-login for users who already have a Kratos session

The flow described above routes through the **APISIX gateway** at `gateway.web3-local-dev.com`, which performs consumer-based API key authentication before proxying to the backend endpoints.

## SIWE (EIP-4361) Authentication Flow

> [!NOTE]
> The SIWE flow replaces the legacy `/api/v1/auth/challenge` and `/api/v1/auth/verify` endpoints with a standardized EIP-4361 message format. See the full spec at [`openspec/specs/siwe/spec.md`](../../openspec/specs/siwe/spec.md).

### Endpoints

| Method | Path                        | Purpose                                    |
| :----- | :-------------------------- | :----------------------------------------- |
| GET    | `/api/v1/siwe/nonce`        | Generate nonce + EIP-4361 formatted message |
| POST   | `/api/v1/siwe/verify`       | Verify → Kratos session (standalone)       |
| POST   | `/api/v1/siwe/authenticate` | Verify → Hydra OAuth2 redirect             |

### Key Differences from Legacy Flow

| Feature            | Legacy (`/auth/challenge`)  | SIWE (`/siwe/nonce`)                       |
| :----------------- | :-------------------------- | :----------------------------------------- |
| Message format     | Custom plain text           | EIP-4361 standard (human-readable in wallet) |
| Protocol support   | EIP-191 only                | EIP-191 (SIWE) + EIP-712                   |
| Message config     | Hardcoded in config.go      | PostgreSQL `message_templates` + Redis cache |
| OAuth2 integration | Separate `/oauth2/login`    | Built into `/siwe/authenticate`            |
| Client-specific    | No                          | Yes (via `app_clients.message_template_id`) |

### Dual-Protocol Support

- **SIWE (EIP-4361)**: Uses `personal_sign` (EIP-191). The wallet displays a human-readable message conforming to the SIWE ABNF format.
- **EIP-712**: Uses `eth_signTypedData_v4`. The wallet displays structured typed data with a domain separator.

The `protocol` query parameter on `/api/v1/siwe/nonce` determines which format is generated. Sign-in message templates are stored in the `message_templates` table and cached in Redis with a 1-hour TTL.

## OAuth2 Logout Flow

When the user clicks "Sign Out" on the Profile page, the frontend clears localStorage and redirects to Hydra's logout endpoint. The backend receives the logout challenge, revokes all Kratos sessions (to prevent auto-re-login), and accepts the Hydra logout.

```mermaid
sequenceDiagram
    autonumber
    actor User as Browser (ProfilePage)
    participant HY as Hydra (OAuth2)
    participant API as Backend API
    participant KR as Kratos (Identity)

    Note over User: 1. Frontend Cleanup
    User->>User: Clear localStorage (access_token, id_token)
    User->>HY: GET /oauth2/sessions/logout?id_token_hint=...&post_logout_redirect_uri=/logout

    Note over HY,API: 2. Logout Challenge
    HY->>API: Redirect → /api/v1/oauth2/logout?logout_challenge=xxx
    API->>HY: Get logout request (admin API) → extract subject
    API->>KR: DELETE /admin/identities/{subject}/sessions (revoke all sessions globally)
    KR-->>API: 204 No Content
    API->>HY: Accept logout (admin API)

    Note over User: 3. Post-Logout Redirect
    HY->>User: Redirect → post_logout_redirect_uri (app.web3-local-dev.com/logout)
    User->>User: React Router catches /logout → redirects to /?logout=true
    User->>User: HomePage detects ?logout=true → displays manual "Sign In" button
```

> [!IMPORTANT]
> **Logout Must Revoke Kratos Sessions:**
> Hydra's logout only kills the Hydra/OAuth2 session. The `ory_kratos_session` cookie remains valid in the browser. Without revoking Kratos sessions in `HandleLogout`, the next OAuth2 flow (triggered by HomePage auto-redirect) would find the valid Kratos session via `HandleLogin` → auto-accept → user is immediately re-authenticated in a loop.

> [!NOTE]
> The `?logout=true` query parameter prevents the HomePage from auto-triggering the OAuth2 flow. Without it, the HomePage's `useEffect` immediately starts a new OAuth2 flow, which — if Kratos sessions were not revoked — would re-authenticate the user instantly.

## Cross-Domain & Cross-TLD Single Sign-On (SSO) Flow

Because Kratos securely manages the root identity session via HTTP-only cookies tied to the central gateway domain (`.web3-local-dev.com`), **any** frontend application—even those on entirely different top-level domains (e.g., `.net`)—can achieve instantaneous SSO without requiring the user to sign a new wallet message or enter a password.

The cookie is strictly bound to the central authentication provider. When the user initiates a login from `app.web3-local-dev.net` (App-3), their browser is redirected to the central gateway, which seamlessly attaches the existing session cookie.

```mermaid
sequenceDiagram
    autonumber
    actor User as Browser
    participant App2 as App 2 (app.web3-local-dev-2)
    participant HY as Hydra (OAuth2)
    participant API as Backend API
    
    Note over User,HY: 1. Unauthenticated Visit
    User->>App2: Visit Home Page
    App2->>HY: GET /oauth2/auth (client_id=app-2, redirect_uri=app-2/callback)
    
    Note over HY,API: 2. Check Root Session
    HY->>API: Redirect → /api/v1/oauth2/login?login_challenge=xxx
    API->>HY: GET /admin/oauth2/auth/requests/login
    Note right of API: API forwards User cookies to Kratos /sessions/whoami
    API->>API: Valid ory_kratos_session found!
    
    Note over API,HY: 3. Auto-Accept Login
    API->>HY: PUT /admin/oauth2/auth/requests/login/accept (subject=identity_id)
    HY->>API: Redirect URL (Internal Consent)
    
    API->>HY: GET /oauth2/auth (follow redirect to consent)
    API->>HY: PUT /admin/oauth2/auth/requests/consent/accept
    HY-->>API: Final callback URL (app-2/callback?code=zzz)
    
    Note over User,App2: 4. Instant Token Exchange
    API-->>User: 302 Redirect → app.web3-local-dev-2.com/callback?code=zzz
    User->>App2: React app loads callback
    App2->>HY: POST /oauth2/token (authorization_code)
    App2->>User: Save access_token, id_token → Navigate to /profile
```

> [!TIP]
> **Public Client Considerations**
> Since `App-2` is a Single Page Application (SPA), it must be registered as a Public Client (`token_endpoint_auth_method: none`) in Hydra. The frontend must **not** send a `client_secret` during the token code exchange.

> [!IMPORTANT]
> **Database Registry Requirement for Cross-Domain Clients:**
> Registering a new frontend application purely in Hydra (via `curl -X POST /admin/clients`) is **not enough**. 
> The central Web3 Account API intercepts all login challenges from Hydra to determine where to redirect the user for the UI. It dynamically looks up the frontend URLs using the `oauth2_client_id`. 
> Therefore, you **must** also insert the application's configuration metadata into the core PostgreSQL `app_clients` table (which is then cached in Redis), otherwise the API will fallback to vomiting raw JSON payloads instead of redirecting the user to a polished login UI.

