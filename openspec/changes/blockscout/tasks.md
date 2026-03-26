## 1. OpenSpec

- [x] 1.1 Create `openspec/specs/blockscout/spec.md` with 3-service architecture
- [x] 1.2 Create `openspec/changes/blockscout/tasks.md` (this file)

## 2. Storage

- [x] 2.1 Add Blockscout Postgres PV (5Gi) to `pv.yaml`

## 3. Kubernetes Manifests

- [x] 3.1 Create `blockscout/postgres.yaml` — StatefulSet (1 replica) + headless/ClusterIP services
- [x] 3.2 Create `blockscout/blockscout.yaml` — Backend Deployment (Elixir) + ClusterIP service (port 4000)
- [x] 3.3 Create `blockscout/frontend.yaml` — Frontend Deployment (Next.js) + ClusterIP service (port 3000)
- [x] 3.4 Create `blockscout/stats.yaml` — Stats Deployment (Rust) + ClusterIP service (port 8050)
- [x] 3.5 Create `blockscout/proxy.yaml` — Nginx reverse proxy (frontend + backend unified access)

## 4. Native Token Pricing (labETH)

- [x] 4.1 Configure `MARKET_COINGECKO_BASE_URL` → `http://mock-price-api:4050`
- [x] 4.2 Configure `MARKET_COINGECKO_COIN_ID` → `labeth`
- [x] 4.3 Verify `coin_price` appears in Blockscout stats API

## 5. NFT Metadata & Seed Data

- [x] 5.1 Token instance fetcher indexes ERC-721/1155 `tokenURI()` / `uri()`
- [x] 5.2 `seed/update-blockscout-icons.sh` — set ERC-20 `icon_url`, fix blacklisted ERC-721 metadata
- [x] 5.3 ERC-1155 metadata auto-fetched (correct `Content-Type` via MinIO)

## 6. Makefile

- [x] 6.1 `deploy-blockscout` / `delete-blockscout` targets (deploys all components incl. proxy)
- [x] 6.2 `port-forward-blockscout` (proxy 3001:80)
- [x] 6.3 `port-forward-blockscout-backend` (4001:4000)
- [x] 6.4 `port-forward-blockscout-postgres` (5433:5432)
- [x] 6.5 `port-forward-blockscout-stats` (8150:8050)
- [x] 6.6 `cleanup-blockscout-pvc` / `cleanup-blockscout-pv` / `cleanup-blockscout-data`
- [x] 6.7 `seed-update-icons` target
- [x] 6.8 Update `deploy-all` / `delete-all` aggregates
- [x] 6.9 Update `verify-nodes` to include all Blockscout components

## 7. Verification

- [x] 7.1 YAML syntax validation
- [x] 7.2 Deploy to Minikube, verify Blockscout indexes blocks from Geth
- [x] 7.3 Native token price displayed on dashboard
- [x] 7.4 Token icons and NFT metadata visible in explorer

## Status: ✅ ARCHIVED
