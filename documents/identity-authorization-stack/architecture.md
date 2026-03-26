# Identity & Authorization Architecture

The Web3-Lab platform relies on a decoupled, microservice-oriented identity and authorization stack based primarily on the Ory ecosystem (Kratos, Hydra, Oathkeeper) combined with AuthZed SpiceDB.

## High-Level Architecture (Top-Bottom)

```mermaid
graph TB
    Client((Web Client / Wallet App))
    Frontend["Frontend App<br>app.web3-local-dev.com"]

    subgraph "Ingress Layer (TLS via Cert-Manager)"
        APISIX_ING["NGINX Ingress<br>gateway.web3-local-dev.com<br>(unified entrypoint)"]
        FE_ING["NGINX Ingress<br>app.web3-local-dev.com<br>(frontend)"]
        OK_Proxy_Direct["Oathkeeper Direct<br>auth.web3-local-dev.com<br>(legacy)"]
        API_Direct["Web3 API Direct<br>api.web3-local-dev.com<br>(legacy)"]
    end

    subgraph "API Gateway (APISIX)"
        GW["APISIX Gateway<br>rate-limit, cors, key-auth, prometheus<br>Path-based routing"]
    end

    subgraph "Identity & Access Control (Web3 Namespace)"
        direction LR
        API["Backend API<br>(Go/Gin)"]
        KR["Ory Kratos<br>(Identity & Sessions & OIDC)"]
        HY["Ory Hydra<br>(OAuth2 & OIDC)"]
        SP["AuthZed SpiceDB<br>(Fine-Grained AuthZ)"]
        OK_API["Oathkeeper API<br>(Rules Engine)"]
    end

    subgraph "External Providers"
        Google["Google OIDC<br>(accounts.google.com)"]
    end

    subgraph "ERC-4337 Smart Wallet Bridge"
        direction LR
        ZK["ZK Prover / TEE<br>(Generates Proof from JWT)"]
        BM["Bundler Node<br>(Mempool & Relay)"]
        EntryPoint["Geth Smart Contracts<br>(EntryPoint, Paymaster, Factory)"]
    end

    subgraph "Datastores"
        PG[(Shared PostgreSQL)]
        RD[(Redis<br>Nonce Context)]
    end

    %% Frontend
    Client --> Frontend
    Frontend -->|"HTTPS"| FE_ING

    %% APISIX gateway flow (primary)
    Client -->|"HTTPS (gateway.web3-local-dev.com)"| APISIX_ING
    APISIX_ING --> GW
    GW -->|"/identity/*"| KR
    GW -->|"/oauth2/*"| HY
    GW -->|"/userinfo, /.well-known/*"| HY
    GW -->|"/auth/*"| OK_API
    GW -->|"/api/*"| API
    GW -->|"/admin/* (key-auth)"| KR
    GW -->|"/admin/* (key-auth)"| HY
    GW -->|"/admin/* (key-auth)"| SP

    %% Legacy direct flows
    Client -.->|"legacy direct"| OK_Proxy_Direct
    Client -.->|"legacy direct"| API_Direct
    OK_Proxy_Direct -.-> OK_API
    API_Direct -.-> API

    %% Backend direct interactions
    API -->|Generate/Verify Challenge| RD
    API -->|Create Session| KR
    API -->|"OAuth2 Login/Consent/Logout"| HY
    API -->|Write Permissions| SP

    %% Kratos OIDC
    KR -->|"Social Login"| Google
    KR -->|"Registration Webhook"| API

    %% Smart Wallet Bridge
    API -->|"1. Generate Proof (UserOp + Global Account_ID)"| ZK
    API -->|"2. Sign Paymaster & Forward"| BM
    BM -->|"3. eth_sendUserOperation"| EntryPoint

    %% Storage
    KR --> PG
    HY --> PG
    SP --> PG
    API --> PG
```

> [!NOTE]
> The dashed lines represent **legacy direct ingress** paths (`*.web3-local-dev.com`) that are kept for backward compatibility. The primary recommended path is through the APISIX gateway at `gateway.web3-local-dev.com`.

## Component Responsibilities

| Component           | Technology         | Primary Responsibility                                                                                                                                                                                                                                         |
| :------------------ | :----------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **API Gateway**     | Apache APISIX      | Unified API gateway at `gateway.web3-local-dev.com`. Routes by path prefix, applies rate limiting (`limit-req`), admin key authentication (`key-auth`), CORS, and Prometheus metrics. Also routes `/userinfo` and `/.well-known/*` to Hydra for OIDC discovery. |
| **Identity**        | Ory Kratos         | Manages user identities, secure sessions, and registration flows. Supports Google OIDC social login via `SELFSERVICE_METHODS_OIDC_CONFIG_PROVIDERS` env var. The backend API orchestrates wallet signatures to Kratos identity sessions.                        |
| **OAuth2 Provider** | Ory Hydra          | An OAuth 2.0 and OpenID Connect provider. Issues access/refresh tokens, handles login/consent/logout challenges (delegated to Backend API), and exposes `/userinfo` at root path. `URLS_SELF_ISSUER` must NOT include `/oauth2` suffix.                          |
| **Backend API**     | Go/Gin             | Handles OAuth2 login/consent/logout webhook callbacks from Hydra, auto-accepts consent for first-party clients, manages app client configs (cached in Redis), and coordinates wallet auth challenge/verify flows.                                                |
| **Authorization**   | AuthZed SpiceDB    | Implements scalable Relationship-Based Access Control (ReBAC) based on Google Zanzibar. Calculates whether "Subject X can perform Action Y on Resource Z".                                                                                                     |
| **Edge Proxy**      | Ory Oathkeeper     | Sits at the edge of internal APIs, authenticating requests by resolving cookies (Kratos) or tokens (Hydra), checking permissions (SpiceDB), and proxying valid traffic forward.                                                                                |
| **Frontend**        | React/Vite         | Single-page app at `app.web3-local-dev.com` with login (email, Google OIDC, SIWE), OAuth2 callback, profile page (fetches `/userinfo`), and dashboard.                                                                                                        |
| **Datastores**      | PostgreSQL & Redis | Shared Postgres isolates schemas for Hydra, Kratos, SpiceDB and the application. Redis caches app client configs and manages ephemeral data like Wallet Authentication nonces.                                                                                 |
| **ZK Prover**       | TEE / Microservice | (Web2.5 Bridge) Generates Groth16/Plonk proofs affirming that a given UserOperation intent was cryptographically authorized by a valid authentication session mapped to a Global Account ID. |
| **Bundler**         | Go-Ethereum Node   | (Web2.5 Bridge) Collects EIP-4337 UserOperations, bundles them into standard Ethereum transactions, pays native gas fees (refunded by Paymaster), and submits them to the EntryPoint contract. |
