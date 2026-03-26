# Blockscout Explorer — Operations

> See [architecture.md](architecture.md) for component diagrams and data flow.

## Full Redeployment (Clean Slate)

Tear down Blockscout, wipe all data, and redeploy.

```bash
# 1. Teardown — delete all workloads and storage
make delete-blockscout && \
make cleanup-blockscout-pvc && \
make cleanup-blockscout-pv && \
make cleanup-blockscout-data

# 2. Deploy — bring up all components
make deploy-blockscout
```

### Step Breakdown

| Step | Target                    | What It Does                                                                |
| ---- | ------------------------- | --------------------------------------------------------------------------- |
| 1a   | `delete-blockscout`       | Deletes proxy, frontend, backend, stats, and postgres workloads             |
| 1b   | `cleanup-blockscout-pvc`  | Deletes PVCs labeled `app=blockscout-postgres`                              |
| 1c   | `cleanup-blockscout-pv`   | Deletes PV `blockscout-postgres-pv-0`                                       |
| 1d   | `cleanup-blockscout-data` | SSH into Minikube node and `rm -rf /data/blockscout/postgres/data0`         |
| 2    | `deploy-blockscout`       | Applies postgres → stats → backend → frontend → proxy (in dependency order) |

> [!IMPORTANT]
> Blockscout depends on the Geth PoS cluster. If the chain was redeployed (new genesis), Blockscout data must also be wiped and redeployed to re-index from block 0.

## Accessing the Explorer

Only **one port-forward** is needed — all traffic goes through the nginx proxy:

```bash
make port-forward-blockscout    # localhost:3001 → proxy:80
```

Open **http://localhost:3001** in your browser.

### Additional Port Forwards (debugging only)

| Target                             | Local → Remote | Use Case                       |
| ---------------------------------- | -------------- | ------------------------------ |
| `port-forward-blockscout-backend`  | `4001 → 4000`  | Direct backend API access      |
| `port-forward-blockscout-postgres` | `5433 → 5432`  | Database inspection via `psql` |
| `port-forward-blockscout-stats`    | `8051 → 8050`  | Direct stats API access        |

## Kubernetes Manifests

```
deployments/kubernetes/minikube/blockscout/
├── postgres.yaml    # StatefulSet + PVC + Service (port 5432)
├── blockscout.yaml  # Backend Deployment + Service (port 4000)
├── frontend.yaml    # Frontend Deployment + Service (port 3000)
├── stats.yaml       # Stats Deployment + Service (port 8050)
└── proxy.yaml       # Nginx ConfigMap + Deployment + Service (port 80)
```

## Database Wipe (Without Full Redeploy)

When the Geth chain is reset but Blockscout workloads are still healthy, wipe only the DB to force a clean re-index:

```bash
# 1. Scale down consumers
kubectl --context web3-lab -n web3 scale deploy/blockscout-backend deploy/blockscout-stats --replicas=0
sleep 5

# 2. Terminate connections and drop/recreate DB
kubectl --context web3-lab -n web3 exec -it blockscout-postgres-0 -- \
  psql -U blockscout -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='blockscout' AND pid <> pg_backend_pid();"
kubectl --context web3-lab -n web3 exec -it blockscout-postgres-0 -- \
  psql -U blockscout -d postgres -c "DROP DATABASE IF EXISTS blockscout;"
kubectl --context web3-lab -n web3 exec -it blockscout-postgres-0 -- \
  psql -U blockscout -d postgres -c "CREATE DATABASE blockscout OWNER blockscout;"

# 3. Scale back up
kubectl --context web3-lab -n web3 scale deploy/blockscout-backend deploy/blockscout-stats --replicas=1
```

> [!IMPORTANT]
> You MUST connect to the `postgres` database (not `blockscout`) when running `DROP DATABASE blockscout`, otherwise Postgres will reject the drop with "cannot drop the currently open database".

## Restart Blockscout (No Data Loss)

Rollout restart all Blockscout services without wiping data:

```bash
make restart-blockscout    # restarts backend, stats, frontend
```

## Key Environment Variables

### Backend (`blockscout.yaml`)

| Variable                                        | Value                                                                    |
| ----------------------------------------------- | ------------------------------------------------------------------------ |
| `DATABASE_URL`                                  | `postgresql://blockscout:blockscout@blockscout-postgres:5432/blockscout` |
| `ETHEREUM_JSONRPC_HTTP_URL`                     | `http://geth-rpc:8545`                                                   |
| `ETHEREUM_JSONRPC_WS_URL`                       | `ws://geth-rpc:8546`                                                     |
| `ETHEREUM_JSONRPC_TRACE_URL`                    | `http://geth-rpc:8545`                                                   |
| `ETHEREUM_JSONRPC_VARIANT`                      | `geth`                                                                   |
| `CHAIN_ID`                                      | `72390`                                                                  |
| `INDEXER_INTERNAL_TRANSACTIONS_TRACER_TYPE`     | `call_tracer`                                                            |
| `INDEXER_DISABLE_EMPTY_BLOCKS_SANITIZER`        | `true`                                                                   |
| `INDEXER_DISABLE_INTERNAL_TRANSACTIONS_FETCHER` | `false`                                                                  |

### Frontend (`frontend.yaml`)

| Variable                     | Value                   | Notes                               |
| ---------------------------- | ----------------------- | ----------------------------------- |
| `NEXT_PUBLIC_API_HOST`       | `localhost`             | Browser-facing API host             |
| `NEXT_PUBLIC_API_PORT`       | `3001`                  | Must match proxy port-forward       |
| `NEXT_PUBLIC_STATS_API_HOST` | `http://localhost:3001` | Stats routed through proxy          |
| `NEXT_PUBLIC_APP_HOST`       | `0.0.0.0`               | Bind to all interfaces in container |
| `HOSTNAME`                   | `0.0.0.0`               | Required for Next.js server binding |

### Stats (`stats.yaml`)

| Variable                                    | Value                            |
| ------------------------------------------- | -------------------------------- |
| `STATS__BLOCKSCOUT_API_URL`                 | `http://blockscout-backend:4000` |
| `STATS__SERVER__HTTP__CORS__ENABLED`        | `true`                           |
| `STATS__SERVER__HTTP__CORS__ALLOWED_ORIGIN` | `http://localhost:3001`          |

## Troubleshooting

### "Something went wrong" error banner

Usually caused by a frontend/backend version mismatch. Verify compatible versions:

```bash
# Check backend version
kubectl --context web3-lab -n web3 exec deploy/blockscout-backend -- \
  curl -s http://localhost:4000/api/v2/config/backend-version
```

### CORS errors in browser console

All requests should go through the nginx proxy (`localhost:3001`). If CORS errors appear, verify:

1. Port-forward is using `svc/blockscout-proxy`, not `svc/blockscout-frontend`
2. `NEXT_PUBLIC_API_PORT` matches the proxy port-forward port (3001)

### No blocks showing / "scanning new blocks..."

Blockscout depends on the Geth PoS chain producing blocks. Check Geth health:

```bash
make check-pos-status
```

### Pod CrashLoopBackOff

Check logs for the failing pod:

```bash
kubectl --context web3-lab -n web3 logs deploy/blockscout-backend --tail=20
kubectl --context web3-lab -n web3 logs deploy/blockscout-frontend --tail=20
kubectl --context web3-lab -n web3 logs deploy/blockscout-stats --tail=20
```

### Internal transactions not showing

If internal transaction lists are empty for ERC-4337 transactions:

1. Verify `ETHEREUM_JSONRPC_TRACE_URL` is set in `blockscout.yaml`
2. Check logs for `REFUSED JOIN blocks:indexing_internal_transactions` — indicates the internal tx channel is blocked
3. Check indexing progress: `curl -s http://localhost:3001/api/v2/main-page/indexing-status | python3 -m json.tool`
4. If `indexed_internal_transactions_ratio` is `null`, the trace URL is misconfigured
5. If logs show `header not found` errors, the DB has stale data from a previous chain — perform a DB wipe

### EmptyBlocksSanitizer crash loop

If logs contain repeated `GenServer Indexer.Fetcher.EmptyBlocksSanitizer terminating` errors, set `INDEXER_DISABLE_EMPTY_BLOCKS_SANITIZER=true`. This is a known v8.1.1 bug.

## Quick Reference

| Operation              | Command                                                                                                                                         |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| **Deploy**             | `make deploy-blockscout`                                                                                                                        |
| **Full redeploy**      | `make delete-blockscout && make cleanup-blockscout-pvc && make cleanup-blockscout-pv && make cleanup-blockscout-data && make deploy-blockscout` |
| **Restart**            | `make restart-blockscout`                                                                                                                       |
| **Access UI**          | `make port-forward-blockscout` → http://localhost:3001                                                                                          |
| **Delete (keep data)** | `make delete-blockscout`                                                                                                                        |
| **Wipe DB only**       | See "Database Wipe" section above                                                                                                               |
| **Wipe all data**      | `make cleanup-blockscout-pvc && make cleanup-blockscout-pv && make cleanup-blockscout-data`                                                     |
| **Seed tokens**        | `make seed-upload && make test-interact && make seed-update-icons`                                                                              |

## Seed Data & Token Icons

Blockscout does not automatically display images for all token types. The following workflow populates token logos and NFT artwork.

### Prerequisites

- MinIO port-forward active: `make port-forward-minio`
- Geth RPC port-forward active: `make port-forward-geth-rpc`
- Contracts deployed: `make deploy-contracts`

### Workflow

```bash
# 1. Upload images + metadata to MinIO (with correct Content-Type)
make seed-upload

# 2. Deploy test tokens and interact (saves addresses to seed-addresses.json)
make test-interact

# 3. Update Blockscout DB with ERC-20 icons and fix blacklisted ERC-721 metadata
make seed-update-icons
```

### What `seed-update-icons` Does

| Token Type   | Problem                              | Fix                                                      |
| ------------ | ------------------------------------ | -------------------------------------------------------- |
| **ERC-20**   | No standard `tokenURI` for logos     | Sets `tokens.icon_url` in Postgres                       |
| **ERC-721**  | Metadata URLs blacklisted by fetcher | Clears `token_instances.error`, inserts metadata JSON    |
| **ERC-1155** | Uses `{id}` substitution in `uri()`  | Fetcher handles this natively if Content-Type is correct |

### Seed Files

```
seed/
├── images/
│   ├── erc20/       # 1.png .. 4.png (token logos)
│   ├── erc721/      # 0.png .. 3.png (NFT artwork)
│   └── erc1155/     # 1.png .. 4.png (item artwork)
├── metadata/
│   ├── erc721/      # 0, 1, 2, 3 (JSON, no extension)
│   └── erc1155/     # 1.json .. 4.json
├── upload.sh        # MinIO upload with Content-Type fix
└── update-blockscout-icons.sh  # Postgres DB update
```

> [!WARNING]
> Metadata image URLs MUST use `http://localhost:9000/...` (via port-forward), NOT `minio.web3.svc.cluster.local`. Blockscout's frontend renders images in the user's browser, which cannot resolve k8s internal DNS.
