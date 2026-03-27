# Frontend ERC-721 and ERC-1155 UX Tasks

## 1. Documentation & Specification
- [x] 1.1 Draft OpenSpec (`openspec/specs/frontend-erc721-erc1155-ux/spec.md`)
- [x] 1.2 Create task tracking document (`openspec/changes/frontend-erc721-erc1155-ux/tasks.md`)

## 2. Shared Data Fetching (Metadata)
- [x] 2.1 Ensure the frontend currently fetches user ERC-721 and ERC-1155 balances (Already implemented in `portfolio` state via Blockscout API `token-balances`).

## 3. UI Component Updates: Mint and Transfer Dropdown
- [x] 3.1 Unify the dropdown logic in `DashboardPage.tsx` so that `Select Asset (Portfolio)` dropdown appears for ALL `activeToken` types (`ERC20`, `ERC721`, `ERC1155`) during both `transfer` and `mint` actions.
- [x] 3.2 Update the dropdown filter to use `p.type.replace('-', '') === activeToken` (e.g. mapping `ERC-721` to `ERC721`).
- [x] 3.3 Add a "Custom Contract Address..." fallback to the dropdown to allow users to manually enter a contract address, which is critical for minting newly deployed contracts not yet indexed in their portfolio.
- [x] 3.4 Render the `Target Token Address (0x)` input field when the "Custom Contract Address..." option is selected or when the user's portfolio is completely empty for that token type.
- [x] 3.5 Inject `ethers.js` on-chain ownership polling (`contract.owner()`) to computationally filter out unowned assets from the `Mint` dropdown to prevent transaction permission reverts.

## 4. Sub-field Updates (Amount & Token ID)
- [x] 4.1 Ensure the "MAX" button and `Amount` decimal validation dynamically adapt correctly for ERC-1155 (parsing with `selectedAsset.decimals`, typically 0).
- [x] 4.2 Verify that `Token Identifier (ID)` is always required for ERC-721 and ERC-1155 operations to pass to the SCW execution backend correctly.

## 5. Testing & Verification
- [x] 5.1 Click on Transfer for ERC721 and verify the dropdown populates.
- [x] 5.2 Click on Mint for ERC1155 and verify a Custom Address can be entered if needed.

## 6. Post-Implementation Bug Fixes
- [x] 6.1 Fix `Target Token Address` input visibility bug by ensuring `portfolio.some()` filters by the exact `activeToken` type instead of evaluating the entire global portfolio during tab switches.
- [x] 6.2 Fix `Mint To / Recipient Address` input visibility bug by decoupling the `interactTo` empty string state (`''` = `Self`) from the `Custom` interaction flow to prevent the dropdown from snapping back.
