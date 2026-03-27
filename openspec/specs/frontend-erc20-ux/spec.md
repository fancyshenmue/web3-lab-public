# OpenSpec: Frontend ERC-20 Asset Handling & UX

## Status

Implemented ✅

## Context

The `web3-lab` decentralized application interacts with various ERC-20 tokens deployed via the `Web3LabERC20Factory`. Unlike native ETH, which strictly uses 18 decimals, custom ERC-20 tokens can have any number of decimals (usually between 0 and 18).

A common UX pitfall in Web3 development occurs when frontends incorrectly assume all tokens use 18 decimals. If a user attempts to transfer tokens and the frontend multiplies their input by `10^18` (e.g., using `ethers.parseEther()`), but the token only supports 2 decimals, the resulting transaction payload will demand a base unit amount far exceeding the user's actual balance. This leads directly to an `ERC20InsufficientBalance` execution revert from the smart contract.

This specification outlines the standard operating procedure for fetching, formatting, and processing user inputs for ERC-20 tokens to guarantee a consumer-friendly UI and prevent decimal-shift transaction reverts.

---

## 1. Data Fetching (Metadata Resolution)

When loading a user's token portfolio, the frontend must assemble a complete representation of each asset. **Querying `balanceOf(address)` in isolation is never sufficient.**

For every ERC-20 token, the frontend (or the backend indexer serving the frontend) MUST resolve the following tuple:
- `address`: The ERC-20 contract address.
- `balanceRaw` (BigInt): The raw base unit balance from the blockchain.
- `decimals` (number): The precision of the token (e.g., `2`, `6`, `18`).
- `symbol` (string): The ticker symbol for display.

### Recommended Fetching Strategy
To optimize RPC calls, frontends SHOULD use a `Multicall` contract or rely on a deeply indexed backend (like Blockscout APIs or a custom Go indexer) that pre-aggregates `decimals` alongside balances.

### Standard UX: The Token Dropdown
To provide a seamless experience, the UI SHOULD parse the user's SCW portfolio and render an **Asset Selection Dropdown** on the transfer page.
- **Preventing Errors**: Rather than making users manually paste ERC-20 contract addresses (which risks phishing and decimal mismatches), the dropdown guarantees the frontend already possesses the strictly verified `{ address, decimals, symbol, balanceRaw }` data for the selected asset.
- **Dynamic Re-binding**: When the user switches token selection in the dropdown, the transfer input's maximum value, decimal validation restrictions, and "MAX" button payload MUST instantly re-bind to the newly selected token's properties.

---

## 2. Formatting (Blockchain to UI)

When displaying balances or transaction amounts to the user, the app MUST convert the raw, unreadable base units into human-readable strings.

- **Conversion Rule**: Use the native ethers.js utility: `ethers.formatUnits(balanceRaw, decimals)`.
- **UI Localization**: The output string should be formatted using standard locale thousands separators (e.g., `10,000,000.00`).
- **Precision Truncation**: For tokens with 18 decimals, the UI SHOULD truncate or round to maximum 4-6 decimal places on standard portfolio views to avoid visual clutter (e.g., `1.0003...` instead of `1.000300400000000000`).

---

## 3. Parsing (UI to Blockchain) ⚠️ Critical

The core of the transaction failure issue lies in translating user-typed inputs back into blockchain payloads.

### Input Validation
The `<input>` component handling token amounts MUST enforce a strict decimal policy based on the token's loaded `decimals` property.

1. **Step 1: Read String Input**: The UI should capture the raw string the user types (e.g., `"3.9"`).
2. **Step 2: Decimal Bound Check**: Split the string by the decimal point. If the number of digits after the decimal exceeds the token's `decimals`, the UI MUST either:
   - **Auto-truncate**: Strip the excess digits immediately.
   - **Validation Error**: Turn the input red and display: *"Maximum precision for this token is {decimals} decimal places."*
3. **Step 3: Submit (Parse)**: When submitting the transaction payload, the frontend MUST encode the transfer amount using `ethers.parseUnits(userInputString, tokenDecimals)`, NOT `parseEther()`.

```javascript
// ✅ CORRECT: Dynamically parse using the specific token's decimals
const payloadAmount = ethers.parseUnits("3.9", 2);  // Returns 390n

// ❌ INCORRECT: Assuming 18 decimals blindly
const payloadAmount = ethers.parseEther("3.9"); // Returns 3900000000000000000n (Fails due to insufficient balance)
```

---

## 4. The "MAX" Button Implementation

To completely eliminate manual precision errors when users want to transfer their entire balance, a `MAX` button MUST be implemented on transfer forms.

- **Mechanism**: The MAX button bypasses user typing logic. It takes the cached `balanceRaw` (BigInt) from the blockchain and directly applies it to the transaction payload.
- **UI Reflection**: Simultaneously, it uses `ethers.formatUnits(balanceRaw, decimals)` to populate the visible input box so the user understands what is being sent.
- **Benefit**: This guarantees that the `amount` in the transaction perfectly matches the user's `balanceOf` on-chain, preventing trailing dust or insufficient balance reverts.

---

## 5. Backend Payload Contract

For the dynamic frontend resolution to work flawlessly, the backend responsible for encoding the `UserOperation` execution payload (e.g., `bundler_service.go`) MUST adhere to a strict invariant: **Never scale ERC-20 quantities.**

1. **Frontend Responsibility**: The frontend calculates and submits the final, exact **Base Unit** string via `ethers.parseUnits()` (or fetching exact `balanceRaw`).
2. **Backend Responsibility**: The backend must treat the `amount` string as absolute. It MUST NOT perform naive conversions like `amount * 10^18` out of an assumption that all ERC-20 tokens are 18-decimal standard. It must directly pack the parsed integer into the smart contract caller bytes.

```go
// ✅ CORRECT: Directly encode the parsed integer
amount := new(big.Int)
amount.SetString(amountStr, 10)
callData = append(callData, common.LeftPadBytes(amount.Bytes(), 32)...)

// ❌ INCORRECT: Blindly assuming 18 decimals in the backend
amountWei := new(big.Int).Mul(amount, big.NewInt(1e18))
```
