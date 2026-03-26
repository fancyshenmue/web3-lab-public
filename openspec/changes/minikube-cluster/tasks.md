## 1. OpenSpec

- [x] 1.1 Create `openspec/specs/minikube-cluster/spec.md` with cluster requirements
- [x] 1.2 Create `openspec/changes/minikube-cluster/tasks.md` (this file)

## 2. Cluster Setup

- [x] 2.1 Minikube 3-node cluster (profile: `web3-lab`, driver: Docker)
- [x] 2.2 Configurable resources via Makefile variables (`--memory`, `--cpus`)

## 3. Namespace Isolation

- [x] 3.1 Dedicated `web3` namespace for all workloads

## 4. Persistent Storage

- [x] 4.1 Create `pv.yaml` — hostPath PVs with nodeAffinity
- [x] 4.2 Geth (EL): 3 PVs × 5Gi (`/data/geth/data{0,1,2}`)
- [x] 4.3 Beacon (CL): 3 PVs × 5Gi (`/data/beacon/data{0,1,2}`)
- [x] 4.4 Validator: 3 PVs × 1Gi (`/data/validator/data{0,1,2}`)

## 5. Makefile

- [x] 5.1 `minikube-start` / `minikube-stop` / `minikube-delete` / `minikube-status`
- [x] 5.2 `create-namespace`
- [x] 5.3 `apply-pv`
- [x] 5.4 `init-infra` (aggregate: `apply-pv` + `create-namespace`)
- [x] 5.5 `check-host-paths` — verify PV host directories on all nodes

## Status: ✅ ARCHIVED
