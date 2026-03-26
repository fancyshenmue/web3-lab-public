## 1. OpenSpec

- [x] 1.1 Create `openspec/specs/native-token-pricing/spec.md` with architecture and endpoints
- [x] 1.2 Create `openspec/changes/native-token-pricing/tasks.md` (this file)

## 2. Go Source

- [x] 2.1 Write `backend/cmd/mock-price/main.go` (~150 lines, stdlib-only HTTP server)
- [x] 2.2 Initialize `backend/go.mod` (Go 1.26 via pixi)
- [x] 2.3 Endpoints: `/coins/{id}`, `/coins/{id}/market_chart`, `/simple/price`, `/health`

## 3. Container Build

- [x] 3.1 Create `deployments/build/mock-price/Dockerfile` (multi-stage Go 1.26 → Alpine 3.20)

## 4. Kubernetes Deployment (Kustomize)

- [x] 4.1 Create kustomize base (`deployments/kustomize/mock-price/base/`)
- [x] 4.2 Create minikube overlay (`deployments/kustomize/mock-price/overlays/minikube/`)

## 5. Makefile

- [x] 5.1 `build-mock-price` — Docker build with VERSION tag
- [x] 5.2 `load-mock-price` — Minikube image load
- [x] 5.3 `deploy-mock-price` — kustomize edit image + kubectl apply
- [x] 5.4 `delete-mock-price` — kubectl delete
- [x] 5.5 `kustomize-update-mock-price` — update image tag

## 6. Blockscout Integration

- [x] 6.1 Update `blockscout.yaml` with `MARKET_COINGECKO_BASE_URL` + `MARKET_COINGECKO_COIN_ID`

## 7. Verification

- [x] 7.1 Build, load, deploy (`make build-mock-price load-mock-price deploy-mock-price`)
- [x] 7.2 Verify `coin_price` shows in Blockscout stats API ✅ `"coin_price":"2.48"`

## Status: ✅ ARCHIVED
