# OpenSpec: Blockscout Explorer

## Status

In Progress

## Context

Blockscout is an open-source blockchain explorer for EVM-compatible chains. The modern Blockscout v6+ architecture consists of three separate services:

- **Backend** — Elixir/Phoenix API server that indexes the blockchain and serves the REST/GraphQL API
- **Frontend** — Next.js web application providing the explorer UI
- **Stats** — Rust microservice that computes and serves chart/statistics data

All three connect to a shared PostgreSQL database. The backend connects to the Geth EL node for block ingestion.

## Architecture

```
                        ┌──────────────┐
                        │   Frontend   │
                        │  (Next.js)   │
                        │  port 3000   │
                        └──────┬───────┘
                               │ API calls
                        ┌──────┴───────┐
                        │   Backend    │ ←── RPC/WS ──→ Geth (EL)
                        │  (Elixir)    │                port 8545
                        │  port 4000   │
                        └──────┬───────┘
                               │ SQL
                        ┌──────┴───────┐
          ┌────────────→│  PostgreSQL   │←────────────┐
          │ SQL          │   port 5432   │  SQL         │
   ┌──────┴───────┐     └──────────────┘      ┌───────┴──────┐
   │    Stats     │                            │   (Backend)  │
   │   (Rust)     │                            └──────────────┘
   │  port 8050   │
   └──────────────┘
```

| Component               | Image                                | Ports     | PV  |
| ----------------------- | ------------------------------------ | --------- | --- |
| **Backend**             | `blockscout/blockscout:latest`       | HTTP 4000 | —   |
| **Frontend**            | `ghcr.io/blockscout/frontend:latest` | HTTP 3000 | —   |
| **Stats**               | `ghcr.io/blockscout/stats:latest`    | HTTP 8050 | —   |
| **Blockscout Postgres** | `postgres:16-alpine`                 | 5432      | 5Gi |

## Requirements

### Requirement: Blockchain Indexing (Backend)

The backend SHALL connect to the Geth EL cluster via JSON-RPC (`http://geth-rpc:8545`) and WebSocket (`ws://geth-rpc:8546`) to index blocks, transactions, and smart contracts.

- The backend SHALL run DB migrations on startup.
- The backend SHALL serve REST API on port 4000.

### Requirement: Explorer UI (Frontend)

The frontend SHALL connect to the backend API (`http://blockscout-backend:4000`) and provide a web-based blockchain explorer UI on port 3000, accessible via `kubectl port-forward`.

### Requirement: Statistics Service (Stats)

The stats service SHALL connect to the same PostgreSQL database and serve chart/statistics data on port 8050.

- The backend SHALL be configured to proxy stats API requests to the stats service.

### Requirement: PostgreSQL Backend

Blockscout SHALL use a dedicated PostgreSQL instance for storing indexed blockchain data.

- PostgreSQL SHALL run as a StatefulSet with a 5Gi PersistentVolume.
- The database SHALL be pre-created on startup via `POSTGRES_DB` env var.

### Requirement: Environment Configuration

Each service SHALL be configured via environment variables:

- **Backend**: `DATABASE_URL`, `ETHEREUM_JSONRPC_HTTP_URL`, `ETHEREUM_JSONRPC_WS_URL`, `ETHEREUM_JSONRPC_TRACE_URL`, `ETHEREUM_JSONRPC_VARIANT=geth`, `CHAIN_ID=72390`
- **Frontend**: `NEXT_PUBLIC_APP_HOST`, `NEXT_PUBLIC_API_HOST`, `NEXT_PUBLIC_API_PORT`, `NEXT_PUBLIC_API_PROTOCOL`, `NEXT_PUBLIC_STATS_API_HOST`, `NEXT_PUBLIC_NETWORK_NAME`
- **Stats**: `STATS__DB_URL`, `STATS__BLOCKSCOUT_DB_URL`, `STATS__BLOCKSCOUT_API_URL`, `STATS__SERVER__HTTP__ADDR`

### Requirement: Internal Transaction Indexing

The backend SHALL index internal transactions (CALL, CREATE, DELEGATECALL, STATICCALL) via Geth's `debug_traceTransaction` / `debug_traceBlockByNumber` APIs.

The following environment variables MUST be set:

| Variable                                        | Value                  | Purpose                                                   |
| ----------------------------------------------- | ---------------------- | --------------------------------------------------------- |
| `ETHEREUM_JSONRPC_TRACE_URL`                    | `http://geth-rpc:8545` | Dedicated endpoint for trace API calls                    |
| `INDEXER_INTERNAL_TRANSACTIONS_TRACER_TYPE`     | `call_tracer`          | Explicit Geth callTracer (default since Blockscout 5.1.0) |
| `INDEXER_DISABLE_INTERNAL_TRANSACTIONS_FETCHER` | `false`                | Must NOT be disabled                                      |
| `INDEXER_DISABLE_EMPTY_BLOCKS_SANITIZER`        | `true`                 | Workaround for v8.1.1 bug (see Known Issues)              |

> [!IMPORTANT]
> Without `ETHEREUM_JSONRPC_TRACE_URL`, the internal transaction fetcher silently fails, resulting in empty internal transaction lists and preventing token discovery for contracts deployed via internal `CREATE` operations (e.g., ERC-4337 factory deployments).

> [!WARNING]
> The Geth EL nodes MUST have the `debug` API enabled (`--http.api=eth,net,web3,txpool,debug,admin`). The current PoS cluster configuration already includes this.

### Requirement: ERC-4337 Account Abstraction Interplay

All user transactions in web3-lab are routed through the ERC-4337 **EntryPoint** contract (`0xf5059a5D33d5853360D16C683c16e67980206f36`). This means:

1. **Every transaction's `to` field** points to the EntryPoint — Blockscout shows this as the interacted contract
2. **Smart Wallet calls** (execute, mint, transfer) are nested as internal transactions
3. **Factory-deployed contracts** (ERC20/721/1155) are created via internal `CREATE` opcodes, not top-level transactions
4. **Token classification** depends on the internal transaction indexer completing — tokens created via factories only appear in the token list once their internal `CREATE` is indexed

Internal transaction execution order in Blockscout's "Internal txns" tab is displayed **top-to-bottom** (first executed at top).

Typical ERC-4337 UserOp call trace:

```
EOA (Bundler) → EntryPoint.handleOps()
  ├─ CALL: EntryPoint → SmartWallet (validateUserOp)
  ├─ CALL: EntryPoint → Paymaster (validatePaymasterUserOp)
  ├─ CALL: EntryPoint → SmartWallet.execute(target, value, data)
  │   └─ DELEGATECALL: SmartWallet → Implementation
  │       └─ CALL: SmartWallet → TokenFactory.createToken()  (or mint/transfer)
  │           └─ CREATE: Factory → New Token Contract
  └─ CALL: EntryPoint → Bundler (gas refund)
```

### Requirement: NFT Metadata & Image Display

The backend's **token instance fetcher** SHALL call each ERC-721/1155 contract's `tokenURI()` / `uri()`, fetch the returned JSON metadata, and store the result in the `token_instances.metadata` column.

> [!IMPORTANT]
>
> - Metadata endpoints MUST return `Content-Type: application/json`. If they return `application/octet-stream`, the fetcher **blacklists** the URL permanently (`token_instances.error = 'blacklist'`).
> - Image URLs in metadata MUST be accessible from the **user's browser** (e.g., `http://localhost:9000/...`), NOT internal k8s DNS. Blockscout passes the `image` field directly to the frontend.
> - ERC-20 tokens do not have a standard `tokenURI`. ERC-20 logos are set via the `tokens.icon_url` column directly.

### Requirement: Seed Data Integration

After deploying tokens via `test-interact.js`, the `update-blockscout-icons.sh` script SHALL:

1. Set `tokens.icon_url` for ERC-20 tokens (no standard on-chain mechanism)
2. Clear `token_instances.error = 'blacklist'` and re-insert correct metadata for ERC-721 NFTs

## Known Issues

### EmptyBlocksSanitizer crash (v8.1.1)

The `Indexer.Fetcher.EmptyBlocksSanitizer` GenServer crashes with `Protocol.UndefinedError` in Blockscout v8.1.1 when connected to Geth (Enumerable not implemented for nil). This creates continuous error logs and can disrupt the indexing pipeline.

**Workaround**: Set `INDEXER_DISABLE_EMPTY_BLOCKS_SANITIZER=true`.

### Stale DB after chain reset

If the Geth PoS cluster is reset (new genesis), the Blockscout database retains references to blocks from the previous chain. This causes:

- `coin_balance_catchup` fetcher errors: `(-32000) header not found`
- `REFUSED JOIN blocks:indexing_internal_transactions` messages
- Internal transactions not being re-fetched for already-indexed blocks

**Fix**: Full DB wipe is required after any chain reset (see [operations.md](../../../documents/blockscout/operations.md)).
