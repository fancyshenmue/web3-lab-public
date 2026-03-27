# OpenSpec: Frontend ERC-721 and ERC-1155 Asset Handling & UX

## Status

Drafting 📝

## Context

Following the standardization of ERC-20 UX, it is necessary to extend a streamlined user experience to NFT (ERC-721) and Multi-Token (ERC-1155) operations. Currently, users must manually locate and paste the smart contract addresses for these assets when attempting to interact (mint or transfer).

This specification establishes a dropdown-driven UI for ERC-721 and ERC-1155 tokens, mirroring the ERC-20 UX, allowing users to dynamically select from their `portfolio` instead of copying raw addresses.

---

## 1. Portfolio Dropdown Selection

### Transfer Operations
When the user wants to transfer an ERC-721 or ERC-1155 asset, the application MUST present a **Token Selection Dropdown** sourced from the user's fetched portfolio.

- The dropdown must filter and display only assets matching the currently active token standard (e.g., `portfolio.filter(p => p.type === 'ERC-721')`).
- If the user's portfolio contains no assets of that standard, a fallback input field OR a descriptive empty state warning (e.g. "⚠️ No ERC-721 assets found in this wallet.") should be displayed, mirroring the ERC-20 behavior.

### Mint Operations
Minting strictly requires the user to hold the correct privileges (e.g., `owner` or `MINTER_ROLE`) on the target smart contract. Because the standard portfolio endpoint returns *all* tokens the user holds a balance of (including those they do not locally own), displaying the unfiltered portfolio dropdown for minting introduces permission conflicts.
- **On-Chain Ownership Polling**: The frontend MUST dynamically verify ownership by concurrently pinging the blockchain via `ethers.js` (`contract.owner()`) for every asset in the portfolio.
- **Dropdown Filtering**: The **Token Selection Dropdown** for minting MUST computationally exclude any portfolio asset where the explicit `owner()` does not strictly map to the user's active Smart Contract Wallet Address. 
- **Owned Indicator**: Verified assets MUST render a clear visual badge (e.g., `⭐ Owned`) inside the dropdown to assure the user the mint transaction will not predictably revert.
- **Fallback Mechanism**: To support minting from newly deployed contracts with a `0` balance (which won't naturally index in the Blockscout portfolio), the UI MUST explicitly provide a manual input field fallback option (e.g., `+(Enter Custom Contract Address)`).

---

## 2. Token ID and Amount Handling

Unlike ERC-20 tokens, ERC-721 and ERC-1155 assets rely fundamentally on `tokenId`.

### ERC-721 (Non-Fungible)
- **Amount**: Strictly `1`. The UI does not need to show an `Amount` input field.
- **Token ID**: For transfers, the user MUST specify the exact `tokenId` they wish to send. *(Note: The portfolio endpoint may only return the contract address and total balance, not the specific enumerated token IDs the user owns. Thus, manual `tokenId` input remains required).*
- **Minting**: Token ID is often auto-incremented by the contract, but the interface can allow an optional or disabled `tokenId` field depending on standard implementation.

### ERC-1155 (Multi-Token)
- **Amount**: Required. Can be any valid integer. ERC-1155 tokens generally lack decimals (i.e. decimal = 0), so scaling via `parseEther` is dangerous. 
- **Token ID**: Required for both Minting and Transferring.

---

## 3. Formatting & Parsing

- **Format**: When displaying balances for ERC-1155, format using the known `decimals` (usually 0) to avoid sub-unit display errors.
- **Parse**: Input amounts for ERC-1155 MUST be parsed strictly using `ethers.parseUnits(amount, decimals)`.

## 4. Implementation Invariants

1. **Active Standard Tracking**: The UI must instantly toggle between ERC-20, ERC-721, and ERC-1155 filtering based on the `activeToken` state.
2. **Unified Interface**: The `Select Asset (Portfolio)` dropdown must replace the `Target Token Address (0x)` manual input field for transfers, preventing manual address entry errors for tokens the user already holds.
3. **Dropdown State Isolation**: To prevent fallback UI input fields from erroneously hiding during tab switches, portfolio matching filters MUST computationally evaluate strictly against the scoped `activeToken` derived array (rather than the global portfolio representation).
4. **Custom Fallback State Resolution**: Explicit custom input dropdown triggers MUST be bound to distinct discrete string states (e.g., `'custom'`) rather than overloading empty strings (`''`), preventing fallback evaluation collisions with options like inherently empty `Self` payloads.
