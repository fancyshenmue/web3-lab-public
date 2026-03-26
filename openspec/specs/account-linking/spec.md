# Account Linking & Multi-SCW Support

## Overview

In the Web3 Lab Architecture, a single user represents one **Account**. However, each user can authenticate using multiple **Account Identities** (e.g., Google OAuth, Email/Password, EOA Wallet via SIWE). 

Because the Smart Contract Wallet (SCW) address derivation salt is tied strictly to the `account_identities.identity_id` (the app-owned identity PK) and NOT the global `account_id`, **each linked login method generates a mathematically distinct SCW**.

This specification outlines the requirements for the backend and frontend to support linking multiple identities and displaying their respective SCWs.

---

## 1. Architectural Rules

1. **1-to-1 Mapping**: `1 Account Identity (Provider + Provider User ID)` = `1 identity_id` = `1 SCW Address`.
2. **Account Grouping**: Many `account_identities` can map to `1 Account` via the `account_id` foreign key.
3. **Asset Isolation**: Linking a new identity to an existing account does **not** merge their SCWs. The assets remain isolated in their respective SCWs.
4. **Primary Identity**: One identity is marked as `is_primary = true`. This is usually the identity that was used to create the account.

---

## 2. Backend Requirements

The backend must expose APIs allowing the frontend to view and manage all identities associated with the currently authenticated account.

### 2.1 Fetch Linked Identities
**Endpoint:** `GET /api/v1/accounts/me/identities`

Returns a list of all active (`unlinked_at IS NULL`) identities for the current `account_id`.

**Response:**
```json
[
  {
    "identity_id": "uuid-1",
    "provider_id": "google",
    "provider_user_id": "10423...",
    "display_name": "Alice Google",
    "is_primary": true,
    "linked_at": "2026-03-26T10:00:00Z"
  },
  {
    "identity_id": "uuid-2",
    "provider_id": "eoa",
    "provider_user_id": "0xABC...",
    "display_name": "Alice MetaMask",
    "is_primary": false,
    "linked_at": "2026-03-26T11:00:00Z"
  }
]
```

### 2.2 Link New EOA (SIWE)
Because EOA linking bypassing standard Kratos OIDC settings, a custom SIWE endpoint is needed to link an EOA to an *already authenticated* session.

**Endpoint:** `POST /api/v1/auth/siwe/link`
- **Auth**: Requires valid Bearer Token (JWT).
- **Body**: Standard EIP-4361 Payload (message + signature).
- **Behavior**: Validates signature → Creates new `account_identities` row with `provider_id='eoa'` mapped to the user's current `account_id`.

### 2.3 Unlink Identity
**Endpoint:** `DELETE /api/v1/accounts/me/identities/:identity_id`
- **Behavior**: Sets `unlinked_at = NOW()` (Soft delete).
- **Constraint**: Cannot unlink if it is the only remaining identity.

---

## 3. Frontend UI Requirements

The frontend Dashboard must be updated to display a "Linked Identities & Wallets" overview.

### 3.1 Profile/Dashboard Display
The user's profile should iterate over the response from `/api/v1/accounts/me/identities`.

For **each** identity, the UI MUST render:
- **Provider Icon/Name** (e.g., Google, MetaMask, Email)
- **Identity ID** (`account_identities.identity_id`)
- **Derived SCW Address** (The frontend will dynamically salt the Factory contract using the identity's `identity_id` to display the specific SCW).
- **External Address / Identifier** (`provider_user_id`)

### 3.2 Linking Flow Actions
- **Add EOA**: Button to "Link Wallet", triggers SIWE signing flow but targets the `/auth/siwe/link` endpoint.
- **Add Web2**: Button to "Link Google/Email". Redirects the user to the Ory Kratos `/self-service/settings/browser` flow to attach a new OIDC/Password connection to the active session.

---

## 4. Derived SCW Calculation

The frontend receives the list of identities and must calculate the SCW Address for each row locally. 

```typescript
const computeScwAddress = async (identityId: string) => {
  // 1. Get EOA signer (admin)
  // 2. Call factory.getAddress(signer.address, identityId)
  return await accountFactory.getAddress(signerAddress, identityId);
};
```
*Note: The frontend must show the unique SCW adjacent to each identity to clearly communicate asset isolation to the user.*
