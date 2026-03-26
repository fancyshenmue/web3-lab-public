# Local TLS & Environment Setup

This stack relies heavily on secure browser features like HTTP-Only Secure Cookies, SameSite=None, and proper CORS origins. To maintain exact parity with the production environment, the local Minikube cluster utilizes **NGINX Ingress** and **Cert-Manager** to serve everything over HTTPS (`*.web3-local-dev.com`).

## The `tls-setup` Process

To avoid browser "Not Secure" warnings, `cert-manager` dynamically generates a Root Certificate Authority (CA) inside the Kubernetes cluster. The host development machine running Minikube must explicitly trust this Root CA.

We have provided a fully automated Makefile target to map your local domains and trust the certificate.

### Execution

Simply run the following command in your terminal:

```bash
make tls-setup
```

### What this script does behind the scenes:

1. **Extracts the Root CA**:
   It reaches into the Minikube cluster and downloads the `cert-manager` self-signed root certificate.

   ```bash
   kubectl get secret root-ca-secret -n cert-manager -o jsonpath='{.data.ca.crt}' | base64 -d > /tmp/root-ca.crt
   ```

2. **Trusts the Certificate (macOS System Keychain)**:
   It utilizes the native macOS `security` tool to inject the CA into the System Keychain. This tells Chrome, Safari, and the OS to treat any certificate issued by this CA as perfectly secure.
   _(This step prompts for your macOS administrator password)._

   ```bash
   sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain /tmp/root-ca.crt
   ```

3. **Configures DNS (`/etc/hosts`)**:
   It maps the cluster's Ingress domains to `127.0.0.1` (localhost) by appending them to `/etc/hosts`.

### Important: Minikube Tunnel Required
Because the domains are mapped to `127.0.0.1`, you **MUST** run the Minikube tunnel to forward traffic to the internal Ingress controller. Keep a separate terminal window open running:

```bash
make minikube-tunnel
```

### The `/etc/hosts` Configuration

If you prefer to map the `/etc/hosts` manually, the required format for `127.0.0.1` is:

```text
# Web3-Lab Local Auth
127.0.0.1 gateway.web3-local-dev.com
127.0.0.1 hydra.web3-local-dev.com
127.0.0.1 hydra-admin.web3-local-dev.com
127.0.0.1 kratos.web3-local-dev.com
127.0.0.1 kratos-admin.web3-local-dev.com
127.0.0.1 auth.web3-local-dev.com
127.0.0.1 auth-api.web3-local-dev.com
127.0.0.1 spicedb.web3-local-dev.com
127.0.0.1 api.web3-local-dev.com
127.0.0.1 app.web3-local-dev.com
```

### Hostname Mapping Index

> [!TIP]
> With APISIX deployed, all services are also accessible through the unified gateway at `gateway.web3-local-dev.com`. The per-service hostnames below remain available for backward compatibility and direct debugging.

| Component            | Direct Ingress Domain     | APISIX Gateway Path                        |
| :------------------- | :------------------------ | :----------------------------------------- |
| **APISIX Gateway**   | `gateway.web3-local-dev.com`      | — (unified entrypoint)                     |
| **Frontend App**     | `app.web3-local-dev.com`          | — (served via own Ingress)                 |
| **Kratos Public**    | `kratos.web3-local-dev.com`       | `gateway.web3-local-dev.com/identity/*`            |
| **Kratos Admin**     | `kratos-admin.web3-local-dev.com` | `gateway.web3-local-dev.com/admin/identity/*`      |
| **Hydra Public**     | `hydra.web3-local-dev.com`        | `gateway.web3-local-dev.com/oauth2/*`              |
| **Hydra Userinfo**   | —                                 | `gateway.web3-local-dev.com/userinfo`              |
| **Hydra Discovery**  | —                                 | `gateway.web3-local-dev.com/.well-known/*`         |
| **Hydra Admin**      | `hydra-admin.web3-local-dev.com`  | `gateway.web3-local-dev.com/admin/oauth2/*`        |
| **Oathkeeper Proxy** | `auth.web3-local-dev.com`         | `gateway.web3-local-dev.com/auth/*`                |
| **Oathkeeper API**   | `auth-api.web3-local-dev.com`     | `gateway.web3-local-dev.com/admin/auth/*`          |
| **SpiceDB**          | `spicedb.web3-local-dev.com`      | `gateway.web3-local-dev.com/admin/authz/*` (HTTP)  |
| **Backend API**      | `api.web3-local-dev.com`          | `gateway.web3-local-dev.com/api/*`                 |

