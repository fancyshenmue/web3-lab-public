# Account Linking & Multi-SCW Implementation Tasks

## 1. Documentation & Specification
- [x] 1.1 Draft OpenSpec (`openspec/specs/account-linking/spec.md`)
- [x] 1.2 Create task tracking document (`openspec/changes/account-linking/tasks.md`)

## 2. Backend API Updates
- [ ] 2.1 Implement `GET /api/v1/accounts/me/identities` to list all linked identities for an account.
- [ ] 2.2 Implement `POST /api/v1/auth/siwe/link` to allow linking a new EOA to an active session.
- [ ] 2.3 Implement `DELETE /api/v1/accounts/me/identities/:identity_id` for unlinking.
- [ ] 2.4 Ensure Kratos webhook correctly merges Web2 accounts into the existing `account_id` if a session is present during OIDC flow.

## 3. Frontend UI Updates
- [ ] 3.1 Update Dashboard & Profile Page to fetch and iterate over `/api/v1/accounts/me/identities`.
- [ ] 3.2 For each identity, dynamically display the corresponding derived Smart Contract Wallet address.
- [ ] 3.3 Add "Link Wallet" button to trigger SIWE signature and call the new `/auth/siwe/link` endpoint.
- [ ] 3.4 Add "Link Google/Email" button to trigger Kratos `/self-service/settings/browser`.

## 4. Verification & E2E Testing
- [ ] 4.1 Log in with Google, verify 1 identity and 1 SCW is displayed.
- [ ] 4.2 Click "Link Wallet", sign SIWE message, verify a 2nd identity appears with a mathematically distinct SCW.
- [ ] 4.3 Log out, log in via MetaMask directly, verify access to the exact same unified `account_id` and both SCWs are visible.
