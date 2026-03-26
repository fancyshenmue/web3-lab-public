# Blockscout Explorer — Architecture

## Overview

Blockscout is an open-source EVM blockchain explorer deployed on Kubernetes. It indexes blockchain data from the Geth PoS cluster and serves it through a web UI.

## Component Architecture

```mermaid
graph TB
    subgraph Browser
        User["User Browser<br/>localhost:3001"]
    end

    subgraph K8s["Kubernetes — web3 namespace"]
        Proxy["Nginx Proxy<br/>port 80"]

        subgraph Services
            Frontend["Frontend<br/>Next.js v2.3.5+<br/>port 3000"]
            Backend["Backend<br/>Elixir/Phoenix v9.0.2+<br/>port 4000"]
            Stats["Stats<br/>Rust<br/>port 8050"]
        end

        Postgres[("PostgreSQL 16<br/>port 5432")]
    end

    subgraph Chain["Geth PoS Cluster"]
        GethRPC["geth-rpc<br/>HTTP :8545 / WS :8546"]
    end

    User -->|"port-forward 3001:80"| Proxy
    Proxy -->|"/*"| Frontend
    Proxy -->|"/api, /socket"| Backend
    Proxy -->|"/stats-api"| Stats

    Backend -->|"JSON-RPC"| GethRPC
    Backend -->|"read/write"| Postgres
    Stats -->|"read"| Postgres
    Stats -->|"API"| Backend
```

## Request Flow

```mermaid
sequenceDiagram
    participant B as Browser
    participant P as Nginx Proxy
    participant F as Frontend
    participant API as Backend
    participant S as Stats
    participant DB as PostgreSQL
    participant G as Geth RPC

    B->>P: GET localhost:3001/
    P->>F: proxy /
    F-->>B: HTML + JS

    B->>P: GET /api/v2/stats
    P->>API: proxy /api/*
    API->>DB: query indexed data
    DB-->>API: results
    API-->>B: JSON response

    B->>P: GET /api/v2/main-page/blocks
    P->>API: proxy /api/*
    API->>DB: query blocks
    DB-->>API: block data
    API-->>B: JSON response

    Note over API,G: Background indexing
    API->>G: eth_getBlockByNumber (JSON-RPC)
    G-->>API: block + transactions
    API->>DB: insert indexed data
```

## Components

| Component    | Image                                  | Role                                   |
| ------------ | -------------------------------------- | -------------------------------------- |
| **Postgres** | `postgres:16-alpine`                   | Shared database (blockscout + stats)   |
| **Backend**  | `ghcr.io/blockscout/blockscout:latest` | Indexer + REST/WS API (Elixir/Phoenix) |
| **Frontend** | `ghcr.io/blockscout/frontend:latest`   | Web UI (Next.js SSR)                   |
| **Stats**    | `ghcr.io/blockscout/stats:latest`      | Chart/analytics service (Rust)         |
| **Proxy**    | `nginx:alpine`                         | Reverse proxy — single entry point     |

> [!IMPORTANT]
> Backend and Frontend versions must be compatible. Backend v9.x pairs with Frontend v2.x. The Docker Hub `blockscout/blockscout:latest` image is outdated (v7.0.2) — always use `ghcr.io/blockscout/blockscout:latest`.

## Nginx Proxy Routing

The proxy eliminates CORS issues by serving all components on a single origin:

```mermaid
graph LR
    subgraph Proxy["Nginx :80"]
        R1["/api/v1/pages/main"] -->|"mock 200"| Mock["Static JSON"]
        R2["/api/*"] -->|"proxy_pass"| Backend["backend:4000"]
        R3["/socket/*"] -->|"WebSocket upgrade"| Backend
        R4["/stats-api/*"] -->|"rewrite + proxy"| Stats["stats:8050"]
        R5["/* (default)"] -->|"proxy_pass"| Frontend["frontend:3000"]
    end
```

| Path               | Destination            | Notes                                    |
| ------------------ | ---------------------- | ---------------------------------------- |
| `/api/v1/pages/main` | Mock 200 response    | Legacy endpoint removed in backend v9    |
| `/api/*`           | `backend:4000`         | All REST API calls                       |
| `/socket/*`        | `backend:4000`         | WebSocket with connection upgrade        |
| `/stats-api/*`     | `stats:8050`           | Path rewritten (`/stats-api/x` → `/x`)  |
| `/*`               | `frontend:3000`        | Default — serves the Next.js UI          |

## Data Flow

```mermaid
graph LR
    subgraph Indexing
        G["Geth Node"] -->|"JSON-RPC<br/>HTTP + WS"| B["Backend"]
        B -->|"INSERT blocks,<br/>txns, logs"| DB[("PostgreSQL")]
    end

    subgraph Serving
        DB -->|"SELECT"| B
        DB -->|"SELECT<br/>(stats DB)"| S["Stats"]
        S -->|"chart data"| B
        B -->|"REST API"| F["Frontend"]
    end
```

The backend continuously indexes the chain via JSON-RPC:

1. **Block fetcher** — polls `eth_getBlockByNumber` for new blocks
2. **Transaction fetcher** — fetches full transaction details + receipts
3. **Internal transaction tracer** — calls `debug_traceTransaction` for internal txns
4. **Token indexer** — detects ERC-20/721/1155 transfers from event logs

## Kubernetes Manifests

```
deployments/kubernetes/minikube/blockscout/
├── postgres.yaml    # StatefulSet + PVC + Service (port 5432)
├── blockscout.yaml  # Backend Deployment + Service (port 4000)
├── frontend.yaml    # Frontend Deployment + Service (port 3000)
├── stats.yaml       # Stats Deployment + Service (port 8050)
└── proxy.yaml       # Nginx ConfigMap + Deployment + Service (port 80)
```

## Database Schema

PostgreSQL hosts two logical databases:

| Database           | Used By          | Contents                                   |
| ------------------ | ---------------- | ------------------------------------------ |
| `blockscout`       | Backend          | Blocks, transactions, addresses, tokens    |
| `blockscout_stats` | Stats            | Aggregated chart data, daily metrics       |

Both databases are created automatically on first startup via `CREATE_DATABASE=true` and `STATS__CREATE_DATABASE=true`.

## Dependencies

```mermaid
graph BT
    Postgres --> Backend
    Postgres --> Stats
    Backend --> Stats
    Frontend --> Backend
    Proxy --> Frontend
    Proxy --> Backend
    Proxy --> Stats
    GethRPC["Geth PoS Cluster"] --> Backend

    style GethRPC fill:#f96,stroke:#333
    style Postgres fill:#69b,stroke:#333
```

Blockscout requires:

- **Geth PoS cluster** running and producing blocks
- **PostgreSQL** for persistent storage
- **All 5 pods** healthy before the UI is fully functional
