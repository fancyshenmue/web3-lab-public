# ERC-721 and ERC-1155 Metadata Indexing Quirks

During the integration of decentralized Object Storage (MinIO) with the Smart Contract Wallet Bridge and Blockscout Indexer, three highly nuanced, "tricky" edge cases were identified and resolved.

This document serves as an architectural post-mortem and reference guide for future development involving NFT metadata generation, smart wallet batched transactions, and Blockscout indexing.

## 1. Blockscout SSRF Internal IP Filtering

**Symptom**: ERC-721 and ERC-1155 tokens were successfully minted, and the on-chain `tokenURI` perfectly pointed to the local Kubernetes MinIO cluster (`http://minio.web3.svc.cluster.local...`), yet Blockscout constantly displayed "There is no metadata for this NFT."

**Root Cause**: Blockscout's Elixir indexer employs a strict **Server-Side Request Forgery (SSRF)** security model. By default, any requested URL that resolves to an internal or private IP address (e.g., `10.x.x.x` or loopback) is silently dropped and blacklisted to prevent malicious smart contracts from scanning internal cluster firewalls.

**Resolution**: To allow Blockscout to seamlessly index MinIO within the same Kubernetes cluster, SSRF URL-to-IP resolution discarding must be explicitly bypassed via an environment variable injected into the `blockscout-backend` deployment:

```yaml
- name: INDEXER_TOKEN_INSTANCE_HOST_FILTERING_ENABLED
  value: "false"
```

## 2. ERC-1155 `{id}` Zero Padding Standard (EIP-1155)

**Symptom**: ERC-721 metadata fetched flawlessly after the SSRF patch, but ERC-1155 metadata still returned a `404 Not Found` within Blockscout despite the MinIO files physically existing.

**Root Cause**: Unlike ERC-721 (which dynamically appends the raw token ID string to a base URI), the EIP-1155 specification strictly dictates that clients (like Blockscout, OpenSea, or Wallets) looking to replace the `{id}` substitution macro in a URI **MUST replace it with a 64-character lowercase zero-padded hexadecimal string**.
Our frontend uploaded images mathematically as raw decimals (`0.png`, `0.json`), causing Blockscout's strictly compliant `0000000000000000000000000000000000000000000000000000000000000000.json` request to perfectly miss the bucket key.

**Resolution**: We centralized this conversion in the Go Backend's MinIO storage handler (`storage_service.go`). For all ERC-1155 requests, the raw decimal string is parsed via `math/big` and left-padded using the `%064x` format directive prior to generating Presigned URLs and JSON metadata.

```go
if parsedID, ok := new(big.Int).SetString(tokenID, 10); ok {
    paddedTokenID = fmt.Sprintf("%064x", parsedID)
}
```

## 3. Account Abstraction (`Entrypoint`) Revert Silencing

**Symptom**: During contract deployments via the Smart Wallet Bridge, the Factory deployment succeeded, but the subsequent `set_uri` binding randomly failed, leaving the contract bound to its global default URI (`https://api.web3lab.com/...`).

**Root Cause**: The React frontend sends the `deploy_contract` and the `set_uri` transactions sequentially. However, because both are routed through an ERC-4337 Smart Wallet `UserOperation`, if the global system Paymaster is completely depleted of ETH (`AA31 paymaster deposit too low`), the secondary `set_uri` inner transaction will revert.
Crucially, because the Outer transaction pipeline isn't fundamentally broken, the deployment transaction itself does not broadcast a standard failure response to the frontend, leading to silent metadata disconnection.

**Resolution**: The cluster `Paymaster` contract must be rigorously monitored and funded (`make fund-paymaster`) to prevent silent secondary transaction drops within the Smart Wallet Bridge execution lifecycle.

## 4. Post-Deploy `set_uri` Overwrites Contract Address Display

**Symptom**: After deploying an ERC-721 or ERC-1155 contract, the deployed contract address briefly appeared in the UI but then disappeared. The displayed TX hash also changed from the deploy transaction to a different, unrelated-looking transaction. ERC-721 appeared to work intermittently, while ERC-1155 consistently failed to display the contract address.

**Root Cause**: The deploy flow triggers two sequential transactions:

1. `executeTransaction('deploy_contract')` — deploys the token contract via the Factory.
2. `executeTransaction('set_uri')` — binds the MinIO metadata URI to the new contract.

Both calls shared the `executeTransaction()` function, which unconditionally resets UI state at the start:

```javascript
setTxLoading(true);
setTxResult(null); // ← Clears the deploy TX hash
setError(null);
setDeployedContractAddress(null); // ← Clears the contract address
```

When `set_uri` completed, it overwrote `txResult` with the `set_uri` response (a completely different TX hash). The timing difference between ERC-721 and ERC-1155 `set_uri` execution caused ERC-721 to intermittently display correctly (due to React's automatic state batching merging the restore before commit), while ERC-1155's longer execution allowed the overwrite to persist.

**Resolution**: The `set_uri` call was refactored from `await executeTransaction('set_uri')` into a fire-and-forget `fetch()` call that runs silently in the background without touching any React UI state (`txResult`, `txLoading`, `deployedContractAddress`). The deploy TX hash and contract address now remain stable and visible regardless of `set_uri` timing.

```javascript
// Fire-and-forget — does NOT modify UI state
fetch(`${gatewayUrl}/api/v1/wallet/execute`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
  body: JSON.stringify({ action: 'set_uri', token_type: activeToken, token_address: contractAddr, ... })
}).catch(err => console.error('Background set_uri failed:', err));
```
