# Frontend ERC-20 UX & Decimals Handling Tasks

## 1. Documentation & Specification
- [x] 1.1 Draft OpenSpec (`openspec/specs/frontend-erc20-ux/spec.md`)
- [x] 1.2 Create task tracking document (`openspec/changes/frontend-erc20-ux/tasks.md`)

## 2. Shared Data Fetching (Metadata)
- [x] 2.1 Identify where the frontend currently fetches user token balances (e.g. Blockscout API, viem multicall, or custom indexer).
- [x] 2.2 Update the fetching logic to strictly require querying and storing `decimals` alongside the raw `balanceOf`.
- [x] 2.3 Refactor the token portfolio state to retain `{ address, balanceRaw, decimals, symbol }` objects.

## 3. UI Component Updates (Display)
- [x] 3.1 Locate backend/frontend components responsible for rendering the user's portfolio or account balances.
- [x] 3.2 Implement `ethers.formatUnits(balanceRaw, decimals)` to display balances, replacing any hardcoded `parseEther` or `/ 1e18` logic.
- [x] 3.3 Add proper human-readable formatting (thousands separators, truncating to 4-6 decimal places for display purposes if decimals >= 18).

## 4. Transfer Input Form & Dropdown (Parsing)
- [x] 4.1 Implement a **Token Selection Dropdown** that iterates over the cached `{ address, decimals, symbol, balanceRaw }` portfolio fetched in Step 2.
- [x] 4.2 Restrict the user from bypassing the dropdown and pasting arbitrary addresses into the `amount` calculation unless the UI explicitly queries the new address for its `decimals`.
- [x] 4.3 Locate the specific token transfer modal or input field component.
- [x] 4.4 Enforce maximum decimal length validation in the input field dynamically based on the *currently selected token's* `decimals`. (Show red error text if length is exceeded).
- [x] 4.5 Replace the transaction builder payload value conversion to dynamically use `ethers.parseUnits(inputString, selectedToken.decimals)`.

## 5. "MAX" Button Feature
- [x] 5.1 Implement a "MAX" button in the token transfer flow.
- [x] 5.2 Bind the "MAX" button to pull `balanceRaw` directly into the Smart Contract Wallet's execution payload.
- [x] 5.3 Ensure clicking "MAX" visually updates the text input properly via `formatUnits`.

## 6. Testing & Verification
- [x] 6.1 Fund a test account with a 2-decimal token (e.g. `20AAC`).
- [x] 6.2 Attempt to manually type an invalid decimal amount (e.g., `3.906`) and verify the UI validation explicitly rejects it.
- [x] 6.3 Attempt to transfer `10.0` tokens and verify the Smart Contract Wallet payload contains exactly `1000` base units without throwing an `ERC20InsufficientBalance` error.
- [x] 6.4 Click "MAX" and verify 100% of the token balance is successfully transferred.
