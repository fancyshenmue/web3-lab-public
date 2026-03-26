# Geth PoS Cluster — Testing Guide

Assumes port-forwarding is active:

```bash
make port-forward-geth-rpc    # localhost:8545
make port-forward-geth-ws     # localhost:8546
make port-forward-beacon-api  # localhost:3500
```

## Geth JSON-RPC (HTTP)

### Check Node Status

```bash
# Chain ID
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' | python -m json.tool

# Expected: "result": "0x11aa6" (72390)

# Latest block number
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | python -m json.tool

# Syncing status
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' | python -m json.tool

# Expected: "result": false (not syncing = in sync)
```

### Check Peers

```bash
# Peer count
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' | python -m json.tool

# Expected: "result": "0x2" (2 peers)

# Peer details
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"admin_peers","params":[],"id":1}' | python -m json.tool

# Node info (enode URL)
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":1}' | python -m json.tool
```

### Query Accounts

```bash
# Check pre-funded account balance
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x123463a4b065722e99115d6c222f267d9cabb524","latest"],"id":1}' | python -m json.tool

# Expected: "result": "0x3635c9adc5dea00000" (1000 ETH in wei)

# List all pre-funded accounts (check any of these):
# 0x123463a4b065722e99115d6c222f267d9cabb524
# 0x14dc79964da2c08dba798bbb5e846ceff39bfebc
# 0x23618e81e3f5cdf7f54c3d65f7fbc0abf5b21e8f
# 0xa0ee7a142d267c1f36714e4a8f75612f20a79720
# 0xbcd4042de499d14e55001ccbb24a551f3b954096
# 0x71be63f3384f5fb98995898a86b02fb2426c5788
# 0xfabb0ac9d68b0b445fb7357272ff202c5651694a
# 0x1cbd3b2770909d4e10f157cabc84c7264073c9ec
# 0xdf3e18d64bc6a983f673ab319ccae4f1a57c7097
# 0xcd3b766ccdd6ae721141f452c550ca635964ce71
```

### Get Block Details

```bash
# Latest block with full transactions
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest",true],"id":1}' | python -m json.tool

# Genesis block
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0",false],"id":1}' | python -m json.tool
```

### Transaction Pool

```bash
# Pending transactions
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"txpool_status","params":[],"id":1}' | python -m json.tool
```

### Gas Price

```bash
curl -s localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":1}' | python -m json.tool
```

## Beacon API (HTTP REST)

### Node Status

```bash
# Node version
curl -s localhost:3500/eth/v1/node/version | python -m json.tool

# Node identity (peer ID, ENR, multiaddrs)
curl -s localhost:3500/eth/v1/node/identity | python -m json.tool

# Health check
curl -s -o /dev/null -w "%{http_code}" localhost:3500/eth/v1/node/health
# Expected: 200

# Syncing status
curl -s localhost:3500/eth/v1/node/syncing | python -m json.tool
```

### Peer Info

```bash
# Peer count
curl -s localhost:3500/eth/v1/node/peer_count | python -m json.tool

# List connected peers
curl -s localhost:3500/eth/v1/node/peers | python -m json.tool
```

### Chain Head

```bash
# Head block root + slot
curl -s localhost:3500/eth/v1/beacon/headers/head | python -m json.tool

# Finalized checkpoint
curl -s localhost:3500/eth/v1/beacon/states/head/finality_checkpoints | python -m json.tool
```

### Validators

```bash
# Active validators (first page)
curl -s "localhost:3500/eth/v1/beacon/states/head/validators?status=active" | python -m json.tool | head -50

# Specific validator by index
curl -s localhost:3500/eth/v1/beacon/states/head/validators/0 | python -m json.tool
```

## Node.js (ethers.js)

### Setup

```bash
npm init -y && npm install ethers
```

### Connect and Query

```js
// test-geth.mjs
import { ethers } from "ethers";

const provider = new ethers.JsonRpcProvider("http://localhost:8545");

async function main() {
  // Network info
  const network = await provider.getNetwork();
  console.log("Chain ID:", network.chainId.toString());

  // Latest block
  const block = await provider.getBlock("latest");
  console.log("Block:", block.number, "| Hash:", block.hash);
  console.log("Timestamp:", new Date(block.timestamp * 1000).toISOString());

  // Account balance
  const addr = "0x123463a4b065722e99115d6c222f267d9cabb524";
  const balance = await provider.getBalance(addr);
  console.log("Balance:", ethers.formatEther(balance), "ETH");

  // Gas price
  const feeData = await provider.getFeeData();
  console.log(
    "Gas Price:",
    ethers.formatUnits(feeData.gasPrice, "gwei"),
    "gwei",
  );

  // Peer count
  const peers = await provider.send("net_peerCount", []);
  console.log("Peers:", parseInt(peers, 16));
}

main().catch(console.error);
```

```bash
node test-geth.mjs
```

### Send Transaction

```js
// send-tx.mjs
import { ethers } from "ethers";

const provider = new ethers.JsonRpcProvider("http://localhost:8545");

// WARNING: This is a test private key from Hardhat/Foundry defaults
// DO NOT use on mainnet
const PRIVATE_KEY =
  "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";

async function main() {
  const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
  console.log("Sender:", wallet.address);

  const balance = await provider.getBalance(wallet.address);
  console.log("Balance:", ethers.formatEther(balance), "ETH");

  // Send 1 ETH to another pre-funded account
  const tx = await wallet.sendTransaction({
    to: "0x14dc79964da2c08dba798bbb5e846ceff39bfebc",
    value: ethers.parseEther("1.0"),
  });

  console.log("TX Hash:", tx.hash);
  const receipt = await tx.wait();
  console.log("Confirmed in block:", receipt.blockNumber);
  console.log("Gas used:", receipt.gasUsed.toString());
}

main().catch(console.error);
```

```bash
node send-tx.mjs
```

### WebSocket Subscription

```js
// subscribe-blocks.mjs
import { ethers } from "ethers";

const wsProvider = new ethers.WebSocketProvider("ws://localhost:8546");

wsProvider.on("block", async (blockNumber) => {
  const block = await wsProvider.getBlock(blockNumber);
  console.log(
    `Block #${blockNumber} | txs: ${block.transactions.length} | ` +
      `gas: ${block.gasUsed.toString()} | time: ${new Date(block.timestamp * 1000).toLocaleTimeString()}`,
  );
});

console.log("Listening for new blocks... (Ctrl+C to stop)");
```

```bash
# Requires: make port-forward-geth-ws
node subscribe-blocks.mjs
```

## Quick Smoke Test

One-liner to verify the entire stack:

```bash
# Check chain is alive and producing blocks
curl -sf localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | \
  python -c "import sys,json; bn=int(json.load(sys.stdin)['result'],16); print(f'✅ Chain alive — block #{bn}') if bn > 0 else print('⚠️  Block 0 — chain may not be producing blocks yet')"
```
