## 1. OpenSpec

- [x] 1.1 Create `openspec/specs/smart-contract-assets/spec.md` with architecture and deployment order
- [x] 1.2 Create `openspec/changes/smart-contract-assets/tasks.md` (this file)

## 2. Hardhat Setup

- [x] 2.1 Hardhat config (`hardhat.config.ts`) — Solidity 0.8.28, optimizer 200 runs, viaIR
- [x] 2.2 `localGeth` network (chain 72390, RPC via env var)
- [x] 2.3 OpenZeppelin + Account Abstraction dependencies

## 3. Core Contracts (ERC-4337)

- [x] 3.1 `erc4337/Web3LabAccount.sol` — Smart Contract Wallet (inherits SimpleAccount)
- [x] 3.2 `erc4337/Web3LabAccountFactory.sol` — CREATE2 factory for deterministic SCW deployment
- [x] 3.3 `erc4337/Web3LabPaymaster.sol` — Gas sponsor contract
- [x] 3.4 `Web3LabEntryPoint.sol` — EntryPoint contract

## 4. Asset Contracts

- [x] 4.1 `erc20/Web3LabERC20.sol` + `Web3LabERC20Factory.sol` — Fungible token with factory
- [x] 4.2 `erc721/Web3LabERC721.sol` + `Web3LabERC721Factory.sol` — NFT with factory
- [x] 4.3 `erc1155/Web3LabERC1155.sol` + `Web3LabERC1155Factory.sol` — Multi-token with factory

## 5. Scripts

- [x] 5.1 `deploy.js` — Deployment script (ordered: EntryPoint → Factory → ERC Factories → Paymaster)
- [x] 5.2 `test-interact.js` — Full AA lifecycle demo (SCW deploy, token mint, transfer)
- [x] 5.3 Output files: `deployments.json`, `seed-addresses.json`

## 6. Tests

- [x] 6.1 `deploy.test.ts` — Unit tests for deployment and AA lifecycle

## 7. Seed Data & Metadata

- [x] 7.1 `seed/images/` — Pre-generated artwork (ERC-20 logos, ERC-721 NFTs, ERC-1155 items)
- [x] 7.2 `seed/metadata/` — JSON metadata with MinIO URLs (`http://localhost:9000/web3lab-assets/...`)
- [x] 7.3 `seed/upload.sh` — Upload to MinIO with correct `Content-Type`
- [x] 7.4 `seed/update-blockscout-icons.sh` — Fix Blockscout DB (ERC-20 icons + ERC-721 metadata)

## 8. Makefile

- [x] 8.1 `compile-contracts` / `test-contracts` / `clean-contracts`
- [x] 8.2 `deploy-contracts` — Deploy to local Geth
- [x] 8.3 `test-interact` — Interactive simulation test
- [x] 8.4 `seed-upload` / `seed-update-icons`

## 9. Verification

- [x] 9.1 All contracts deployed to Geth PoS chain
- [x] 9.2 Token instances visible in Blockscout with correct metadata/icons
- [x] 9.3 ERC-721/ERC-1155 images render correctly via MinIO

## Status: ✅ ARCHIVED
