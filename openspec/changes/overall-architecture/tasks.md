## Phase 1: Project Setup

- [x] 1.1 Monorepo workspace (root `package.json` + workspaces)
- [x] 1.2 Node v22.x via Pixi environment
- [x] 1.3 Root Makefile with lifecycle targets
- [x] 1.4 Versioning system (`VERSION` file, `bump-{major,minor,patch}`)

## Phase 2: Smart Contracts

- [x] 2.1 Hardhat project setup (`contracts/hardhat.config.ts`)
- [x] 2.2 ERC-4337 contracts (`Web3LabAccount`, `Web3LabAccountFactory`, `Web3LabPaymaster`, `Web3LabEntryPoint`)
- [x] 2.3 ERC-20 contracts (`Web3LabERC20`, `Web3LabERC20Factory`)
- [x] 2.4 ERC-721 contracts (`Web3LabERC721`, `Web3LabERC721Factory`)
- [x] 2.5 ERC-1155 contracts (`Web3LabERC1155`, `Web3LabERC1155Factory`)
- [x] 2.6 Deployment scripts (`deploy.js`, `test-interact.js`)
- [x] 2.7 Unit tests (`deploy.test.ts`)
- [x] 2.8 Makefile targets (`compile-contracts`, `test-contracts`, `deploy-contracts`, `test-interact`)

## Phase 3: Frontend

- [ ] 3.1 Next.js User Portal (`frontend/user-portal/`)
- [ ] 3.2 Admin Dashboard (`frontend/admin-dashboard/`)
- [ ] 3.3 Social Login integration (Google, Apple via Ory Kratos)
- [ ] 3.4 Session Key generation (ephemeral ECDSA key pair)
- [ ] 3.5 UserOp builder and signing

## Phase 4: Backend

- [/] 4.1 Go module setup (`backend/go.mod`)
- [ ] 4.2 API Service (`backend/cmd/api/`)
- [ ] 4.3 Bundler Service (`backend/cmd/bundler/`)
- [ ] 4.4 Paymaster Service (`backend/cmd/paymaster/`)
- [x] 4.5 Mock Price API (`backend/cmd/mock-price/`) — completed as separate spec

## Phase 5: Infrastructure (Auth & Authz)

- [ ] 5.1 Ory Kratos (Identity / Social Login)
- [ ] 5.2 Ory Hydra (OAuth2 / OIDC)
- [ ] 5.3 Ory Oathkeeper (API Gateway)
- [ ] 5.4 SpiceDB (Permissions / ReBAC)
- [ ] 5.5 Redis deployment

## Status: ⏳ IN PROGRESS (Phase 1–2 complete)
