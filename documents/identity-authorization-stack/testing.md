# Identity & Authorization Testing Guide

This guide outlines how to verify the health and functional status of the Web3 Identity & Authorization stack running locally on Minikube.

## Pre-requisites

Before executing these tests, ensure your local development environment is bridged properly:

1. **Minikube is running**: `make minikube-start`
2. **The Authentication Stack is deployed**: `make deploy-auth`
3. **Local `/etc/hosts` & Keychain is configured**: `make tls-setup`
4. **The Minikube Tunnel is active**: Keep a terminal window open running `make minikube-tunnel` (this binds ports 80/443 on your Mac to the Minikube Ingress).

---

## 1. Top-Level Health Checks

Run the automated health check make target to ensure all pods and services are successfully provisioned:

```bash
make check-auth-status
```

All components (`auth-postgres`, `redis`, `hydra`, `kratos`, `oathkeeper`, `spicedb`, `web3-api`) should be `Running`.

---

## 2. Component Endpoint Testing (cURL)

Because all traffic goes through NGINX Ingress using a locally trusted Cert-Manager Root CA, we can `curl` the fully qualified domains.

### Kratos Health (Identity Engine)

Check the administrative and public alive endpoints. _(Note: We use `-k` to avoid curl complaining if your local machine hasn't loaded the Keychain CA into curl's internal store yet)._

```bash
# Admin API
curl -s -k -v https://kratos-admin.web3-local-dev.com/admin/health/alive

# Public API
curl -s -k -v https://kratos.web3-local-dev.com/health/alive
```

**Expected Output for both:**

```json
{ "status": "ok" }
```

### Hydra Health (OAuth2 Provider)

Check the administrative and public alive endpoints to ensure the OAuth2 engine is running.

```bash
# Admin API
curl -s -k -v https://hydra-admin.web3-local-dev.com/health/alive

# Public API
curl -s -k -v https://hydra.web3-local-dev.com/health/alive
```

**Expected Output for both:**

```json
{ "status": "ok" }
```

### Oathkeeper Health (API Gateway)

Check the Oathkeeper routing API to ensure the configuration maps are loaded and NGINX routes cleanly.

```bash
curl -s -k -v https://auth-api.web3-local-dev.com/health/alive
```

**Expected Output:**

```json
{ "status": "ok" }
```

### Backend Web3 API Health

Check the custom Go API backend.

```bash
curl -s -k -v https://api.web3-local-dev.com/health
```

**Expected Output:**

```text
OK
```

---

## 3. Functionality Verification

To verify the components are functioning collectively as an identity ecosystem, test their actual APIs:

### Verify Kratos Identities List

```bash
# Query the Kratos Admin API to list all registered identities
curl -s -k https://kratos-admin.web3-local-dev.com/admin/identities | jq .
```

_(If no users have logged in via the Wallet yet, you will receive an empty array `[]`)_

### Verify SpiceDB Schema

To check if SpiceDB successfully applied its schema, use the automated Makefile targets which use the `zed` CLI:

```bash
# Terminal 1: Forward the gRPC port
make port-forward-spicedb

# Terminal 2: Use zed CLI locally to read the schema
make spicedb-verify
```

### Verify Oathkeeper Routing

```bash
# Fetch the list of configured Oathkeeper access rules loaded from the ConfigMap
curl -s -k https://auth-api.web3-local-dev.com/rules | jq .
```

_(You should see an array of rules defining how traffic on `auth.web3-local-dev.com` routes to your backend APIs)._

---

## 4. APISIX Gateway Testing

With the APISIX auth gateway deployed (`make deploy-apisix-auth-gateway`), all services are accessible through the unified `gateway.web3-local-dev.com` endpoint.

> [!NOTE]
> On macOS Docker driver, `make minikube-tunnel` runs with `sudo` to bind privileged ports (80/443).
> To clean up stale tunnel processes: `make minikube-tunnel-stop`.

### Public Endpoints (rate-limited, CORS)

```bash
# Kratos Public — Identity health
curl -sk https://gateway.web3-local-dev.com/identity/health/alive
# Expected: {"status":"ok"}

# Hydra Public — OAuth2 health
curl -sk https://gateway.web3-local-dev.com/oauth2/health/alive
# Expected: {"status":"ok"}

# Oathkeeper Proxy — handles access decisions only, no health endpoint
# Health check is on the admin route: /admin/auth/health/alive (see Admin section)

# Backend API — Go API health
curl -sk https://gateway.web3-local-dev.com/api/health
# Expected: OK

# Hydra Userinfo (root-level, requires Bearer token)
curl -sk -H "Authorization: Bearer <access_token>" \
  https://gateway.web3-local-dev.com/userinfo
# Expected: {"sub":"...","email":"...","name":"..."}

# Hydra OIDC Discovery
curl -sk https://gateway.web3-local-dev.com/.well-known/openid-configuration | jq .
# Expected: JSON with issuer, authorization_endpoint, token_endpoint, userinfo_endpoint
```

### Admin Endpoints (key-auth protected)

All admin endpoints require `X-Admin-Key` header.

```bash
# --- Without key (should 401) ---

curl -sk https://gateway.web3-local-dev.com/admin/identity/admin/health/alive
# Expected: {"message":"Missing API key in request"}  HTTP 401

curl -sk https://gateway.web3-local-dev.com/admin/oauth2/health/alive
# Expected: {"message":"Missing API key in request"}  HTTP 401

curl -sk https://gateway.web3-local-dev.com/admin/auth/health/alive
# Expected: {"message":"Missing API key in request"}  HTTP 401

curl -sk https://gateway.web3-local-dev.com/admin/authz/v1/schema
# Expected: {"message":"Missing API key in request"}  HTTP 401
```

```bash
# --- With valid key (should 200) ---

curl -sk -H "X-Admin-Key: web3-admin-secret-key" \
  https://gateway.web3-local-dev.com/admin/identity/admin/health/alive
# Expected: {"status":"ok"}

curl -sk -H "X-Admin-Key: web3-admin-secret-key" \
  https://gateway.web3-local-dev.com/admin/oauth2/health/alive
# Expected: {"status":"ok"}

curl -sk -H "X-Admin-Key: web3-admin-secret-key" \
  https://gateway.web3-local-dev.com/admin/auth/health/alive
# Expected: {"status":"ok"}

curl -sk -X POST \
  -H "X-Admin-Key: web3-admin-secret-key" \
  -H "Authorization: Bearer web3-lab-spicedb-key-not-for-production" \
  -H "Content-Type: application/json" \
  -d '{}' \
  https://gateway.web3-local-dev.com/admin/authz/v1/schema/read
# Expected: SpiceDB schema JSON object
```

```bash
# --- With wrong key (should 401) ---

curl -sk -H "X-Admin-Key: wrong-key" \
  https://gateway.web3-local-dev.com/admin/identity/admin/health/alive
# Expected: {"message":"Invalid API key in request"}  HTTP 401
```

### Root Path

```bash
curl -sk https://gateway.web3-local-dev.com/
# Expected: {"error_msg":"404 Route Not Found"}  HTTP 404
```

### Rate Limiting Verification

```bash
# Rapid-fire 20 requests — should see 429 responses after the burst limit
for i in {1..20}; do
  curl -sk -o /dev/null -w "%{http_code}\n" \
    https://gateway.web3-local-dev.com/identity/health/alive
  sleep 0.05
done
# Expected: first ~15 return 200, remaining return 429
```

> [!NOTE]
> The direct per-service endpoints (`kratos.web3-local-dev.com`, `hydra.web3-local-dev.com`, etc.) remain available for backward compatibility and debugging. See [local-setup.md](local-setup.md) for the full hostname mapping.
