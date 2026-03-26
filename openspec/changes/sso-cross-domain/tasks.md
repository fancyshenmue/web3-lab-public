## 1. Specification
- [x] 1.1 Create `openspec/specs/sso-cross-domain/spec.md` with SSO architecture
- [x] 1.2 Create `openspec/changes/sso-cross-domain/tasks.md` (this file)

## 2. Platform Infrastructure Updates
- [x] 2.1 Update Let's Encrypt / Cert-Manager `certificate.yaml` to include `app.web3-local-dev-2.com`
- [x] 2.2 Update Frontend NGINX Ingress rules to route `app.web3-local-dev-2.com` to the new frontend-2 service (or APISIX routes if handled there)

## 3. Frontend App-2 Setup
- [x] 3.1 Duplicate/create `frontend/app-2` based on `frontend/app`
- [x] 3.2 Modify `frontend/app-2` branding (colors, titles) to clearly distinguish it as "App 2"
- [x] 3.3 Create Dockerfile for `frontend/app-2`
- [x] 3.4 Create Kustomize manifests for `frontend-2` (Deployment, Service) in `deployments/kustomize/frontend-2`

## 4. Makefile and Local System
- [x] 4.1 Update `Makefile` with targets for `build-frontend-2`, `load-frontend-2`, `deploy-frontend-2`
- [x] 4.2 Document `/etc/hosts` addition for `app.web3-local-dev-2.com`

## 5. Seed Data & Database
- [x] 5.1 Use the API or Update SQL migration seeds to register the `app-2` client in `app_clients` table
- [x] 5.2 Verify Hydra auto-provisions the OAuth2 client config with correct redirect URIs `https://app.web3-local-dev-2.com`

## 6. End-to-End SSO Testing
- [x] 6.1 Build, Load, and Deploy App-2
- [x] 6.2 Verify local domain resolution and TLS
- [x] 6.3 Perform Login on App-1 -> check session state
- [x] 6.4 Open App-2 -> verify instant SSO (no second login prompt)
- [x] 6.5 Test Cross-Domain Logout (Single Logout) behavior
