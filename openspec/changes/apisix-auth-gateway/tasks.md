# APISIX Auth Gateway Tasks

## 0. APISIX Installation & Infrastructure

- [x] 0.1 Create `deployments/kubernetes/minikube/storage/apisix-storage.yaml` (3 PVs for etcd)
- [x] 0.2 Create `deployments/helm/apisix-values.yaml` (Helm values with volumePermissions fix)
- [x] 0.3 Install APISIX via Helm (`helm install apisix apisix/apisix -f apisix-values.yaml`)
- [x] 0.4 Create `deployments/kustomize/apisix-auth-gateway/base/gateway-proxy.yaml` (GatewayProxy CRD)
- [x] 0.5 Patch IngressClass to link GatewayProxy
- [x] 0.6 ~~Create ExternalName bridge~~ → Moved ingress/cert to `apisix` namespace (ExternalName DNS failed)
- [x] 0.7 Add `apisix-install` / `apisix-uninstall` Makefile targets
- [x] 0.8 ~~Fix consumer sync: Admin API separated endpoints~~ → Fixed by adding `ingressClassName: apisix` to ApisixConsumer CRD
- [x] 0.9 ~~Create consumer via Admin API post-deploy~~ → No longer needed, CRD syncs via ingress controller
- [x] 0.10 Add `minikube-tunnel-stop` Makefile target (kills zombie SSH tunnels + lock file)

## 1. OpenSpec & Planning

- [x] 1.1 Create `openspec/specs/apisix-auth-gateway/spec.md`
- [x] 1.2 Create `openspec/changes/apisix-auth-gateway/tasks.md` (this file)

## 2. Documentation

- [x] 2.1 Create `documents/apisix/architecture.md` (TB diagram + sequence diagrams)

## 3. APISIX Route Manifests (Kustomize)

- [x] 3.1 Create `deployments/kustomize/apisix-auth-gateway/base/kustomization.yaml`
- [x] 3.2 Create `deployments/kustomize/apisix-auth-gateway/base/apisix-route-public.yaml`
- [x] 3.3 Create `deployments/kustomize/apisix-auth-gateway/base/apisix-route-admin.yaml`
- [x] 3.4 Create `deployments/kustomize/apisix-auth-gateway/base/apisix-consumer.yaml`
- [x] 3.5 Create `deployments/kustomize/apisix-auth-gateway/overlays/minikube/kustomization.yaml`
- [x] 3.6 Create `deployments/kustomize/apisix-auth-gateway/overlays/minikube/ingress.yaml`
- [x] 3.7 Create `deployments/kustomize/apisix-auth-gateway/overlays/minikube/certificate.yaml`

## 4. Service URL Reconfiguration

- [ ] 4.1 Update Hydra config: `urls.self.issuer` → `gateway.web3-local-dev.com/oauth2`
- [ ] 4.2 Update Kratos config: `serve.public.base_url` → `gateway.web3-local-dev.com/identity`
- [ ] 4.3 Update OAuth2 redirect/consent URLs

## 5. Ingress Migration

- [ ] 5.1 Remove per-service Ingress resources (kratos, hydra, oathkeeper, spicedb, api)
- [x] 5.2 Update `/etc/hosts` to add `gateway.web3-local-dev.com`
- [x] 5.3 Update `make tls-setup` target

## 6. Makefile Targets

- [x] 6.1 Add `deploy-apisix-auth-gateway` target
- [x] 6.2 Add `delete-apisix-auth-gateway` target

## 7. Verification

- [x] 7.1 Test public endpoints → 200 OK with auth services running
- [x] 7.2 Test admin endpoints → 401 without key, 200 with key
- [x] 7.3 Verify consumer persists through 90s ADC sync cycle
- [ ] 7.4 Test rate limiting (429 on burst)
- [ ] 7.5 Verify Prometheus metrics scraping
