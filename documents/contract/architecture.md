# Web3Lab Smart Contract Architecture

This document outlines the high-level architecture of the Web3Lab Smart Contract stack, specifically highlighting the Factory Deployment models and the EIP-4337 Account Abstraction execution layers.

## 1. Directory Structure

To adhere to industry-standard EVM development patterns, the contracts are strictly separated into discrete standard directories:

```text
contracts/
├── contracts/
│   ├── erc4337/
│   │   ├── Web3LabEntryPoint.sol       (EIP-4337 Core)
│   │   ├── Web3LabAccount.sol          (Smart Contract Wallet)
│   │   ├── Web3LabAccountFactory.sol   (SCW Generator)
│   │   └── Web3LabPaymaster.sol        (Gas Sponsorship)
│   ├── erc20/
│   │   ├── Web3LabERC20.sol
│   │   └── Web3LabERC20Factory.sol
│   ├── erc721/
│   │   ├── Web3LabERC721.sol
│   │   └── Web3LabERC721Factory.sol
│   └── erc1155/
│       ├── Web3LabERC1155.sol
│       └── Web3LabERC1155Factory.sol
```

## 2. High-Level Dependency Graph (Top-Bottom)

```mermaid
graph TB
    subgraph EIP-4337 Core
        EP[EntryPoint]
    end

    subgraph Factory Layer
        AF[Web3LabAccountFactory]
        F20[Web3LabERC20Factory]
        F721[Web3LabERC721Factory]
        F1155[Web3LabERC1155Factory]
    end

    subgraph Deployed Instances
        SCW[Web3LabAccount / SCW]
        PM[Web3LabPaymaster]
        T20[Web3LabERC20 Token]
        T721[Web3LabERC721 NFT]
        T1155[Web3LabERC1155 MultiToken]
    end

    AF -- "Predicts & Deploys" --> SCW
    SCW -- "Trusts & Validates via" --> EP
    AF -- "Registers EP into" --> SCW
    
    PM -- "Sponsors Gas for" --> EP
    
    F20 -- "Deploys" --> T20
    F721 -- "Deploys" --> T721
    F1155 -- "Deploys" --> T1155
    
    SCW -. "Executes Payloads on" .-> T20
    SCW -. "Executes Payloads on" .-> T721
    SCW -. "Executes Payloads on" .-> T1155
```

## 3. Interaction Sequence: SCW Wallet Creation & Asset Transfer

The integration tests and application backend interact with the Smart Contracts solely by executing internal calls via the proxy nature of the SCW. The following sequence demonstrates Wallet creation and subsequent ERC-20 token execution.

```mermaid
sequenceDiagram
    autonumber
    actor Alice as Alice (EOA)
    participant AF as AccountFactory
    participant SCW as Alice's SCW
    participant TF as ERC20 Factory
    participant T20 as New ERC20 Token
    participant Bob as Bob's SCW (Recipient)

    Note over Alice: Wallet Generation
    Alice->>AF: getAddress(Alice, salt=0)
    AF-->>Alice: Returns computed address (0xSCW_A)
    Alice->>AF: createAccount(Alice, 0)
    AF->>SCW: Deploys SCW with Owner=Alice

    Note over Alice: ERC20 Factory Deployment
    Alice->>TF: createToken("Lab Token", "LAB")
    TF->>T20: Instantiates new Web3LabERC20
    TF-->>Alice: Returns Token Address (0xT20)

    Note over Alice: Direct Minting to SCW
    Alice->>T20: mint(SCW_A, 100)
    T20-->>SCW: Transfers 100 Tokens to SCW_A

    Note over Alice, SCW: EIP-4337 Payload Execution
    Alice->>SCW: execute(0xT20, value=0, transfer(Bob, 50))
    activate SCW
    SCW->>T20: transfer(Bob, 50)
    activate T20
    T20-->>Bob: Credits 50 Tokens
    T20-->>SCW: Success
    deactivate T20
    SCW-->>Alice: Execution Complete
    deactivate SCW
```
