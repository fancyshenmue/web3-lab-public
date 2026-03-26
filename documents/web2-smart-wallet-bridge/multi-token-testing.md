# Multi-Token Production-like Testing Guide

This document captures the operational guidelines for orchestrating the 10-user multi-token simulated production test over the Web2.5 Smart Wallet Bridge.

## Scenario Outline
Using the `1,000,000,000 labETH` pre-funded genesis EOAs defined in `documents/geth/chain-economics.md`, this test verifies complete isolation and batch functionality of ERC-4337.

10 individual Web2 email/social logins will each execute:
1. `Mint` 10 x ERC-20 test tokens 
2. `Mint` 10 x ERC-721 NFTs
3. `Mint` 10 x ERC-1155 Multi-Token assets
4. ERC-4337 Native Paymaster-sponsored transfers to peer recipients.

## Smart Contract Operations (Deployment)

To support this validation, standard generic test tokens will be provisioned directly onto the Geth PoS container exactly alongside the pre-existing infrastructure.

**Deployment Operations:**
1. Hardhat standard OpenZeppelin extensions will be crafted in `contracts/contracts/mocks/`:
   - `MockERC20.sol`
   - `MockERC721.sol`
   - `MockERC1155.sol`
2. The Hardhat deploy script (`contracts/scripts/deploy.js`) automatically triggers during the `contracts-deployer` Kubernetes job execution. It will be updated to orchestrate these standalone standard mock deployments instantly on cluster launch.
3. The resultant deployment addresses will be injected securely into the backend Kubernetes ConfigMap (`web3-api-config`) for deterministic dynamic routing without manual code edits.

## Backend Operations (Identity Routing)

The current monolithic prototype utilized a single test wallet identity. For production scale:
- The Go backend will be reprogrammed. Upon processing `POST /execute`, it calculates the integer hash of the incoming UUID `account_id` modulo 10.
- `H(UUID) % 10` effectively yields a static integer between 0 and 9, which identically maps the given Web2 login session to one of the 10 Pre-funded Hardhat genesis EOAs.
- The backend signs the transaction using exactly that designated EOA, isolating the derived `Web3LabAccount` creation correctly separating the 10 distinctive environments exclusively for their respective test users.

## API Payload Contract Layout
The frontend will interface with the newly expanded endpoint.
```json
// POST /api/v1/wallet/execute
{
    "account_id": "user-uuid...",
    "action": "mint | transfer",
    "token_type": "ERC20 | ERC721 | ERC1155",
    "to": "0xOptionalTargetAddress...",
    "amount_or_id": "100"
}
```

The Golang logic parses the specific intent, decodes the targeted OpenZeppelin payload parameters, hashes the encoded strings via standard standard `abi.Pack`, constructs the nested JSON payload internally, and relays exactly as it does currently.
