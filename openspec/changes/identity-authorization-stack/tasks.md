## 1. OpenSpec & Planning

- [x] 1.1 Create `openspec/specs/identity-authorization-stack/spec.md`
- [x] 1.2 Create `openspec/changes/identity-authorization-stack/tasks.md` (this file)

## 2. PostgreSQL (Shared Database) Kustomize Manifests

- [x] 2.1 Create `deployments/kustomize/postgres/base/kustomization.yaml`
- [x] 2.2 Create `deployments/kustomize/postgres/base/statefulset.yaml`
- [x] 2.3 Create `deployments/kustomize/postgres/base/service.yaml`
- [x] 2.4 Create `deployments/kustomize/postgres/base/configmap-init.yaml`
- [x] 2.5 Create `deployments/kustomize/postgres/overlays/minikube/kustomization.yaml`
- [x] 2.6 Create `deployments/kustomize/postgres/overlays/minikube/patch-statefulset.yaml`

## 3. Redis (Nonce Storage) Kustomize Manifests

- [x] 3.1 Create `deployments/kustomize/redis/base/kustomization.yaml`
- [x] 3.2 Create `deployments/kustomize/redis/base/deployment.yaml`
- [x] 3.3 Create `deployments/kustomize/redis/base/service.yaml`
- [x] 3.4 Create `deployments/kustomize/redis/overlays/minikube/kustomization.yaml`
- [x] 3.5 Create `deployments/kustomize/redis/overlays/minikube/patch-deployment.yaml`

## 4. Hydra (OAuth2) Kustomize Manifests

- [x] 4.1 Create `deployments/kustomize/hydra/base/kustomization.yaml`
- [x] 4.2 Create `deployments/kustomize/hydra/base/deployment.yaml`
- [x] 4.3 Create `deployments/kustomize/hydra/base/service.yaml`
- [x] 4.4 Create `deployments/kustomize/hydra/base/configmap-data/configmap-config.yaml`
- [x] 4.5 Create `deployments/kustomize/hydra/overlays/minikube/kustomization.yaml`
- [x] 4.6 Create `deployments/kustomize/hydra/overlays/minikube/configmap-env.yaml`
- [x] 4.7 Create `deployments/kustomize/hydra/overlays/minikube/patch-deployment.yaml`

## 5. Kratos (Identity) Kustomize Manifests

- [x] 5.1 Create `deployments/kustomize/kratos/base/kustomization.yaml`
- [x] 5.2 Create `deployments/kustomize/kratos/base/deployment.yaml`
- [x] 5.3 Create `deployments/kustomize/kratos/base/service.yaml`
- [x] 5.4 Create `deployments/kustomize/kratos/overlays/minikube/kustomization.yaml`
- [x] 5.5 Create `deployments/kustomize/kratos/overlays/minikube/configmap-env.yaml`
- [x] 5.6 Create `deployments/kustomize/kratos/overlays/minikube/patch-deployment.yaml`

## 6. Oathkeeper (API Gateway) Kustomize Manifests

- [x] 6.1 Create `deployments/kustomize/oathkeeper/base/kustomization.yaml`
- [x] 6.2 Create `deployments/kustomize/oathkeeper/base/deployment.yaml`
- [x] 6.3 Create `deployments/kustomize/oathkeeper/base/service.yaml`
- [x] 6.4 Create `deployments/kustomize/oathkeeper/overlays/minikube/kustomization.yaml`
- [x] 6.5 Create `deployments/kustomize/oathkeeper/overlays/minikube/configmap-env.yaml`
- [x] 6.6 Create `deployments/kustomize/oathkeeper/overlays/minikube/patch-deployment.yaml`

## 7. SpiceDB (Authorization) Kustomize Manifests — PostgreSQL

- [x] 7.1 Create `deployments/kustomize/spicedb/base/kustomization.yaml`
- [x] 7.2 Create `deployments/kustomize/spicedb/base/deployment.yaml`
- [x] 7.3 Create `deployments/kustomize/spicedb/base/service.yaml`
- [x] 7.4 Create `deployments/kustomize/spicedb/overlays/minikube/kustomization.yaml`
- [x] 7.5 Create `deployments/kustomize/spicedb/overlays/minikube/configmap-env.yaml`
- [x] 7.6 Create `deployments/kustomize/spicedb/overlays/minikube/patch-deployment.yaml`
- [x] 7.7 Create `deployments/kustomize/spicedb/overlays/minikube/job-migrate.yaml`

## 8. Backend API Kustomize Manifests

- [x] 8.1 Create `deployments/kustomize/api/base/kustomization.yaml`
- [x] 8.2 Create `deployments/kustomize/api/base/deployment.yaml`
- [x] 8.3 Create `deployments/kustomize/api/base/service.yaml`
- [x] 8.4 Create `deployments/kustomize/api/overlays/minikube/kustomization.yaml`
- [x] 8.5 Create `deployments/kustomize/api/overlays/minikube/configmap-env.yaml`
- [x] 8.6 Create `deployments/kustomize/api/overlays/minikube/configmap-config.yaml`
- [x] 8.7 Create `deployments/kustomize/api/overlays/minikube/patch-deployment.yaml`

## 9. Makefile Targets

- [x] 9.1 Add deploy/delete targets for all seven services
- [x] 9.2 Add aggregate deploy-auth / delete-auth targets
- [x] 9.3 Add build-api / load-api targets
- [x] 9.4 Add check-auth-status target
- [x] 9.5 Update deploy-all / delete-all

## 10. Verification

- [x] 10.1 Run `kubectl kustomize` dry-run on all seven services ✅
- [x] 10.2 Verify SpiceDB uses PostgreSQL engine ✅
- [x] 10.3 Verify namespace `web3` in all overlays ✅

## 11. TLS & Ingress Configuration

- [x] 11.1 Enable Minikube `ingress` addon
- [x] 11.2 Install `cert-manager` to Minikube
- [x] 11.3 Create `local-ca.yaml` (SelfSigned Issuer, Root CA Certificate, Local CA Issuer)
- [x] 11.4 Create `deployments/kustomize/hydra/overlays/minikube/ingress.yaml` and update Kustomization
- [x] 11.5 Create `deployments/kustomize/kratos/overlays/minikube/ingress.yaml` and update Kustomization
- [x] 11.6 Create `deployments/kustomize/oathkeeper/overlays/minikube/ingress.yaml` and update Kustomization
- [x] 11.7 Create `deployments/kustomize/spicedb/overlays/minikube/ingress.yaml` and update Kustomization
- [x] 11.8 Create `deployments/kustomize/api/overlays/minikube/ingress.yaml` and update Kustomization

## 12. Documentation

- [x] 12.1 Write architecture and sequence flow diagrams (`architecture.md`, `authentication-flow.md`)
- [x] 12.2 Write local TLS Minikube networking guide (`local-setup.md`)
- [x] 12.3 Write testing and health check commands (`testing.md`)

## 13. Kratos OAuth2 Integration

- [x] 13.1 Configure `oauth2_provider` in Kratos config to point to Hydra admin
- [x] 13.2 Add `session` hook before `web_hook` in registration `after.password.hooks`
- [x] 13.3 Add `session` hook before `web_hook` in registration `after.oidc.hooks`
- [x] 13.4 Configure Kratos cookies: `domain: .web3-local-dev.com`, `same_site: None`, `path: /`
- [x] 13.5 Create provider-specific Jsonnet webhook templates (`registration-webhook-email.jsonnet`, `registration-webhook-oidc.jsonnet`)

## 14. Backend HandleLogin Session Bridging

- [x] 14.1 Add `KRATOS_PUBLIC_URL` to API config for session verification
- [x] 14.2 Implement `CheckSessionWithCookies` in `kratos_admin_service.go` (forwards cookies to `/sessions/whoami`)
- [x] 14.3 Update `HandleLogin` to check Kratos session via `whoami` when Hydra `skip=false`
- [x] 14.4 Auto-accept Hydra login if valid Kratos session exists

## 15. Frontend Auth Flow Updates

- [x] 15.1 Create Kratos login flow with `login_challenge` on LoginPage load (stores OAuth2 context in cookies)
- [x] 15.2 Handle Kratos 422 `redirect_browser_to` response before `submitRes.ok` check (sign-in)
- [x] 15.3 Handle Kratos 422 `redirect_browser_to` response for sign-up
- [x] 15.4 Auto-trigger OAuth2 flow in HomePage on mount

## 16. Documentation Updates (Auth Flow Fixes)

- [x] 16.1 Update `authentication-flow.md` with Email Registration + OAuth2 Auto-Login sequence
- [x] 16.2 Add Email Sign-In + OAuth2 flow diagram to `authentication-flow.md`
- [x] 16.3 Update configuration gotchas with session hook and 422 response notes
- [x] 16.4 Update `spec.md` Kratos requirement with `oauth2_provider`, session hook, cookie config
- [x] 16.5 Update `spec.md` Backend API requirement with `HandleLogin` session check
- [x] 16.6 Update `tasks.md` with new task sections (this file)

## 17. Sign Out Fix (Kratos Session Revocation)

- [x] 17.1 Add `RevokeIdentitySessions` to `kratos_admin_service.go` (`DELETE /admin/identities/{id}/sessions`)
- [x] 17.2 Update `HandleLogout` to get `subject` from Hydra logout request and revoke Kratos sessions
- [x] 17.3 Update `HomePage.tsx` to check `?logout=true` and show "Signed out" + manual Sign In button
- [x] 17.4 Update Hydra `URLS_POST_LOGOUT_REDIRECT` to include `?logout=true`

## 18. Documentation Updates (Sign Out Fix)

- [x] 18.1 Update `authentication-flow.md` OAuth2 Logout Flow diagram with Kratos session revocation
- [x] 18.2 Add logout-then-auto-login loop gotcha note to `authentication-flow.md`
- [x] 18.3 Update `spec.md` Backend API with `HandleLogout` session revocation requirement
- [x] 18.4 Update `spec.md` Frontend with `?logout=true` HomePage behavior
- [x] 18.5 Update `tasks.md` with Sign Out fix tasks (this file)
