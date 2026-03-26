## 1. OpenSpec & Planning

- [x] 1.1 Create `openspec/specs/geth-pos-cluster/spec.md` with architecture, requirements, and layer responsibilities
- [x] 1.2 Create `openspec/config.yaml` with project context and conventions
- [x] 1.3 Create `openspec/changes/geth-pos-cluster/tasks.md` (this file)

## 2. Storage

- [x] 2.1 Create PVs for Geth (EL) — 3 × 5Gi across 3 minikube nodes
- [x] 2.2 Create PVs for Beacon (CL) — 3 × 5Gi across 3 minikube nodes
- [x] 2.3 Create PVs for Validator — 3 × 1Gi across 3 minikube nodes

## 3. Geth (Execution Layer) Manifests

- [x] 3.1 Create `genesis-configmap.yaml` — Genesis JSON with chainId 72390, PoS from genesis (TTD=0)
- [x] 3.2 Create `jwt-secret.yaml` — Shared JWT secret for Engine API auth (Geth ↔ Beacon)
- [x] 3.3 Create `geth.yaml` — StatefulSet (3 replicas) with init container for `geth init`, Engine API on 8551
- [x] 3.4 Create `services.yaml` — Headless service + ClusterIP RPC/WS service
- [x] 3.5 Fix geth P2P — `--nat=extip:$POD_IP` via K8s downward API, rewrite enode with peer DNS

## 4. Beacon (Consensus Layer) Manifests

- [x] 4.1 Create `beacon.yaml` — StatefulSet (3 replicas) connecting to Geth via Engine API
- [x] 4.2 Create `beacon-services.yaml` — Headless service + ClusterIP gRPC/HTTP service
- [x] 4.3 Distroless fix — busybox init container generates YAML config, Prysm reads via `--config-file`
- [x] 4.4 Genesis generation — `prysmctl` init container with conditional check-db (restart-safe)
- [x] 4.5 Beacon P2P peering — optional ConfigMap + `setup-beacon-peers` Makefile target

## 5. Validator Manifests

- [x] 5.1 Create `validator.yaml` — StatefulSet (3 replicas) connecting to Beacon via gRPC
- [x] 5.2 Key sharding — 64 interop keys split across 3 instances
- [x] 5.3 Monitoring port — `--monitoring-host/port=8081` for health probes

## 6. Health Probes

- [x] 6.1 Geth — TCP liveness/readiness on :8545
- [x] 6.2 Beacon — TCP liveness on :4000, HTTP readiness `/eth/v1/node/health` on :3500
- [x] 6.3 Validator — HTTP liveness/readiness `/healthz` on :8081

## 7. Makefile

- [x] 7.1 Deploy/delete targets for all layers
- [x] 7.2 Cleanup targets (PVC, PV, data)
- [x] 7.3 Port-forward targets
- [x] 7.4 `setup-beacon-peers` — query identities → ConfigMap → rollout restart
- [x] 7.5 `check-pos-status` — deep health check (geth peers, beacon sync, validators)

## 8. Documentation

- [x] 8.1 `documents/geth/architecture.md` — full architecture with Mermaid diagrams
- [x] 8.2 `documents/geth/testing.md` — curl + ethers.js test examples
- [x] 8.3 `documents/geth/chain-economics.md` — pre-funded accounts, supply, minting
- [x] 8.4 Update OpenSpec to reflect final implementation

## 9. Verification

- [x] 9.1 All 9 pods Running (3 geth, 3 beacon, 3 validator)
- [x] 9.2 Geth: 2 peers each (full mesh)
- [x] 9.3 Beacon: connected=2 each (P2P peering working)
- [x] 9.4 Validators: ready, 0 restarts

## Status: ✅ ARCHIVED
