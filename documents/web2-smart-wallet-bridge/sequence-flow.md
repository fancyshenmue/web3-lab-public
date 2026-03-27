# Web2.5 Smart Wallet Bridge — Sequence & TB Flow

## Table of Contents

- [1. Authentication & SCW Creation](#1-authentication--scw-creation)
- [2. Deploy Contract (ERC-20 / ERC-721 / ERC-1155)](#2-deploy-contract)
- [3. Mint Asset (ERC-20 / ERC-721 / ERC-1155)](#3-mint-asset)
- [4. Transfer Asset (ERC-20 / ERC-721 / ERC-1155)](#4-transfer-asset)
- [5. Unified UserOperation Pipeline](#5-unified-useroperation-pipeline)
- [6. On-chain Execution Flow](#6-on-chain-execution-flow)
- [7. Security Boundaries](#7-security-boundaries)

---

## 1. Authentication & SCW Creation

### Sequence Diagram

```mermaid
sequenceDiagram
    actor User
    participant Browser as Dashboard (React)
    participant Hydra as Ory Hydra
    participant Kratos as Ory Kratos
    participant API as Go Backend
    participant DB as PostgreSQL (account_db)
    participant Factory as AccountFactory
    participant Chain as Geth PoS

    Note over User,Chain: Login (OAuth2 / SIWE)
    User->>Browser: Click "Login"
    Browser->>Hydra: OAuth2 Authorization Request
    Hydra->>Kratos: Login Flow
    Kratos-->>Hydra: Session Cookie
    Hydra-->>Browser: Authorization Code
    Browser->>Hydra: Exchange Code for JWT
    Hydra-->>Browser: Access Token (JWT, sub=kratos_identity_id)
    Browser->>Browser: Store JWT in localStorage

    Note over User,Chain: Identity Resolution & SCW Derivation
    Browser->>API: GET /api/v1/wallet/address/{sub}
    API->>DB: SELECT identity_id FROM account_identities<br>WHERE kratos_identity_id = sub
    DB-->>API: identity_id (app-owned UUID)
    API->>API: GetDeterministicAccount(identity_id)<br>→ Map to Genesis EOA
    API->>Factory: eth_call getAddress(owner, salt=identity_id)
    Factory-->>API: CREATE2 predicted address
    API-->>Browser: wallet_address (0x...)
    Browser->>Browser: Display SCW + identity in sidebar

    Note over User,Chain: SCW Deployment (Lazy - on first UserOp)
    Browser->>API: POST /api/v1/wallet/execute (first tx)
    API->>Chain: eth_getCode(scw_address)
    Chain-->>API: code = 0x (not deployed)
    API->>API: GetInitCode(owner, salt=identity_id)<br>→ factory + createAccount calldata
    API->>API: Include initCode in UserOp
    Note right of API: SCW is deployed by EntryPoint<br>via CREATE2 during first handleOps
```

### TB Flow — Login & SCW Creation

| Step | Actor   | Action                  | Input                       | Output                           | Location        |
| :--- | :------ | :---------------------- | :-------------------------- | :------------------------------- | :-------------- |
| 1    | User    | Click Login             | —                           | Redirect                         | Browser         |
| 2    | Hydra   | OAuth2 flow             | Auth request                | Login UI                         | Off-chain       |
| 3    | Kratos  | Authenticate            | Email + Password            | Session                          | Off-chain       |
| 4    | Hydra   | Issue JWT               | Session                     | `access_token` (sub=kratos_uuid) | Off-chain       |
| 5    | Browser | Store token             | JWT                         | localStorage                     | Browser         |
| 6    | Browser | Fetch SCW address       | JWT `sub`                   | —                                | Browser         |
| 7    | API     | **Resolve identity**    | `kratos_identity_id`        | `account_identities.identity_id` | Off-chain (DB)  |
| 8    | API     | Map identity → EOA      | `identity_id`               | Genesis EOA address              | Off-chain       |
| 9    | Factory | Predict CREATE2 address | `owner`, `salt=identity_id` | SCW address                      | On-chain (view) |
| 10   | Browser | Display SCW + identity  | address, identity_id        | Sidebar widget                   | Browser         |

> **Note**: SCW is **not deployed** at login time. It is lazily deployed during the first `UserOperation` via `initCode`. The `getAddress` call is a view function that predicts the CREATE2 address without deploying.
>
> **Identity Resolution**: The JWT `sub` (Kratos UUID) is resolved to the app-owned `identity_id` via `account_identities.kratos_identity_id` lookup. This `identity_id` is used as the CREATE2 salt, making the SCW address auth-provider-independent.

---

## 2. Deploy Contract

### Sequence Diagram

```mermaid
sequenceDiagram
    actor User
    participant Browser as Dashboard
    participant API as Go Backend
    participant EP as EntryPoint
    participant PM as Paymaster
    participant SCW as Smart Wallet
    participant Factory as Token Factory

    User->>Browser: Fill form (name, symbol, type)
    Browser->>API: POST /wallet/execute<br/>{action: "deploy_contract",<br/>token_type: "ERC20/721/1155",<br/>name, symbol}

    Note over API: UserOp Construction
    API->>API: 1. Resolve owner EOA
    API->>API: 2. Derive SCW address
    API->>API: 3. Check code → build initCode if needed
    API->>API: 4. EncodeExecutionCall:<br/>SCW.execute(factory, 0, createToken(...))
    API->>API: 5. BuildUserOperation (nonce, gas)
    API->>API: 6. SignPaymasterData
    API->>API: 7. HashUserOp via EntryPoint
    API->>API: 8. GenerateZKProof (ECDSA sign)

    API->>EP: eth_sendTransaction(handleOps)
    EP->>SCW: validateUserOp (verify signature)
    SCW-->>EP: validation success
    EP->>PM: validatePaymasterUserOp
    PM-->>EP: sponsorship approved
    EP->>SCW: execute(factory, 0, createToken)
    SCW->>Factory: createToken(name, symbol, ...)
    Factory->>Factory: Deploy new token contract
    Factory-->>SCW: token contract address
    EP->>PM: postOp (settle gas)
    EP-->>API: tx receipt
    API-->>Browser: {status, tx_hash}
    Browser->>User: Show success + TX hash
```

### TB Flow — Deploy Contract

| Step | Component  | Action            | Target             | Calldata Signature                   |
| :--- | :--------- | :---------------- | :----------------- | :----------------------------------- |
| 1    | Browser    | Submit intent     | API                | `{action: "deploy_contract"}`        |
| 2    | API        | Resolve EOA       | SmartWalletService | `GetDeterministicAccount(uuid)`      |
| 3    | API        | Derive address    | Factory (view)     | `getAddress(owner, salt)`            |
| 4    | API        | Encode inner call | —                  | See below per token type             |
| 5    | API        | Wrap in execute   | —                  | `execute(factory, 0, innerCallData)` |
| 6    | API        | Build UserOp      | EntryPoint (view)  | `getNonce(sender, 0)`                |
| 7    | API        | Sign paymaster    | Local              | ECDSA sign                           |
| 8    | API        | Hash UserOp       | EntryPoint (view)  | `getUserOpHash(op)`                  |
| 9    | API        | Sign UserOp       | Local (ZK mock)    | ECDSA sign                           |
| 10   | EntryPoint | handleOps         | On-chain           | Validate + Execute                   |
| 11   | SCW        | execute → Factory | On-chain           | `createToken(...)`                   |

#### Deploy Inner Calldata by Token Type

| Token        | Factory Target       | Method Signature                           | Parameters                            |
| :----------- | :------------------- | :----------------------------------------- | :------------------------------------ |
| **ERC-20**   | `ERC20FactoryAddr`   | `createToken(string,string,uint8,uint256)` | name, symbol, decimals, initialSupply |
| **ERC-721**  | `ERC721FactoryAddr`  | `createNFT(string,string,string)`          | name, symbol, baseURI                 |
| **ERC-1155** | `ERC1155FactoryAddr` | `createMultiToken(string,string,string)`   | name, symbol, uri                     |

---

## 3. Mint Asset

### Sequence Diagram

```mermaid
sequenceDiagram
    actor User
    participant Browser as Dashboard
    participant API as Go Backend
    participant EP as EntryPoint
    participant SCW as Smart Wallet
    participant Token as Token Contract

    User->>Browser: Fill (token_address, amount, token_id)
    Browser->>API: POST /wallet/execute<br/>{action: "mint",<br/>token_type: "ERC20/721/1155",<br/>token_address, amount, token_id}

    Note over API: Same UserOp pipeline
    API->>API: Encode: SCW.execute(token, 0, mint(...))
    API->>API: Build + Sign + Submit

    EP->>SCW: validateUserOp → execute
    SCW->>Token: mint(to, amount/id)
    Token->>Token: _mint(to, ...)
    Token-->>SCW: success
    EP-->>API: tx receipt
    API-->>Browser: {tx_hash}
```

### TB Flow — Mint Asset

| Token        | Target         | Method Signature                      | Parameters                | Semantics                |
| :----------- | :------------- | :------------------------------------ | :------------------------ | :----------------------- |
| **ERC-20**   | Token contract | `mint(address,uint256)`               | to, amount×10¹⁸           | SCW must be owner/minter |
| **ERC-721**  | Token contract | `mint(address)`                       | to                        | Auto-increment tokenId   |
| **ERC-1155** | Token contract | `mint(address,uint256,uint256,bytes)` | to, tokenId, amount, data | Multi-token mint         |

> **Important**: `mint` is NOT `transfer`. Minting creates **new** tokens. The `from` address in the Transfer event is `0x0000...0000`. The calling SCW must have the `MINTER_ROLE` or be the contract owner.
>
> **Zero-Balance Indexing Caveat**: Newly deployed token contracts strictly contain a user balance of `0`. The API portfolio endpoint (which structurally omits 0-balance dependencies) will not index this contract until the initial mint occurs. The active frontend mitigates this by enforcing a `+(Enter Custom Contract Address)` dropdown fallback mechanism for all zero-to-one origin mints.

---

## 4. Transfer Asset

### Sequence Diagram

```mermaid
sequenceDiagram
    actor User
    participant Browser as Dashboard
    participant API as Go Backend
    participant EP as EntryPoint
    participant SCW as Smart Wallet (Sender)
    participant Token as Token Contract
    participant Recipient as Recipient Address

    User->>Browser: Fill (token_address, recipient, amount/id)
    Browser->>API: POST /wallet/execute<br/>{action: "transfer",<br/>token_type: "ERC20/721/1155",<br/>token_address, to, amount, token_id}

    Note over API: Same UserOp pipeline
    API->>API: Encode: SCW.execute(token, 0, transfer(...))
    API->>API: Build + Sign + Submit

    EP->>SCW: validateUserOp → execute
    SCW->>Token: transfer/transferFrom/safeTransferFrom
    Token->>Token: Check balanceOf(SCW) ≥ amount
    Token->>Recipient: Credit balance
    Token-->>SCW: success
    EP-->>API: tx receipt
    API-->>Browser: {tx_hash}
```

### TB Flow — Transfer Asset

| Token        | Target         | Method Signature                                          | Parameters                           | Precondition                     |
| :----------- | :------------- | :-------------------------------------------------------- | :----------------------------------- | :------------------------------- |
| **ERC-20**   | Token contract | `transfer(address,uint256)`                               | to, amount×10¹⁸                      | SCW balance ≥ amount             |
| **ERC-721**  | Token contract | `transferFrom(address,address,uint256)`                   | from(SCW), to, tokenId               | SCW owns tokenId                 |
| **ERC-1155** | Token contract | `safeTransferFrom(address,address,uint256,uint256,bytes)` | from(SCW), to, tokenId, amount, data | SCW balance ≥ amount for tokenId |

> **Failure Case**: If the SCW doesn't have sufficient balance, the on-chain call reverts with `ERC20InsufficientBalance` / `ERC721InsufficientApproval` / `ERC1155InsufficientBalance`. The top-level TX still succeeds (EntryPoint catches the revert), but the UserOperation is marked as failed via `UserOperationRevertReason` event.

---

## 5. Unified UserOperation Pipeline

All actions (deploy, mint, transfer) go through the **same** 8-step pipeline in the Go Backend:

```mermaid
flowchart TD
    A["1. GetDeterministicAccount(UUID)"] --> B["2. DeriveWalletAddress(owner, salt)"]
    B --> C{"3. eth_getCode(scw)<br>Deployed?"}
    C -->|No| D["BuildInitCode(factory+createAccount)"]
    C -->|Yes| E["initCode = 0x"]
    D --> F["4. EncodeExecutionCall(action, tokenType, ...)"]
    E --> F
    F --> G["5. BuildUserOperation(sender, callData, initCode)"]
    G --> H["6. SignPaymasterData(userOp)"]
    H --> I["7. HashUserOp via EntryPoint.getUserOpHash"]
    I --> J["8. GenerateZKProof(ECDSA sign hash)"]
    J --> K["SubmitToBundler(handleOps)"]

    style A fill:#1e40af,color:#fff
    style K fill:#16a34a,color:#fff
```

### Pipeline Detail Table

| #   | Step                 | Service                 | Method                                              | Description                                 |
| :-- | :------------------- | :---------------------- | :-------------------------------------------------- | :------------------------------------------ |
| 0   | **Resolve Identity** | DB (account_identities) | `SELECT identity_id WHERE kratos_identity_id = sub` | JWT sub → app-owned identity_id             |
| 1   | Map Identity         | SmartWalletService      | `GetDeterministicAccount`                           | identity_id mod N → Genesis EOA             |
| 2   | Derive Address       | SmartWalletService      | `DeriveWalletAddress`                               | Factory.getAddress(owner, salt=identity_id) |
| 3   | Check Deploy         | BundlerService          | `GetClient().CodeAt`                                | If code=0x → include initCode               |
| 4   | Encode Call          | BundlerService          | `EncodeExecutionCall`                               | Action-specific ABI encoding                |
| 5   | Build UserOp         | BundlerService          | `BuildUserOperation`                                | Nonce, gas limits, structure                |
| 6   | Paymaster Sign       | BundlerService          | `SignPaymasterData`                                 | Add paymaster sponsorship                   |
| 7   | Hash                 | BundlerService          | `HashUserOp`                                        | EntryPoint.getUserOpHash (EIP-712)          |
| 8   | Prove/Sign           | SmartWalletService      | `GenerateZKProof`                                   | ECDSA signature (ZK mock)                   |
| 9   | Submit               | BundlerService          | `SubmitToBundler`                                   | handleOps → on-chain TX                     |

---

## 6. On-chain Execution Flow

```mermaid
sequenceDiagram
    participant Bundler as Bundler EOA
    participant EP as EntryPoint
    participant Factory as AccountFactory
    participant SCW as Smart Wallet (Proxy)
    participant Impl as SimpleAccount (Impl)
    participant PM as Paymaster
    participant Target as Target Contract

    Note over Bundler,Target: First-time SCW (with initCode)
    Bundler->>EP: handleOps([userOp], beneficiary)
    EP->>Factory: createAccount(owner, salt)
    Factory->>Factory: CREATE2 → deploy Proxy
    Factory-->>EP: SCW address

    Note over Bundler,Target: Validation Phase
    EP->>SCW: validateUserOp(userOp, hash, missingFunds)
    SCW->>Impl: delegatecall validateUserOp
    Impl->>Impl: ecrecover(hash, signature)
    Impl->>Impl: require(signer == owner)
    Impl-->>EP: validationData = 0 (success)

    EP->>PM: validatePaymasterUserOp(userOp, hash, maxCost)
    PM->>PM: Verify paymaster signature
    PM-->>EP: context + validationData

    Note over Bundler,Target: Execution Phase
    EP->>EP: innerHandleOp (self-call)
    EP->>SCW: execute(target, value, calldata)
    SCW->>Impl: delegatecall execute
    Impl->>Target: call(calldata)
    Target-->>Impl: result
    Impl-->>EP: success

    Note over Bundler,Target: Settlement Phase
    EP->>PM: postOp(context, actualGasCost)
    EP->>Bundler: Transfer gas refund (ETH)
```

---

## 7. Security Boundaries

```mermaid
flowchart LR
    subgraph "Off-chain (Trusted)"
        Browser["Browser<br>(JWT holder)"]
        API["Go Backend<br>(Prover + Paymaster)"]
    end

    subgraph "On-chain (Trustless)"
        EP["EntryPoint<br>(Orchestrator)"]
        SCW["SCW<br>(User Wallet)"]
        PM["Paymaster<br>(Gas Sponsor)"]
        Token["Token Contract"]
    end

    Browser -->|"JWT + Intent"| API
    API -->|"Signed UserOp"| EP
    EP -->|"validateUserOp"| SCW
    EP -->|"validatePaymaster"| PM
    EP -->|"execute"| SCW
    SCW -->|"call"| Token

    style Browser fill:#1e3a5f,color:#fff
    style API fill:#1e3a5f,color:#fff
    style EP fill:#7c3aed,color:#fff
    style SCW fill:#16a34a,color:#fff
    style PM fill:#b45309,color:#fff
    style Token fill:#be123c,color:#fff
```

### Permission Model

| Attack Vector                                   | Protection Layer                 | Result                        |
| :---------------------------------------------- | :------------------------------- | :---------------------------- |
| A's SCW tries to spend B's SCW tokens           | ERC-20 `balanceOf` check         | ❌ `ERC20InsufficientBalance` |
| Forge another user's UserOp signature           | SCW `ecrecover` verification     | ❌ Invalid signer             |
| Call SCW.execute() directly (bypass EntryPoint) | `onlyEntryPointOrOwner` modifier | ❌ Unauthorized               |
| Submit UserOp without gas payment               | Paymaster signature validation   | ❌ Invalid paymaster sig      |
| Replay a previously used UserOp                 | EntryPoint nonce tracking        | ❌ Nonce already used         |
