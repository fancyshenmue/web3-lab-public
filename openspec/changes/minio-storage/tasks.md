# MinIO Dynamic Upload & Dedicated Ingress Tasks

## 1. OpenSpec Updates

- [x] 1.1 Update `openspec/specs/minio-storage/spec.md` — dynamic Presigned URL architecture, dedicated Ingress, dual URL strategy
- [x] 1.2 Update `openspec/specs/frontend-erc721-erc1155-ux/spec.md` — image upload sections 5 & 6
- [x] 1.3 Update `openspec/specs/apisix-auth-gateway/spec.md` — note MinIO uses own Ingress
- [x] 1.4 Create change tasks (this file)

## 2. Smart Contracts

- [x] 2.1 Add `setBaseURI(string memory newBaseURI) onlyOwner` to `Web3LabERC721.sol`
- [x] 2.2 Add `setURI(string memory newuri) onlyOwner` to `Web3LabERC1155.sol`
- [x] 2.3 Recompile contracts via Hardhat
- [x] 2.4 Redeploy contracts (`make deploy-contracts`)

## 3. Infrastructure (K8s)

- [x] 3.1 Create MinIO Ingress YAML (`minio.web3-local-dev.com`, TLS, proxy-body-size 50m)
- [x] 3.2 Add CORS env vars to MinIO StatefulSet (`MINIO_API_CORS_ALLOW_ORIGIN`)
- [x] 3.3 Update `/etc/hosts` — add `minio.web3-local-dev.com`
- [x] 3.4 Update TLS cert-manager Certificate to include `minio.web3-local-dev.com`
- [x] 3.5 Add Makefile targets for MinIO Ingress deploy/delete
- [x] 3.6 Verify MinIO accessible via `https://minio.web3-local-dev.com`

## 4. Backend (Go)

- [x] 4.1 Add `MinIOConfig` struct to `config.go`
- [x] 4.2 Add MinIO config to Kustomize ConfigMap (`configmap-config.yaml`)
- [x] 4.3 Create `storage_service.go` — MinIO Go SDK, presigned URL, metadata upload
- [x] 4.4 Create `storage_handler.go` — REST endpoints (`/presigned-url`, `/metadata`, `/erc20-icon`)
- [x] 4.5 Wire storage routes in server setup
- [x] 4.6 Add `github.com/minio/minio-go/v7` dependency
- [x] 4.7 Verify presigned URL generation via curl

## 5. Frontend (DashboardPage)

- [x] 5.1 Add image upload state variables (`uploadFile`, `uploadPreview`, `uploadProgress`)
- [x] 5.2 Add NFT metadata inputs (`nftName`, `nftDescription`)
- [x] 5.3 Implement image upload drop zone component (inline)
- [x] 5.4 Render upload field conditionally (Deploy ERC-20 icon, Mint ERC-721/1155 artwork)
- [x] 5.5 Implement presigned URL upload flow (request URL → PUT binary → status)
- [x] 5.6 Wire mint flow: upload image → generate metadata → execute mint
- [x] 5.7 Wire deploy flow: deploy → upload icon → update Blockscout DB
- [x] 5.8 Propagate changes to `app-2`, `app-3`, `app-4`

## 6. Seed Data Updates

- [x] 6.1 Update `seed/upload.sh` bucket paths to new contract-address-scoped structure
- [x] 6.2 Update `seed/update-blockscout-icons.sh` to use Ingress URLs

## 7. Verification

- [x] 7.1 Presigned URL generation + upload via curl
- [x] 7.2 MinIO Ingress serves assets via `https://minio.web3-local-dev.com`
- [x] 7.3 Blockscout fetches metadata via internal DNS (tokenURI)
- [x] 7.4 Browser renders images via external Ingress URL
- [x] 7.5 E2E: frontend upload → mint → Blockscout shows image

## Status: ✅ DONE
- [x] 8.1 Write deep-dive documentation into documents/web2-smart-wallet-bridge/erc721-erc1155-metadata-quirks.md
