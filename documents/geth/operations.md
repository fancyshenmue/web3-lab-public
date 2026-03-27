# Geth PoS Cluster — Operations

## Full Redeployment (Clean Slate)

Tear down the entire PoS cluster, wipe all storage, and redeploy from scratch.

```bash
# 1. Teardown — delete all workloads, storage resources, and host data
make delete-pos && \
make cleanup-pos-pvc && \
make cleanup-pos-pv && \
make cleanup-pos-data

# 2. Deploy — recreate PVs and bring up all layers
make apply-pv && \
make deploy-pos

# 3. Post-deploy — configure beacon P2P peering
make setup-beacon-peers
```

### Step Breakdown

| Step | Target               | What It Does                                                                                                                       |
| ---- | -------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| 1a   | `delete-pos`         | Deletes Validator, Beacon, and Geth StatefulSets + Services                                                                        |
| 1b   | `cleanup-pos-pvc`    | Deletes all PVCs (`app=geth`, `app=beacon`, `app=validator`)                                                                       |
| 1c   | `cleanup-pos-pv`     | Deletes all 9 PVs (`geth-pv-{0,1,2}`, `beacon-pv-{0,1,2}`, `validator-pv-{0,1,2}`)                                                 |
| 1d   | `cleanup-pos-data`   | SSH into all 3 Minikube nodes and `rm -rf` host path data (`/data/geth`, `/data/beacon`, `/data/validator`)                        |
| 2a   | `apply-pv`           | Re-creates `hostPath` Persistent Volumes                                                                                           |
| 2b   | `deploy-pos`         | Deploys Geth → Beacon (Validator deployment is intentionally delayed)                                                              |
| 3    | `setup-beacon-peers` | Queries peer IDs, creates `beacon-peers` ConfigMap, rolling-restarts beacon, waits for rollout to complete, then deploys Validator |

> [!IMPORTANT]
> `setup-beacon-peers` requires all 3 beacon pods to be **Running** before it can query their peer IDs. Wait for pods to be ready before running this step.

## Graceful Stop / Start

Stop and start the PoS stack without corrupting data. **Always use this before `minikube stop`.**

```bash
# Stop (before minikube stop)
make stop-pos-graceful
minikube stop -p web3-lab

# Start (after minikube start)
minikube start -p web3-lab
make start-pos-graceful
```

`start-pos-graceful` automatically checks finality after startup. If the chain has been stopped too long (>10 epoch gap ≈ 21 minutes), it warns you to run `make reset-pos-chain`.

## Node Repair

Fix a single stuck node without affecting the rest of the cluster:

```bash
make repair-pos-node NODE=0   # repairs geth-0 + beacon-0
```

This clears the node's geth and beacon data on the host, then pods auto-restart and re-sync from peers via the StatefulSet controller.

## Chain Reset

One-command full chain reset when finality is broken (e.g., stopped too long):

````bash
make reset-pos-chain
```

This runs: `stop-pos-graceful` → `cleanup-pos-data` → `deploy-pos` → `setup-beacon-peers`. All chain data (transactions, contracts) is lost but PV/PVC resources are preserved.

> [!CAUTION]
> `reset-pos-chain` destroys all on-chain state. Smart contracts must be redeployed.

## Shutdown Protection

| Component   | `terminationGracePeriodSeconds` | `preStop` Hook          |
| ----------- | ------------------------------- | ----------------------- |
| **Geth**    | 120s                            | `kill -INT 1; sleep 15` |
| **Beacon**  | 60s                             | —                       |

Geth uses PathDB (`--state.scheme=path`) which requires sufficient shutdown time to flush state to disk. The `preStop` hook sends SIGINT before Kubernetes escalates to SIGKILL.

## Health Checks

```bash
# Verify host path directories exist on all Minikube nodes
make check-host-paths

# Deep status check: pod status, geth peers, beacon sync, validator readiness
make check-pos-status
````

### `check-host-paths`

Lists `/data/*` directories on all 3 Minikube nodes. Useful to confirm that host path data was properly created (after deploy) or removed (after cleanup).

### `check-pos-status`

Runs a comprehensive health check across all layers:

| Section                | What It Checks                                                                                         |
| ---------------------- | ------------------------------------------------------------------------------------------------------ |
| **Pod Status**         | All geth/beacon/validator pods via `kubectl get pods -o wide`                                          |
| **Geth Peer Count**    | Attaches to each geth IPC and reads `admin.peers.length` (expect 2 peers each)                         |
| **Beacon Sync Status** | Port-forwards to each beacon's HTTP API and queries `/eth/v1/node/syncing` + `/eth/v1/node/peer_count` |
| **Validator Status**   | Reads pod phase, ready state, and restart count for each validator                                     |

## Quick Reference

| Operation              | Command                                                                                                                                                  |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Graceful stop**      | `make stop-pos-graceful`                                                                                                                                 |
| **Graceful start**     | `make start-pos-graceful`                                                                                                                                |
| **Repair single node** | `make repair-pos-node NODE=0`                                                                                                                            |
| **Reset chain**        | `make reset-pos-chain`                                                                                                                                   |
| **Full redeploy**      | `make delete-pos && make cleanup-pos-pvc && make cleanup-pos-pv && make cleanup-pos-data && make apply-pv && make deploy-pos && make setup-beacon-peers` |
| **Check storage**      | `make check-host-paths`                                                                                                                                  |
| **Cluster health**     | `make check-pos-status`                                                                                                                                  |
| **Check block height** | `make check-latest-block`                                                                                                                                |
| **Verify pods**        | `make verify-nodes`                                                                                                                                      |
| **Geth-only redeploy** | `make delete-geth && make cleanup-geth-data && make deploy-geth`                                                                                         |
