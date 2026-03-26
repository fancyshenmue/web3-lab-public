# APISIX Auth Gateway Architecture

This document describes the architecture for using Apache APISIX as a unified API gateway in front of the Identity & Authorization Stack (Ory Kratos, Hydra, Oathkeeper, AuthZed SpiceDB).

## High-Level Architecture

All external traffic enters through a single TLS-terminated gateway host (`gateway.web3-local-dev.com`). APISIX routes requests by path prefix to internal ClusterIP services, applying plugins (rate limiting, CORS, key-auth, Prometheus) at the gateway layer.

```mermaid
graph TB
    Client((Browser / E2E Test))

    subgraph "TLS Termination (Cert-Manager)"
        ING["NGINX Ingress<br>gateway.web3-local-dev.com"]
    end

    subgraph "API Gateway (APISIX)"
        GW["APISIX Gateway<br>Path-based routing<br>Plugins: rate-limit, cors, key-auth, prometheus"]
    end

    subgraph "Public Endpoints"
        direction LR
        KR_PUB["Kratos Public<br>/identity/* → :4433"]
        HY_PUB["Hydra Public<br>/oauth2/* → :4444"]
        OK["Oathkeeper Proxy<br>/auth/* → :4455"]
        API["Backend API<br>/api/* → :8080"]
    end

    subgraph "Admin Endpoints (key-auth)"
        direction LR
        KR_ADM["Kratos Admin<br>/admin/identity/* → :4434"]
        HY_ADM["Hydra Admin<br>/admin/oauth2/* → :4445"]
        OK_ADM["Oathkeeper API<br>/admin/auth/* → :4456"]
        SP["SpiceDB HTTP<br>/admin/authz/* → :8443"]
    end

    subgraph "Internal Only (ClusterIP)"
        SP_GRPC["SpiceDB gRPC<br>:50051<br>(service-to-service)"]
    end

    subgraph "Datastores"
        PG[(PostgreSQL)]
        RD[(Redis)]
    end

    Client -->|"HTTPS"| ING
    ING --> GW

    GW -->|"rate-limit + cors"| KR_PUB
    GW -->|"rate-limit + cors"| HY_PUB
    GW -->|"rate-limit + cors"| OK
    GW -->|"cors"| API

    GW -->|"key-auth (X-Admin-Key)"| KR_ADM
    GW -->|"key-auth (X-Admin-Key)"| HY_ADM
    GW -->|"key-auth (X-Admin-Key)"| OK_ADM
    GW -->|"key-auth (X-Admin-Key)"| SP

    OK -->|"session check"| KR_PUB
    OK -->|"permission check"| SP_GRPC
    API -->|"nonce storage"| RD
    API -->|"write permissions"| SP_GRPC

    KR_PUB --> PG
    KR_ADM --> PG
    HY_PUB --> PG
    HY_ADM --> PG
    SP --> PG
    SP_GRPC --> PG
    API --> PG
```

## Request Routing

APISIX uses the `proxy-rewrite` plugin to strip path prefixes before forwarding to upstream services.

| Incoming Request                       | Strip Prefix      | Upstream Receives         | Service          |
| -------------------------------------- | ----------------- | ------------------------- | ---------------- |
| `GET /identity/self-service/login`     | `/identity`       | `GET /self-service/login` | Kratos Public    |
| `POST /oauth2/token`                   | `/oauth2`         | `POST /token`             | Hydra Public     |
| `GET /auth/api/v1/users`               | `/auth`           | `GET /api/v1/users`       | Oathkeeper Proxy |
| `GET /api/v1/health`                   | `/api`            | `GET /v1/health`          | Backend API      |
| `GET /admin/identity/admin/identities` | `/admin/identity` | `GET /admin/identities`   | Kratos Admin     |
| `POST /admin/oauth2/admin/clients`     | `/admin/oauth2`   | `POST /admin/clients`     | Hydra Admin      |
| `GET /admin/authz/v1/schema`           | `/admin/authz`    | `GET /v1/schema`          | SpiceDB HTTP     |

## Sequence Diagrams

### Login Flow (Kratos via APISIX)

```mermaid
sequenceDiagram
    actor User as Browser
    participant GW as APISIX Gateway<br>gateway.web3-local-dev.com
    participant KR as Kratos Public<br>:4433

    User->>GW: GET /identity/self-service/login/browser
    Note over GW: Plugin: limit-req (10 rps)<br>Plugin: cors<br>Plugin: proxy-rewrite (strip /identity)
    GW->>KR: GET /self-service/login/browser
    KR-->>GW: 200 OK (Login Flow JSON)
    GW-->>User: 200 OK (Login Flow JSON)

    User->>GW: POST /identity/self-service/login<br>Body: {identifier, password}
    Note over GW: Rate-limit check passes
    GW->>KR: POST /self-service/login<br>Body: {identifier, password}
    KR-->>GW: 200 OK + Set-Cookie: ory_session
    GW-->>User: 200 OK + Set-Cookie: ory_session
```

### OAuth2 Authorization Code Flow (Hydra via APISIX)

```mermaid
sequenceDiagram
    actor User as Browser
    participant GW as APISIX Gateway<br>gateway.web3-local-dev.com
    participant HY as Hydra Public<br>:4444
    participant HY_ADM as Hydra Admin<br>:4445
    participant API as Backend API<br>:8080

    User->>GW: GET /oauth2/oauth2/auth?client_id=...&redirect_uri=...
    Note over GW: Plugin: limit-req (10 rps)<br>Plugin: proxy-rewrite (strip /oauth2)
    GW->>HY: GET /oauth2/auth?client_id=...&redirect_uri=...
    HY-->>GW: 302 Redirect → /identity/self-service/login
    GW-->>User: 302 Redirect → /identity/self-service/login

    Note over User: User completes login (see Login Flow above)

    User->>GW: GET /oauth2/oauth2/auth?login_verifier=...
    GW->>HY: GET /oauth2/auth?login_verifier=...
    HY-->>GW: 302 Redirect → /api/v1/oauth2/consent
    GW-->>User: 302 Redirect → /api/v1/oauth2/consent

    User->>GW: GET /api/v1/oauth2/consent?consent_challenge=...
    GW->>API: GET /v1/oauth2/consent?consent_challenge=...
    API->>GW: PUT /admin/oauth2/admin/oauth2/auth/requests/consent/accept
    Note over GW: Plugin: key-auth (X-Admin-Key)
    GW->>HY_ADM: PUT /admin/oauth2/auth/requests/consent/accept
    HY_ADM-->>GW: 200 OK (redirect_to)
    GW-->>API: 200 OK (redirect_to)
    API-->>GW: 302 Redirect → callback
    GW-->>User: 302 Redirect → callback with ?code=...

    User->>GW: POST /oauth2/oauth2/token (code exchange)
    GW->>HY: POST /oauth2/token
    HY-->>GW: 200 OK {access_token, id_token}
    GW-->>User: 200 OK {access_token, id_token}
```

### Protected API Access (Oathkeeper + SpiceDB)

```mermaid
sequenceDiagram
    actor User as Browser
    participant GW as APISIX Gateway<br>gateway.web3-local-dev.com
    participant OK as Oathkeeper Proxy<br>:4455
    participant KR as Kratos Public<br>:4433
    participant SP as SpiceDB gRPC<br>:50051
    participant API as Backend API<br>:8080

    User->>GW: GET /auth/api/v1/resources<br>Cookie: ory_session=...
    Note over GW: Plugin: limit-req (20 rps)<br>Plugin: cors<br>Plugin: proxy-rewrite (strip /auth)
    GW->>OK: GET /api/v1/resources<br>Cookie: ory_session=...

    OK->>KR: GET /sessions/whoami<br>Cookie: ory_session=...
    KR-->>OK: 200 OK {identity: {id: "user-123"}}

    OK->>SP: CheckPermission(user:user-123, read, resource:*)
    SP-->>OK: PERMISSIONSHIP_HAS_PERMISSION

    OK->>API: GET /api/v1/resources<br>X-User-Id: user-123
    API-->>OK: 200 OK {data: [...]}
    OK-->>GW: 200 OK {data: [...]}
    GW-->>User: 200 OK {data: [...]}
```

### Admin API Access (Key-Auth)

```mermaid
sequenceDiagram
    actor Admin as Admin / E2E Test
    participant GW as APISIX Gateway<br>gateway.web3-local-dev.com
    participant KR_ADM as Kratos Admin<br>:4434

    Admin->>GW: GET /admin/identity/admin/identities<br>(no X-Admin-Key)
    Note over GW: Plugin: key-auth check
    GW-->>Admin: 401 Unauthorized

    Admin->>GW: GET /admin/identity/admin/identities<br>X-Admin-Key: <secret>
    Note over GW: Plugin: key-auth ✓<br>Plugin: proxy-rewrite (strip /admin/identity)
    GW->>KR_ADM: GET /admin/identities
    KR_ADM-->>GW: 200 OK [{identity}, ...]
    GW-->>Admin: 200 OK [{identity}, ...]
```

### Health Path Mapping

Each service has a different health endpoint. The `proxy-rewrite` plugin strips the gateway prefix:

| Gateway Path                         | Rewrites to           | Service        | Port |
| ------------------------------------ | --------------------- | -------------- | ---- |
| `/identity/health/alive`             | `/health/alive`       | kratos-public  | 4433 |
| `/oauth2/health/alive`               | `/health/alive`       | hydra-public   | 4444 |
| `/api/health`                        | `/health`             | web3-api       | 8080 |
| `/admin/identity/admin/health/alive` | `/admin/health/alive` | kratos-admin   | 4434 |
| `/admin/oauth2/health/alive`         | `/health/alive`       | hydra-admin    | 4445 |
| `/admin/auth/health/alive`           | `/health/alive`       | oathkeeper-api | 4456 |

> [!NOTE]
> Kratos admin health is at `/admin/health/alive` (not `/health/alive`). Hydra and Oathkeeper use `/health/alive` on both public and admin ports. The Oathkeeper proxy (port 4455, route `/auth/*`) does **not** have a health endpoint — use `/admin/auth/health/alive` instead.

## Plugin Configuration Summary

### Public Routes

```yaml
plugins:
  - name: limit-req
    enable: true
    config:
      rate: 10 # requests per second
      burst: 5 # burst allowance
      key: remote_addr
      rejected_code: 429
  - name: cors
    enable: true
    config:
      allow_origins: "https://gateway.web3-local-dev.com,http://localhost:3000"
      allow_methods: "GET,POST,PUT,DELETE,OPTIONS"
      allow_headers: "Content-Type,Authorization,X-Session-Token,X-CSRF-Token"
      allow_credential: true
  - name: proxy-rewrite
    enable: true
    config:
      regex_uri: ["^/identity/(.*)", "/$1"] # per-route prefix
  - name: prometheus
    enable: true
    config:
      prefer_name: true
```

### Admin Routes

```yaml
plugins:
  - name: key-auth
    enable: true
    config:
      header: X-Admin-Key
  - name: proxy-rewrite
    enable: true
    config:
      regex_uri: ["^/admin/identity/(.*)", "/$1"] # per-route prefix
  - name: prometheus
    enable: true
    config:
      prefer_name: true
```

## Infrastructure & Installation

### etcd Storage

APISIX uses etcd as its config store. On Minikube, each etcd replica needs a `PersistentVolume` with `hostPath` pinned to a specific node via `nodeAffinity`:

```mermaid
graph LR
    subgraph "web3-lab (node 1)"
        PV0["apisix-etcd-0-pv<br>/data/apisix/etcd-0"]
    end
    subgraph "web3-lab-m02 (node 2)"
        PV1["apisix-etcd-1-pv<br>/data/apisix/etcd-1"]
    end
    subgraph "web3-lab-m03 (node 3)"
        PV2["apisix-etcd-2-pv<br>/data/apisix/etcd-2"]
    end
    PV0 --> E0[apisix-etcd-0]
    PV1 --> E1[apisix-etcd-1]
    PV2 --> E2[apisix-etcd-2]
```

### Helm Values

Key configuration in `deployments/helm/apisix-values.yaml`:

| Setting                                   | Value           | Reason                                            |
| ----------------------------------------- | --------------- | ------------------------------------------------- |
| `gateway.type`                            | `ClusterIP`     | Behind NGINX Ingress, no direct external exposure |
| `etcd.volumePermissions.enabled`          | `true`          | Fixes `/bitnami/etcd/data` permission on hostPath |
| `etcd.volumePermissions.image.repository` | `debian`        | Default `bitnami/os-shell` tag doesn't exist      |
| `etcd.volumePermissions.image.tag`        | `bookworm-slim` | Explicit tag to prevent chart suffix collision    |
| `global.security.allowInsecureImages`     | `true`          | Allows non-Bitnami `debian` image                 |
| `apisix.prometheus.enabled`               | `true`          | Exposes `/apisix/prometheus/metrics`              |

### Cross-Namespace Bridging

The NGINX Ingress and gateway Ingress are in the `apisix` namespace (same as `apisix-gateway` service). ApisixRoute CRDs are in the `web3` namespace. The APISIX Ingress Controller watches all namespaces.

> [!IMPORTANT]
> ExternalName services do **not** work with NGINX Ingress Controller — the internal lua DNS resolver fails to resolve them. The gateway Ingress must be in the same namespace as the `apisix-gateway` Service.

```mermaid
graph LR
    ING["NGINX Ingress<br>apisix namespace"] -->|"apisix-gateway:80"| GW["APISIX Gateway<br>apisix namespace"]
    GW -->|"Routes from ApisixRoute CRDs<br>web3 namespace"| UPSTREAM["Auth Services<br>web3 namespace"]
```

### Post-Install Configuration

APISIX Ingress Controller 2.0+ requires two post-install steps before routes sync:

1. **GatewayProxy CRD** — tells the controller how to connect to the APISIX Admin API
2. **IngressClass patch** — links the IngressClass `apisix` to the GatewayProxy

Both are automated in `make apisix-install`.

### Consumer Configuration

> [!IMPORTANT]
> `ApisixConsumer` CRDs **must** include `ingressClassName: apisix` in the spec. Without it, the controller ignores the consumer during ADC sync, and any manually-created consumers get deleted by the sync cycle.
>
> The consumer is deployed via `make deploy-apisix-auth-gateway` as part of the kustomize apply.

## Migration Checklist

When migrating from per-service NGINX Ingress to unified APISIX gateway:

1. ✅ Install APISIX — `make apisix-install`
2. ✅ Deploy APISIX route manifests — `make deploy-apisix-auth-gateway`
3. ⬜ Update Hydra/Kratos public base URLs to `gateway.web3-local-dev.com` paths
4. ✅ Update `/etc/hosts` to add `127.0.0.1 gateway.web3-local-dev.com`
5. ⬜ Delete old per-service Ingress resources
6. ✅ Run `make tls-setup` to trust new certificate
7. ✅ Verify public endpoints → 503 (routing works, upstream down)
8. ✅ Verify admin key-auth → 401 without key, 503 with key (auth passed)
