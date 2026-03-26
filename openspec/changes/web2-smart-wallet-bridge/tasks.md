# Web2 to Smart Wallet Bridge (ERC-4337) Implementation Tasks

## 1. OpenSpec & Planning

- [x] 1.1 Design spec document (`openspec/specs/web2-smart-wallet-bridge/spec.md`)
- [x] 1.2 Create `openspec/changes/web2-smart-wallet-bridge/tasks.md` (this file)
- [x] 1.3 Review and finalize architecture flows

## 2. Smart Contract (Setup & Verification)

- [x] 2.1 Verify `AccountFactory` CREATE2 address derivation logic matches backend mapping requirements.
- [x] 2.2 Verify `Paymaster` contract has sufficient ETH deposit and correct signing logic for backend API `paymasterAndData`.
- [x] 2.3 Implement ZK Proof verification logic within the `validateUserOp` of the `Smart Contract Wallet` (or use a placeholder ZK verifier contract for now).

## 3. Backend (Identity & Address Mapping)

- [x] 3.1 Implement deterministic wallet address derivation (using `CREATE2` salts based on Kratos `Identity ID`).
- [x] 3.2 Add API endpoint to fetch a user's Smart Contract Wallet address based on their active Kratos session.
- [x] 3.3 Integrate ZK Prover / TEE microservice call: pass validated Kratos session/JWT and UserOpHash to get a ZK Proof.

## 4. Backend (UserOperation & Paymaster)

- [x] 4.1 Implement `UserOperation` builder (fetch nonce, estimate gas via Bundler RPC).
- [x] 4.2 Implement Paymaster Service logic: sign `paymasterAndData` natively in Go using a dedicated Paymaster private key.
- [x] 4.3 Implement `POST /api/v1/wallet/execute` endpoint for accepting frontend high-level intents.
- [x] 4.4 Assemble `UserOperation` + `ZK Proof` + `PaymasterData` and forward to Bundler RPC.

## 5. Frontend (UI & Flow)

- [x] 5.1 Implement "Sign in with Google / Email" using Kratos Redirection (already partially done).
- [x] 5.2 Remove "Connect Wallet" popup requirements for Social Login users.
- [x] 5.3 Implement intent-based transaction buttons (e.g., "Mint NFT", "Transfer") that directly call `POST /api/v1/wallet/execute`.
- [x] 5.4 Handle loading states while backend builds UserOp, generates ZK Proof, and waits for Bundler inclusion.
- [x] 5.5 Display resulting Transaction Hash and updated ERC20/721/1155 balances.

## 6. Testing & Documentation

- [x] 6.1 E2E Test: Social Login -> Call backend -> Generate Proof -> Bundler -> Transaction Success.
- [x] 6.2 Update existing diagrams in `documents/identity-authorization-stack/architecture.md` to reflect ZK Prover integration.
- [x] 6.3 Document security boundaries for the TEE / Backend Prover API.

## 7. Real On-Chain Bundler Execution

- [x] 7.1 Remove dummy transaction hash return in backend handler.
- [x] 7.2 Implement Go-Ethereum `abi.Pack` logic for `handleOps(ops[], beneficiary)`.
- [x] 7.3 Sign and execute a literal Ethereum transaction using the Paymaster EOA as the Bundler.
- [x] 7.4 Verify live transactions flowing through Blockscout.

## 8. Multi-Token Production-like E2E Demonstration (Factory Redesign)

- [x] 8.1 Refactor Backend `smart_wallet_service.go` mock ZK to deterministically slice 10 unique Genesis Private Keys bound to Kratos `Account ID`.
- [x] 8.2 **Smart Contract:** Modify `Web3LabERC*Factory` Solidity contracts to include an on-chain tracker (`mapping(address => address[]) public userDeployedContracts`) for robust token discovery.
- [x] 8.3 **Backend:** Expand `POST /api/v1/wallet/execute` to accept `action: "deploy_contract"` and ABI encode `createToken`/`createNFT` routed precisely to the token factories.
- [x] 8.4 **Backend:** Expand `POST /api/v1/wallet/execute` to parse dynamically executed Mint/Transfer interactions scoped to the newly generated Private Token Addresses.
- [x] 8.5 **Frontend UI:** Overhaul the React Dashboard into a Token deployment portal allowing users to instantiate tokens, view their owned contracts, and mint/transfer completely gaslessly.
- [x] 8.6 **Testing:** Verify 10 isolated Web2 logins can each seamlessly deploy their unique contract ecosystems without crossing boundaries.
- [x] 8.7 **Custom Factory Logic:** Expanded ERC20 factory deployments to support completely custom `decimals` and an upfront `initialSupply` mint directly to the abstract wallet.

## 9. Operational Debugging & Address Synchronization

- [x] 9.1 **ABI Encoding Fix**: Corrected `BundlerService.EncodeExecutionCall` to use `createToken(string,string,uint8,uint256)` instead of the obsolete `createToken(address)` selector.
- [x] 9.2 **ZK Signature EntryPoint Fix**: Removed hardcoded `EntryPoint` address (`0x5FbDB...`) from `HashUserOp` and `SignPaymasterData`; now dynamically reads `cfg.EntryPointAddr` to ensure signature validity.
- [x] 9.3 **Paymaster Funding**: Deposited ETH into `EntryPoint.depositTo(paymaster)` to resolve `AA31 paymaster deposit too low` reverts.
- [x] 9.4 **Contract Address Synchronization**: After a Geth chain reset, synchronized all contract addresses in `configmap-config.yaml` with the latest `contracts/deployments.json` to resolve `AA33 Sender not EntryPoint` mismatches.
- [x] 9.5 **Blockscout DB Reset**: Identified stale Blockscout Postgres data from pre-reset chain causing `ContractCode` fetcher crash loops; resolved by clearing and re-indexing.

## Status: ✅ COMPLETED
