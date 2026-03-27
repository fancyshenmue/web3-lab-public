# Web2.5 Smart Wallet Bridge Testing Guide

This document outlines how to test the components of the Web2.5 Smart Wallet Bridge (ERC-4337). Testing is divided into two phases: **Phase 1** (Testing current API logic with Mocks) and **Phase 2** (Full End-to-End Execution).

---

## Phase 1: Local API & Logic Testing (Current State)

The backend exposes two new API endpoints to handle Smart Wallet interactions natively. You can test these endpoints using `curl` or Postman while the Go API server and Geth node are running locally.

### 1. Test Smart Wallet Address Derivation
This tests the `SmartWalletService`'s ability to call the deployed `AccountFactory` contract over RPC to calculate the deterministic `CREATE2` address for a specific user.

**Endpoint:** `GET /api/v1/wallet/address/:account_id`

**Test Command:**
```bash
# Replace <account-uuid> with a valid UUID format
curl -X GET https://gateway.web3-local-dev.com/api/v1/wallet/address/1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d
```

**Expected Result:**
```json
{
  "wallet_address": "0x..." 
}
```
*(The returned value is the deterministically generated Smart Contract Wallet address.)*

### 2. Test Transaction Request (Execute Intent)
This tests the backend's ability to parse a high-level intent, construct a `UserOperation`, generate a (mocked) ZK Proof signature, and sign the `PaymasterAndData` payload.

**Endpoint:** `POST /api/v1/wallet/execute`

**Test Command:**
```bash
curl -X POST https://gateway.web3-local-dev.com/api/v1/wallet/execute \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
    "intent": "Mint NFT"
  }'
```

**Expected Result:**
```json
{
  "status": "success",
  "message": "UserOperation submitted successfully via ZK Prover and Paymaster.",
  "mock_proof": "0x00000000...ZKPF222...",
  "user_operation": {
    "sender": "0x...",
    "nonce": "0x1",
    "initCode": "0x",
    "callData": "0x00000000",
    "paymasterAndData": "0x5FbDB...abcd1234...",
    "signature": "0x00000000...ZKPF222..."
  },
  "transaction_hash": "0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d"
}
```
*(The transaction hash is now a live, executable Ethereum on-chain EIP-1559 payload.)*

---

## Phase 2: Full End-to-End (E2E) Verification (Future)

To test the bridge against a true blockchain environment where state is actually modified, the following integration steps must be completed and tested:

### 1. ZK Prover (TEE) Integration
- **Action:** Replace `h.walletService.GenerateZKProof` mock with an actual HTTP/gRPC call to the TEE Prover service (e.g., SP1 / RISC Zero node).
- **Verification:** The returned `signature` must be a cryptographically valid Plonk/Groth16 proof that the `AccountContract`'s `validateUserOp` function will accept.

### 2. Frontend Integration
- **Action:** Modify the React frontend. Instead of prompting MetaMask to sign a transaction, the "Mint" or "Transfer" buttons should send an authenticated HTTP POST request containing the user's Session Cookie directly to `POST /api/v1/wallet/execute`.
- **Verification:** The UI should show a loading state while the ZK Proof is generated, and eventually display a success modal with the resulting on-chain Transaction Hash once the Native API Bundler confirms execution.

---

## Phase 3: Isolated Token Ecosystem Validation

Once users are mapped to unique Deterministic Smart Contract Accounts, testing should focus on validating **cryptographic isolation** through native Factory deployments.

### 1. Token Deployment via Smart Wallet
Since the `msg.sender` of a bundled transaction is the Smart Wallet itself, deploying tokens via the Factory assigns exclusive minting capabilities to that specific Web2 User.

**Action:** Deploy an ERC-20 entirely gaslessly via the backend executor.

**Test Command (or Trigger via UI):**
```bash
curl -X POST https://gateway.web3-local-dev.com/api/v1/wallet/execute \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
    "action": "deploy_contract",
    "token_type": "ERC20",
    "name": "My Unique Test Token",
    "symbol": "MTT",
    "decimals": "18",
    "initial_supply": "1000000"
  }'
```

### 2. Minting on Self-Deployed Contracts
Because the Smart Wallet is the sole owner, executing a `mint(address)` operation on a peer's contract address should strictly revert!

**Verification Steps:**
1. Login as User A via Google OIDC.
2. Deploy an ERC-20 token using User A's abstract wallet. (Saves to On-Chain mapping).
3. From the dashboard dropdown, select `+(Enter Custom Contract Address)` and paste the deployed ERC-20 token address (Note: 0-balance freshly deployed contracts bypass the Blockscout portfolio indexer until their first mint). Mint +100 tokens successfully.
4. Login as User B via Email Identity.
5. Provide User A's token contract address into User B's UI and attempt a Mint action.
6. The frontend ownership evaluation will forcefully **hide** the token from User B's portfolio dropdown.
7. If User B attempts to bypass the UI constraint by forcibly using the `+(Enter Custom Contract Address)` option to submit a mint intent for User A's contract, the bundler/prover circuit will cleanly **FAIL** resolving `OwnableUnauthorizedAccount` because User B does not cryptographically control the deployed token infrastructure!
