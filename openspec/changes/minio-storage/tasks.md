## 1. OpenSpec

- [x] 1.1 Create `openspec/specs/minio-storage/spec.md` with architecture and bucket structure
- [x] 1.2 Create `openspec/changes/minio-storage/tasks.md` (this file)

## 2. Storage

- [x] 2.1 Add MinIO PVs (6 × 10Gi) to `pv.yaml` with nodeAffinity

## 3. Kubernetes Manifests

- [x] 3.1 Create `minio/minio-secret.yaml` — admin credentials Secret
- [x] 3.2 Create `minio/minio-service.yaml` — Headless + ClusterIP (API 9000, Console 9001)
- [x] 3.3 Create `minio/minio-statefulset.yaml` — StatefulSet (6 replicas) distributed mode

## 4. Makefile

- [x] 4.1 `deploy-minio` / `delete-minio` targets
- [x] 4.2 `cleanup-minio-pvc` / `cleanup-minio-pv` / `cleanup-minio-data`
- [x] 4.3 `init-minio` — create bucket + set anonymous download policy
- [x] 4.4 `port-forward-minio` (API 9000 + Console 9001)
- [x] 4.5 Update `deploy-all` / `delete-all` to include MinIO
- [x] 4.6 Update `verify-nodes` to include MinIO pods

## 5. Bucket Initialization

- [x] 5.1 `web3lab-assets` bucket auto-creation via `init-minio`
- [x] 5.2 Anonymous download policy for public asset access

## 6. Verification

- [x] 6.1 All 6 MinIO pods Running across 3 nodes
- [x] 6.2 Console accessible via port-forward (localhost:9001)
- [x] 6.3 Bucket creation and object upload/download verified

## Status: ✅ ARCHIVED
