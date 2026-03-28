.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Versioning
VERSION ?= $(shell cat VERSION)
DATE := $(shell date +%Y%m%d)
TAG := $(DATE)_v$(subst .,_,$(VERSION))

.PHONY: bump-major bump-minor bump-patch

bump-major: ## Bump major version (X.y.z -> X+1.0.0)
	@scripts/bump_version.sh major

bump-minor: ## Bump minor version (x.Y.z -> x.Y+1.0)
	@scripts/bump_version.sh minor

bump-patch: ## Bump patch version (x.y.Z -> x.y.Z+1)
	@scripts/bump_version.sh patch

# Docker Image Names
MOCK_PRICE_IMAGE ?= mock-price:$(TAG)

# Minikube Settings
MINIKUBE_PROFILE ?= web3-lab
NAMESPACE ?= web3

.PHONY: minikube-start minikube-stop minikube-delete minikube-status

minikube-start: ## Start Minikube (profile: MINIKUBE_PROFILE)
	minikube start --nodes 3 -p $(MINIKUBE_PROFILE) --driver=docker --memory 40960 --cpus 8

minikube-stop: ## Stop Minikube (profile: MINIKUBE_PROFILE)
	minikube stop -p $(MINIKUBE_PROFILE)

minikube-delete: ## Delete Minikube (profile: MINIKUBE_PROFILE)
	minikube delete -p $(MINIKUBE_PROFILE)

minikube-status: ## Show Minikube status (profile: MINIKUBE_PROFILE)
	minikube status -p $(MINIKUBE_PROFILE)

minikube-tunnel: ## Start Minikube tunnel to enable 127.0.0.1 Ingress routing
	sudo minikube tunnel -p $(MINIKUBE_PROFILE)

minikube-tunnel-stop: ## Stop Minikube tunnel and clean up stale lock/SSH processes
	@sudo pkill -9 -f "ssh.*minikube.*-L 80:" 2>/dev/null || true
	@sudo pkill -f "minikube tunnel" 2>/dev/null || true
	@rm -f $(HOME)/.minikube/profiles/$(MINIKUBE_PROFILE)/.tunnel_lock
	@echo "✅ Tunnel stopped and lock file cleaned"

# Namespace & Storage

create-namespace: ## Create Kubernetes namespace
	kubectl --context $(MINIKUBE_PROFILE) get ns $(NAMESPACE) || kubectl --context $(MINIKUBE_PROFILE) create ns $(NAMESPACE)

apply-pv: ## Apply Persistent Volumes definitions
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/storage/pv.yaml

# Deploy — Geth PoS Cluster (all layers in one directory)

deploy-geth: setup-genesis-time create-namespace ## Deploy Geth EL to Minikube
	@TIME=$$(kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get configmap genesis-time -o jsonpath='{.data.time}'); \
	HEX_TIME=$$(printf '0x%x' $$TIME); \
	cat deployments/kubernetes/minikube/geth-pos-cluster/genesis-configmap.yaml | \
		sed "s/\"timestamp\": \"0x0\"/\"timestamp\": \"$$HEX_TIME\"/" | \
		sed "s/\"shanghaiTime\": 0/\"shanghaiTime\": $$TIME/" | \
		sed "s/\"cancunTime\": 0/\"cancunTime\": $$TIME/" | \
		kubectl --context $(MINIKUBE_PROFILE) apply -f -
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/geth-pos-cluster/jwt-secret.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/geth-pos-cluster/prysm-config.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/geth-pos-cluster/services.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/geth-pos-cluster/geth.yaml

delete-geth: ## Delete Geth EL from Minikube
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/geth-pos-cluster/geth.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/geth-pos-cluster/services.yaml --ignore-not-found=true

cleanup-geth-data: ## Wipe Geth chain data from all minikube nodes
	minikube ssh -p $(MINIKUBE_PROFILE) -- sudo rm -rf /data/geth/data0
	minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m02 -- sudo rm -rf /data/geth/data1
	minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m03 -- sudo rm -rf /data/geth/data2

cleanup-pos-data: ## Wipe all PoS data (geth + beacon + validator) from all minikube nodes
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete configmap genesis-time --ignore-not-found=true
	minikube ssh -p $(MINIKUBE_PROFILE) -- sudo rm -rf /data/geth/data0 /data/beacon/data0 /data/validator/data0
	minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m02 -- sudo rm -rf /data/geth/data1 /data/beacon/data1 /data/validator/data1
	minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m03 -- sudo rm -rf /data/geth/data2 /data/beacon/data2 /data/validator/data2

cleanup-pos-pvc: ## Delete all PoS PVCs (geth, beacon, validator)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete pvc -l app=geth --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete pvc -l app=beacon --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete pvc -l app=validator --ignore-not-found=true

cleanup-pos-pv: ## Delete all PoS PVs (geth, beacon, validator)
	kubectl --context $(MINIKUBE_PROFILE) delete pv geth-pv-0 geth-pv-1 geth-pv-2 --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) delete pv beacon-pv-0 beacon-pv-1 beacon-pv-2 --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) delete pv validator-pv-0 validator-pv-1 validator-pv-2 --ignore-not-found=true

check-host-paths: ## Check PV host path directories on all minikube nodes
	@echo "📂 Node: $(MINIKUBE_PROFILE) (master)"
	@minikube ssh -p $(MINIKUBE_PROFILE) -- 'ls -la /data/ 2>/dev/null && for d in /data/*/; do echo "  $$d:"; ls -la "$$d" 2>/dev/null; done' || echo "  (no /data directory)"
	@echo ""
	@echo "📂 Node: $(MINIKUBE_PROFILE)-m02"
	@minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m02 -- 'ls -la /data/ 2>/dev/null && for d in /data/*/; do echo "  $$d:"; ls -la "$$d" 2>/dev/null; done' || echo "  (no /data directory)"
	@echo ""
	@echo "📂 Node: $(MINIKUBE_PROFILE)-m03"
	@minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m03 -- 'ls -la /data/ 2>/dev/null && for d in /data/*/; do echo "  $$d:"; ls -la "$$d" 2>/dev/null; done' || echo "  (no /data directory)"

setup-genesis-time: create-namespace
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get configmap genesis-time >/dev/null 2>&1 || \
		(echo "📝 Creating deterministic genesis-time ConfigMap..." && \
		 kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) create configmap genesis-time --from-literal=time=$$(date +%s) --dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -)

deploy-beacon: setup-genesis-time create-namespace ## Deploy Beacon CL to Minikube
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/geth-pos-cluster/beacon-services.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/geth-pos-cluster/beacon.yaml

delete-beacon: ## Delete Beacon CL from Minikube
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/geth-pos-cluster/beacon.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/geth-pos-cluster/beacon-services.yaml --ignore-not-found=true

deploy-validator: create-namespace ## Deploy Validator to Minikube
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/geth-pos-cluster/validator.yaml

delete-validator: ## Delete Validator from Minikube
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/geth-pos-cluster/validator.yaml --ignore-not-found=true

# Deploy — Blockscout Explorer

deploy-blockscout: create-namespace ## Deploy Blockscout (postgres + backend + frontend + stats + proxy + ingress)
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/blockscout/postgres.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/blockscout/stats.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/blockscout/blockscout.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/blockscout/frontend.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/blockscout/proxy.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/blockscout/blockscout-ingress.yaml

delete-blockscout: ## Delete Blockscout from Minikube
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/blockscout/blockscout-ingress.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/blockscout/proxy.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/blockscout/frontend.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/blockscout/blockscout.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/blockscout/stats.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/blockscout/postgres.yaml --ignore-not-found=true

cleanup-blockscout-pvc: ## Delete all Blockscout PVCs (postgres)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete pvc -l app=blockscout-postgres --ignore-not-found=true

cleanup-blockscout-pv: ## Delete all Blockscout PVs (postgres)
	kubectl --context $(MINIKUBE_PROFILE) delete pv blockscout-postgres-pv-0 --ignore-not-found=true

cleanup-blockscout-data: ## Wipe Blockscout data (postgres) from minikube nodes
	minikube ssh -p $(MINIKUBE_PROFILE) -- sudo rm -rf /data/blockscout/postgres/data0

restart-blockscout: ## Rollout restart Blockscout (backend + stats + frontend)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/blockscout-backend
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/blockscout-stats
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/blockscout-frontend

# Deploy — MinIO Distributed Storage

deploy-minio: create-namespace ## Deploy MinIO (Secret, Service, StatefulSet, Ingress)
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/minio/minio-secret.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/minio/minio-service.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/minio/minio-statefulset.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/minio/minio-ingress.yaml

delete-minio: ## Delete MinIO from Minikube
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/minio/minio-ingress.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/minio/minio-statefulset.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/minio/minio-service.yaml --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete -f deployments/kubernetes/minikube/minio/minio-secret.yaml --ignore-not-found=true

cleanup-minio-pvc: ## Delete all MinIO PVCs
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete pvc -l app=minio --ignore-not-found=true

cleanup-minio-pv: ## Delete all MinIO PVs
	kubectl --context $(MINIKUBE_PROFILE) delete pv minio-pv-0 minio-pv-1 minio-pv-2 minio-pv-3 minio-pv-4 minio-pv-5 --ignore-not-found=true

cleanup-minio-data: ## Wipe MinIO physical disks from all nodes
	minikube ssh -p $(MINIKUBE_PROFILE) -- sudo rm -rf /data/minio/distributed/data0 /data/minio/distributed/data3
	minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m02 -- sudo rm -rf /data/minio/distributed/data1 /data/minio/distributed/data4
	minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m03 -- sudo rm -rf /data/minio/distributed/data2 /data/minio/distributed/data5

init-minio: ## Initialize MinIO bucket (requires MinIO running)
	@echo "🔗 Starting port-forward to MinIO..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/minio 9000:9000 &>/dev/null & \
	PF_PID=$$!; \
	sleep 2; \
	echo "📝 Setting up mc alias..."; \
	mc alias set web3lab http://localhost:9000 minioadmin minioadmin; \
	echo ""; \
	echo "🪣 Creating bucket..."; \
	mc mb --ignore-existing web3lab/web3lab-assets; \
	mc anonymous set download web3lab/web3lab-assets; \
	kill $${PF_PID} 2>/dev/null; wait $${PF_PID} 2>/dev/null; \
	echo ""; \
	echo "✅ MinIO bucket initialized! (prefixes: erc20/ erc721/ erc1155/ abis/)"

# Beacon P2P Peering

setup-beacon-peers: ## Setup beacon P2P peering (run after deploy-pos, requires beacons running)
	@echo "🔍 Querying beacon node identities..."
	@PEERS_FILE=$$(mktemp); \
	for i in 0 1 2; do \
		echo "  ↳ Querying beacon-$${i}..."; \
		LOCAL_PORT=$$((3500 + i + 10)); \
		kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward pod/beacon-$${i} $${LOCAL_PORT}:3500 &>/dev/null & \
		PF_PID=$$!; \
		sleep 2; \
		PEER_ID=$$(curl -sf http://localhost:$${LOCAL_PORT}/eth/v1/node/identity | \
			python3 -c "import sys,json; print(json.load(sys.stdin)['data']['peer_id'])" 2>/dev/null); \
		kill $${PF_PID} 2>/dev/null; wait $${PF_PID} 2>/dev/null; \
		if [ -z "$${PEER_ID}" ]; then \
			echo "    ❌ Failed to get peer ID for beacon-$${i}. Is it running?"; \
			rm -f $${PEERS_FILE}; \
			exit 1; \
		fi; \
		echo "    ✅ beacon-$${i}: $${PEER_ID}"; \
		echo "$${i}|/dns4/beacon-$${i}.beacon-headless.$(NAMESPACE).svc.cluster.local/tcp/13000/p2p/$${PEER_ID}" >> $${PEERS_FILE}; \
	done; \
	echo ""; \
	echo "📝 Creating beacon-peers ConfigMap..."; \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) create configmap beacon-peers \
		--from-file=peers.txt=$${PEERS_FILE} \
		--dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -; \
	rm -f $${PEERS_FILE}; \
	echo ""; \
	echo "🔄 Rolling restart beacon StatefulSet..."; \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart statefulset/beacon; \
	echo ""; \
	echo "✅ Beacon P2P peering configured! Beacons will peer on restart."; \
	echo "⏳ Waiting for beacon rolling restart to complete..."; \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout status statefulset/beacon --timeout=180s; \
	echo "✅ All beacons restarted and ready. Deploying validators..."; \
	$(MAKE) deploy-validator

deploy-pos: apply-pv deploy-geth deploy-beacon ## Deploy PoS cluster (Geth + Beacon)

delete-pos: delete-validator delete-beacon delete-geth ## Delete PoS cluster

stop-pos-graceful: ## Gracefully stop PoS stack (scale to 0 in dependency order, safe for minikube stop)
	@echo "🛑 Gracefully stopping PoS stack..."
	@echo "  Step 1/3: Stopping validators..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) scale sts validator --replicas=0 2>/dev/null || true
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout status sts/validator --timeout=60s 2>/dev/null || true
	@echo "  Step 2/3: Stopping beacon nodes..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) scale sts beacon --replicas=0 2>/dev/null || true
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout status sts/beacon --timeout=120s 2>/dev/null || true
	@echo "  Step 3/3: Stopping geth nodes..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) scale sts geth --replicas=0 2>/dev/null || true
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout status sts/geth --timeout=180s 2>/dev/null || true
	@echo "✅ All PoS pods stopped cleanly. Safe to run: make minikube-stop"

start-pos-graceful: ## Start PoS stack back (scale up in dependency order, auto-detects stale chain)
	@echo "🚀 Starting PoS stack..."
	@echo "  Step 1/3: Starting geth nodes..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) scale sts geth --replicas=3
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout status sts/geth --timeout=180s
	@echo "  Step 2/3: Starting beacon nodes..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) scale sts beacon --replicas=3
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout status sts/beacon --timeout=180s
	@echo "  Step 3/3: Starting validators..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) scale sts validator --replicas=3
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout status sts/validator --timeout=120s
	@echo "✅ All PoS pods started."
	@echo "🔍 Checking chain finality (waiting 30s for beacon sync)..."
	@sleep 30; \
	GENESIS_TIME=$$(kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get configmap genesis-time -o jsonpath='{.data.time}' 2>/dev/null); \
	if [ -z "$$GENESIS_TIME" ]; then \
		echo "  ⚠️  No genesis-time ConfigMap found, skipping finality check."; \
		exit 0; \
	fi; \
	NOW=$$(date +%s); \
	ELAPSED=$$((NOW - GENESIS_TIME)); \
	CURRENT_EPOCH=$$((ELAPSED / 128)); \
	LOCAL_PORT=3590; \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward pod/beacon-0 $${LOCAL_PORT}:3500 &>/dev/null & \
	PF_PID=$$!; \
	sleep 3; \
	HEAD_SLOT=$$(curl -sf http://localhost:$${LOCAL_PORT}/eth/v1/beacon/headers/head 2>/dev/null | \
		python3 -c "import sys,json; print(json.load(sys.stdin)['data']['header']['message']['slot'])" 2>/dev/null); \
	kill $${PF_PID} 2>/dev/null; wait $${PF_PID} 2>/dev/null; \
	if [ -z "$$HEAD_SLOT" ]; then \
		echo "  ⚠️  Could not query beacon head, skipping finality check."; \
		exit 0; \
	fi; \
	HEAD_EPOCH=$$((HEAD_SLOT / 32)); \
	GAP=$$((CURRENT_EPOCH - HEAD_EPOCH)); \
	echo "  📊 Current epoch: $$CURRENT_EPOCH, Head epoch: $$HEAD_EPOCH, Gap: $$GAP epochs"; \
	if [ $$GAP -gt 10 ]; then \
		echo ""; \
		echo "  ⚠️  Chain is stale ($$GAP epoch gap > 10 threshold)."; \
		echo "  ⚠️  Finality is likely broken. Run: make reset-pos-chain"; \
		echo ""; \
	else \
		echo "  ✅ Chain finality looks healthy. Monitor with: make check-pos-status"; \
	fi

repair-pos-node: ## Repair a stuck PoS node (usage: make repair-pos-node NODE=0)
	@if [ -z "$(NODE)" ]; then echo "❌ Usage: make repair-pos-node NODE=<0|1|2>"; exit 1; fi
	@echo "🔧 Repairing PoS node $(NODE)..."
	@case $(NODE) in \
		0) MINIKUBE_NODE="$(MINIKUBE_PROFILE)" ;; \
		1) MINIKUBE_NODE="$(MINIKUBE_PROFILE)-m02" ;; \
		2) MINIKUBE_NODE="$(MINIKUBE_PROFILE)-m03" ;; \
		*) echo "❌ NODE must be 0, 1, or 2"; exit 1 ;; \
	esac; \
	echo "  Step 1/4: Scaling down geth-$(NODE) and beacon-$(NODE)..."; \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete pod geth-$(NODE) beacon-$(NODE) --grace-period=5 --force 2>/dev/null || true; \
	sleep 5; \
	echo "  Step 2/4: Clearing geth-$(NODE) chain data on host $$MINIKUBE_NODE..."; \
	minikube ssh -p $(MINIKUBE_PROFILE) -n $$MINIKUBE_NODE -- "sudo rm -rf /data/geth/data$(NODE)/geth && echo '  ✅ geth data cleared'"; \
	echo "  Step 3/4: Clearing beacon-$(NODE) database on host $$MINIKUBE_NODE..."; \
	minikube ssh -p $(MINIKUBE_PROFILE) -n $$MINIKUBE_NODE -- "sudo rm -rf /data/beacon/data$(NODE)/beaconchaindata /data/beacon/data$(NODE)/metaData && echo '  ✅ beacon DB cleared'"; \
	echo "  Step 4/4: Pods will auto-restart via StatefulSet controller..."; \
	sleep 3; \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pod geth-$(NODE) beacon-$(NODE) -o wide 2>/dev/null || true; \
	echo "✅ Repair complete. Both pods will re-init from genesis and sync from peers."; \
	echo "   Monitor with: make check-pos-status"

reset-pos-chain: ## Reset PoS chain with new genesis time (keeps PV/PVC, clears all chain data)
	@echo "🔄 Resetting PoS chain with new genesis..."
	@echo "  ⚠️  This will clear ALL chain data (transactions, contracts, etc.)"
	@echo ""
	@echo "  Step 1/5: Stopping PoS stack..."
	@$(MAKE) --no-print-directory stop-pos-graceful
	@echo "  Step 2/5: Clearing all PoS data..."
	@$(MAKE) --no-print-directory cleanup-pos-data
	@echo "  Step 3/5: Deploying PoS with new genesis time..."
	@$(MAKE) --no-print-directory deploy-pos
	@echo "  Step 4/5: Waiting for pods to start (60s)..."
	@sleep 60
	@echo "  Step 5/5: Setting up beacon P2P peering..."
	@$(MAKE) --no-print-directory setup-beacon-peers
	@echo ""
	@echo "✅ PoS chain reset complete! New chain is running."
	@echo "   Monitor with: make check-pos-status"

deploy-all: deploy-pos deploy-blockscout deploy-minio deploy-auth ## Deploy all components (PoS + Blockscout + MinIO + Auth)

delete-all: delete-auth delete-minio delete-blockscout delete-pos ## Delete all components

# Port Forwarding

port-forward-geth-rpc: ## Port forward Geth RPC (8545:8545)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/geth-rpc 8545:8545

port-forward-geth-ws: ## Port forward Geth WebSocket (8546:8546)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/geth-rpc 8546:8546

port-forward-beacon-api: ## Port forward Beacon HTTP API (3500:3500)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/beacon-api 3500:3500

port-forward-beacon-grpc: ## Port forward Beacon gRPC (4000:4000)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/beacon-api 4000:4000

port-forward-blockscout-backend: ## Port forward Blockscout Backend API (4001:4000)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/blockscout-backend 4001:4000

port-forward-blockscout: ## Port forward Blockscout (3001:80 via nginx proxy)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/blockscout-proxy 3001:80

port-forward-blockscout-postgres: ## Port forward Blockscout Postgres (5433:5432)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/blockscout-postgres 5433:5432

port-forward-blockscout-stats: ## Port forward Blockscout Stats (8150:8050)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/blockscout-stats 8150:8050

port-forward-minio: ## Port forward MinIO API (9000) + Console (9001)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/minio 9000:9000 9001:9001

port-forward-auth-postgres: ## Port forward Auth PostgreSQL (5434:5432)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/auth-postgres 5434:5432

port-forward-redis: ## Port forward Redis (6379:6379)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/redis-service 6379:6379

# Init & Verify

init-infra: apply-pv create-namespace ## Initialize infrastructure (PVs + namespace)

verify-nodes: ## Verify all pods are running
	@echo "🔍 Checking Geth pods..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l app=geth
	@echo ""
	@echo "🔍 Checking Beacon pods..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l app=beacon
	@echo ""
	@echo "🔍 Checking Validator pods..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l app=validator
	@echo ""
	@echo "🔍 Checking Blockscout pods..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l app=blockscout-backend
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l app=blockscout-frontend
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l app=blockscout-stats
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l app=blockscout-postgres
	@echo ""
	@echo "🔍 Checking MinIO pods..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l app=minio

check-latest-block: ## Check the latest block number synced across all Geth nodes
	@echo "🔍 Checking latest block number on Geth nodes..."
	@for i in 0 1 2; do \
		BLOCK=$$(kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) exec geth-$${i} -c geth -- \
			geth attach --exec "eth.getBlock('latest').number" /root/.ethereum/geth.ipc 2>/dev/null | tr -d '\r\n'); \
		echo "  geth-$${i}: Block $${BLOCK:-❌ unreachable}"; \
	done

check-pos-status: ## Deep health check for PoS cluster (geth peers, beacon sync, validator)
	@echo "═══════════════════════════════════════════════════"
	@echo "  PoS Cluster Status Check"
	@echo "═══════════════════════════════════════════════════"
	@echo ""
	@echo "📦 Pod Status"
	@echo "───────────────────────────────────────────────────"
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l 'app in (geth,beacon,validator)' -o wide
	@echo ""
	@echo "⛓️  Geth Peer Count"
	@echo "───────────────────────────────────────────────────"
	@for i in 0 1 2; do \
		PEERS=$$(kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) exec geth-$${i} -- \
			geth attach --exec "admin.peers.length" /root/.ethereum/geth.ipc 2>/dev/null); \
		echo "  geth-$${i}: $${PEERS:-❌ unreachable} peers"; \
	done
	@echo ""
	@echo "🔭 Beacon Sync Status"
	@echo "───────────────────────────────────────────────────"
	@for i in 0 1 2; do \
		LOCAL_PORT=$$((3510 + i)); \
		kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward pod/beacon-$${i} $${LOCAL_PORT}:3500 &>/dev/null & \
		PF_PID=$$!; \
		sleep 1; \
		RESULT=$$(curl -sf http://localhost:$${LOCAL_PORT}/eth/v1/node/syncing 2>/dev/null); \
		HEAD_SLOT=$$(echo "$${RESULT}" | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print(f'slot={d[\"head_slot\"]} syncing={d[\"is_syncing\"]}')" 2>/dev/null); \
		PEERS=$$(curl -sf http://localhost:$${LOCAL_PORT}/eth/v1/node/peer_count 2>/dev/null | \
			python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print(f'connected={d[\"connected\"]} disconnected={d[\"disconnected\"]}')" 2>/dev/null); \
		kill $${PF_PID} 2>/dev/null; wait $${PF_PID} 2>/dev/null; \
		echo "  beacon-$${i}: $${HEAD_SLOT:-❌ unreachable}  |  peers: $${PEERS:-N/A}"; \
	done
	@echo ""
	@echo "✅ Validator Status"
	@echo "───────────────────────────────────────────────────"
	@for i in 0 1 2; do \
		STATUS=$$(kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pod validator-$${i} -o jsonpath='{.status.phase}' 2>/dev/null); \
		READY=$$(kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pod validator-$${i} -o jsonpath='{.status.containerStatuses[0].ready}' 2>/dev/null); \
		RESTARTS=$$(kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pod validator-$${i} -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null); \
		echo "  validator-$${i}: status=$${STATUS:-❌} ready=$${READY:-false} restarts=$${RESTARTS:-?}"; \
	done
	@echo ""
	@echo "═══════════════════════════════════════════════════"

# Smart Contracts

compile-contracts: ## Compile Hardhat smart contracts within the workspace
	pixi run npm run compile --workspace=contracts

test-contracts: ## Run Hardhat tests for smart contracts
	pixi run npm run test --workspace=contracts

clean-contracts: ## Remove hardhat artifacts and cache
	rm -rf contracts/artifacts contracts/cache contracts/typechain-types

deploy-contracts: ## Deploy Smart Contracts to local Geth cluster
	pixi run npm run deploy --workspace=contracts

DEPLOYER_PRIVATE_KEY ?= 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
USER_PRIVATE_KEY ?= 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d

fund-paymaster: ## Fund Paymaster with ETH in the EntryPoint
	PRIVATE_KEY=$(DEPLOYER_PRIVATE_KEY) \
	pixi run npm run fund-paymaster --workspace=contracts

test-interact: ## Run the interactive simulation test script (deploys tokens + saves addresses)
	PRIVATE_KEY=$(USER_PRIVATE_KEY) \
	pixi run npm run test-interact --workspace=contracts

# Seed Data

seed-upload: ## Upload seed images & metadata to MinIO (requires port-forward-minio)
	./seed/upload.sh

seed-update-icons: ## Update Blockscout DB with token icons + NFT metadata (requires test-interact first)
	./seed/update-blockscout-icons.sh

# Mock Price API (labETH pricing for Blockscout)

build-mock-price: ## Build mock-price-api Docker image (tagged with VERSION)
	docker build -t $(MOCK_PRICE_IMAGE) -f deployments/build/mock-price/Dockerfile backend/

load-mock-price: ## Load mock-price-api image into Minikube
	minikube -p $(MINIKUBE_PROFILE) image load $(MOCK_PRICE_IMAGE)

deploy-mock-price: kustomize-update-mock-price ## Deploy mock-price-api to the cluster via kustomize
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/mock-price/overlays/minikube/

kustomize-update-mock-price: ## Update mock-price kustomize image tag to current VERSION
	cd deployments/kustomize/mock-price/overlays/minikube && kustomize edit set image mock-price=$(MOCK_PRICE_IMAGE)

delete-mock-price: ## Delete mock-price-api from the cluster
	kubectl --context web3-lab delete -k deployments/kustomize/mock-price/overlays/minikube/ --ignore-not-found

# ============================================================================
# Identity & Authorization Stack
# ============================================================================

# --- Auth PostgreSQL (shared database) ---

deploy-auth-postgres: ## Deploy shared PostgreSQL for auth services
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/postgres/overlays/minikube/
	@echo "Waiting for auth-postgres to be ready..."
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) wait --for=condition=ready pod -l app=auth-postgres --timeout=120s

delete-auth-postgres: ## Delete auth PostgreSQL
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/postgres/overlays/minikube/ --ignore-not-found

# --- Redis (nonce storage) ---

deploy-redis: ## Deploy Redis for wallet auth nonce storage
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/redis/overlays/minikube/
	@echo "Waiting for redis to be ready..."
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) wait --for=condition=ready pod -l app=redis --timeout=60s

delete-redis: ## Delete Redis
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/redis/overlays/minikube/ --ignore-not-found

# --- Hydra (OAuth2) ---

deploy-hydra: ## Deploy Hydra OAuth2 server
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/hydra/overlays/minikube/

hydra-clean-clients: ## Delete all OAuth2 clients via web3-api through APISIX gateway
	@echo "🧹 Cleaning up all existing OAuth2 clients..."
	@IDS=$$(curl -sk https://gateway.web3-local-dev.com/api/v1/admin/clients -H 'X-Admin-Key: web3-admin-secret-key' | python3 -c "import sys,json; data=json.load(sys.stdin); print(' '.join([c['id'] for c in data.get('clients', [])]))" 2>/dev/null) && \
	if [ -n "$$IDS" ]; then \
		for id in $$IDS; do \
			echo "  🗑️ Deleting client $$id..."; \
			curl -sk -X DELETE https://gateway.web3-local-dev.com/api/v1/admin/clients/$$id -H 'X-Admin-Key: web3-admin-secret-key'; \
		done; \
	fi
	@echo "✅ Cleanup complete!"

hydra-seed-clients: ## Create OAuth2 clients and patch frontend configmaps with client IDs
	@echo "🔧 Seeding OAuth2 clients and patching frontend configmaps..."
	@# --- App 1: app.web3-local-dev.com → frontend ---
	@CID1=$$(curl -sk -X POST https://gateway.web3-local-dev.com/api/v1/admin/clients \
		-H 'Content-Type: application/json' \
		-H 'X-Admin-Key: web3-admin-secret-key' \
		-d '{"name":"Web3 Test App","frontend_url":"https://app.web3-local-dev.com","login_path":"/login","logout_url":"https://app.web3-local-dev.com/logout","allowed_cors_origins":["https://app.web3-local-dev.com","https://gateway.web3-local-dev.com"]}' \
		| python3 -c "import sys,json; print(json.load(sys.stdin)['client']['oauth2_client_id'])") && \
	echo "  ✅ app.web3-local-dev.com → $$CID1" && \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) create configmap web3-frontend-config \
		--from-literal=config.json="{\"gatewayUrl\":\"https://gateway.web3-local-dev.com\",\"clientId\":\"$$CID1\",\"authDomain\":\"app.web3-local-dev.com\"}" \
		--dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -
	@# --- App 2: app.web3-local-dev-2.com → frontend-2 ---
	@CID2=$$(curl -sk -X POST https://gateway.web3-local-dev.com/api/v1/admin/clients \
		-H 'Content-Type: application/json' \
		-H 'X-Admin-Key: web3-admin-secret-key' \
		-d '{"name":"Web3 Test App 2","frontend_url":"https://app.web3-local-dev-2.com","login_path":"/login","logout_url":"https://app.web3-local-dev-2.com/logout","allowed_cors_origins":["https://app.web3-local-dev-2.com","https://gateway.web3-local-dev.com"]}' \
		| python3 -c "import sys,json; print(json.load(sys.stdin)['client']['oauth2_client_id'])") && \
	echo "  ✅ app.web3-local-dev-2.com → $$CID2" && \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) create configmap web3-frontend-2-config \
		--from-literal=config.json="{\"gatewayUrl\":\"https://gateway.web3-local-dev.com\",\"clientId\":\"$$CID2\",\"authDomain\":\"app.web3-local-dev-2.com\"}" \
		--dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -
	@# --- App 3: app.web3-local-dev.net → frontend-3 ---
	@CID3=$$(curl -sk -X POST https://gateway.web3-local-dev.com/api/v1/admin/clients \
		-H 'Content-Type: application/json' \
		-H 'X-Admin-Key: web3-admin-secret-key' \
		-d '{"name":"Web3 Test App Net","frontend_url":"https://app.web3-local-dev.net","login_path":"/login","logout_url":"https://app.web3-local-dev.net/logout","allowed_cors_origins":["https://app.web3-local-dev.net","https://gateway.web3-local-dev.com"]}' \
		| python3 -c "import sys,json; print(json.load(sys.stdin)['client']['oauth2_client_id'])") && \
	echo "  ✅ app.web3-local-dev.net → $$CID3" && \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) create configmap web3-frontend-3-config \
		--from-literal=config.json="{\"gatewayUrl\":\"https://gateway.web3-local-dev.com\",\"clientId\":\"$$CID3\",\"authDomain\":\"app.web3-local-dev.net\"}" \
		--dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -
	@# --- App 4: app.web3-local-dev-2.net → frontend-4 ---
	@CID4=$$(curl -sk -X POST https://gateway.web3-local-dev.com/api/v1/admin/clients \
		-H 'Content-Type: application/json' \
		-H 'X-Admin-Key: web3-admin-secret-key' \
		-d '{"name":"Web3 Test App 2 Net","frontend_url":"https://app.web3-local-dev-2.net","login_path":"/login","logout_url":"https://app.web3-local-dev-2.net/logout","allowed_cors_origins":["https://app.web3-local-dev-2.net","https://gateway.web3-local-dev.com"]}' \
		| python3 -c "import sys,json; print(json.load(sys.stdin)['client']['oauth2_client_id'])") && \
	echo "  ✅ app.web3-local-dev-2.net → $$CID4" && \
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) create configmap web3-frontend-4-config \
		--from-literal=config.json="{\"gatewayUrl\":\"https://gateway.web3-local-dev.com\",\"clientId\":\"$$CID4\",\"authDomain\":\"app.web3-local-dev-2.net\"}" \
		--dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -
	@# --- Restart frontends to pick up new configmaps ---
	@echo "🔄 Restarting frontends..."
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/web3-frontend 2>/dev/null || true
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/web3-frontend-2 2>/dev/null || true
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/web3-frontend-3 2>/dev/null || true
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/web3-frontend-4 2>/dev/null || true
	@echo "✅ All 4 OAuth2 clients seeded and frontend configmaps patched!"

hydra-get-clients: ## List all registered OAuth2 clients via web3-api through APISIX gateway
	@curl -skf -H 'X-Admin-Key: web3-admin-secret-key' https://gateway.web3-local-dev.com/api/v1/admin/clients | python3 -m json.tool

delete-hydra: ## Delete Hydra
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/hydra/overlays/minikube/ --ignore-not-found

restart-hydra: ## Rollout restart Hydra
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/hydra-deployment

# --- Kratos (Identity) ---

deploy-kratos: ## Deploy Kratos identity management
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/kratos/overlays/minikube/

delete-kratos: ## Delete Kratos
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/kratos/overlays/minikube/ --ignore-not-found

restart-kratos: ## Rollout restart Kratos
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/kratos-deployment

# --- Oathkeeper (API Gateway) ---

deploy-oathkeeper: ## Deploy Oathkeeper API gateway
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/oathkeeper/overlays/minikube/

delete-oathkeeper: ## Delete Oathkeeper
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/oathkeeper/overlays/minikube/ --ignore-not-found

restart-oathkeeper: ## Rollout restart Oathkeeper
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/oathkeeper-deployment

# --- SpiceDB (Authorization) ---

deploy-spicedb: ## Deploy SpiceDB authorization (PostgreSQL datastore)
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/spicedb/overlays/minikube/

delete-spicedb: ## Delete SpiceDB
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/spicedb/overlays/minikube/ --ignore-not-found

restart-spicedb: ## Rollout restart SpiceDB
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/spicedb-deployment

spicedb-schema: ## Apply SpiceDB schema using zed CLI
	@if ! command -v zed >/dev/null 2>&1; then \
		echo "Error: zed cli not found. Please install it first."; \
		exit 1; \
	fi
	@echo "Applying SpiceDB schema..."
	@zed schema write migrations/spicedb/schema.zed --endpoint localhost:50051 --insecure --token "web3-lab-spicedb-key-not-for-production"

spicedb-verify: ## Verify SpiceDB schema using zed CLI
	@if ! command -v zed >/dev/null 2>&1; then \
		echo "Error: zed cli not found. Please install it first."; \
		exit 1; \
	fi
	@echo "Reading SpiceDB schema..."
	@zed schema read --endpoint localhost:50051 --insecure --token "web3-lab-spicedb-key-not-for-production"

port-forward-spicedb: ## Port-forward SpiceDB gRPC port (50051) to localhost
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) port-forward svc/spicedb-grpc 50051:50051

# --- Frontend App ---

FRONTEND_IMAGE ?= web3-frontend:$(TAG)

# --- Aggregate Frontend Commands ---
build-all-frontends: build-frontend build-frontend-2 build-frontend-3 build-frontend-4 ## Build all 4 frontend images
load-all-frontends: load-frontend load-frontend-2 load-frontend-3 load-frontend-4 ## Load all 4 frontend images into Minikube
deploy-all-frontends: deploy-frontend deploy-frontend-2 deploy-frontend-3 deploy-frontend-4 ## Deploy all 4 frontends and patch auth configs
	@echo "All frontends deployed. Synchronizing OAuth2 ConfigMaps..."
	$(MAKE) hydra-seed-clients

build-frontend: ## Build frontend Docker image
	docker build -t $(FRONTEND_IMAGE) -f frontend/app/Dockerfile frontend/app/

load-frontend: ## Load frontend image into Minikube
	minikube -p $(MINIKUBE_PROFILE) image load $(FRONTEND_IMAGE)

deploy-frontend: kustomize-update-frontend ## Deploy frontend
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/frontend/overlays/minikube/

kustomize-update-frontend: ## Update frontend kustomize image tag to current VERSION
	cd deployments/kustomize/frontend/overlays/minikube && kustomize edit set image web3-frontend=$(FRONTEND_IMAGE)

delete-frontend: ## Delete frontend
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/frontend/overlays/minikube/ --ignore-not-found

restart-frontend: ## Rollout restart web3-frontend
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/web3-frontend

# --- Frontend App 2 ---

FRONTEND_2_IMAGE ?= web3-frontend-2:$(TAG)

build-frontend-2: ## Build frontend-2 Docker image
	docker build -t $(FRONTEND_2_IMAGE) -f frontend/app-2/Dockerfile frontend/app-2/

load-frontend-2: ## Load frontend-2 image into Minikube
	minikube -p $(MINIKUBE_PROFILE) image load $(FRONTEND_2_IMAGE)

deploy-frontend-2: kustomize-update-frontend-2 ## Deploy frontend-2
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/frontend-2/overlays/minikube/

kustomize-update-frontend-2: ## Update frontend-2 kustomize image tag to current VERSION
	cd deployments/kustomize/frontend-2/overlays/minikube && kustomize edit set image web3-frontend-2=$(FRONTEND_2_IMAGE)

delete-frontend-2: ## Delete frontend-2
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/frontend-2/overlays/minikube/ --ignore-not-found

restart-frontend-2: ## Rollout restart web3-frontend-2
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/web3-frontend-2

# --- Frontend App 3 ---

FRONTEND_3_IMAGE ?= web3-frontend-3:$(TAG)

build-frontend-3: ## Build frontend-3 Docker image
	docker build -t $(FRONTEND_3_IMAGE) -f frontend/app-3/Dockerfile frontend/app-3/

load-frontend-3: ## Load frontend-3 image into Minikube
	minikube -p $(MINIKUBE_PROFILE) image load $(FRONTEND_3_IMAGE)

deploy-frontend-3: kustomize-update-frontend-3 ## Deploy frontend-3
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/frontend-3/overlays/minikube/

kustomize-update-frontend-3: ## Update frontend-3 kustomize image tag to current VERSION
	cd deployments/kustomize/frontend-3/overlays/minikube && kustomize edit set image web3-frontend-3=$(FRONTEND_3_IMAGE)

delete-frontend-3: ## Delete frontend-3
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/frontend-3/overlays/minikube/ --ignore-not-found

restart-frontend-3: ## Rollout restart web3-frontend-3
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/web3-frontend-3

# --- Frontend App 4 ---

FRONTEND_4_IMAGE ?= web3-frontend-4:$(TAG)

build-frontend-4: ## Build frontend-4 Docker image
	docker build -t $(FRONTEND_4_IMAGE) -f frontend/app-4/Dockerfile frontend/app-4/

load-frontend-4: ## Load frontend-4 image into Minikube
	minikube -p $(MINIKUBE_PROFILE) image load $(FRONTEND_4_IMAGE)

deploy-frontend-4: kustomize-update-frontend-4 ## Deploy frontend-4
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/frontend-4/overlays/minikube/

kustomize-update-frontend-4: ## Update frontend-4 kustomize image tag to current VERSION
	cd deployments/kustomize/frontend-4/overlays/minikube && kustomize edit set image web3-frontend-4=$(FRONTEND_4_IMAGE)

delete-frontend-4: ## Delete frontend-4
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/frontend-4/overlays/minikube/ --ignore-not-found

restart-frontend-4: ## Rollout restart web3-frontend-4
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/web3-frontend-4

# --- Backend API ---

API_IMAGE ?= web3-account-api:$(TAG)

build-api: ## Build web3-account-api Docker image
	docker build -t $(API_IMAGE) -f deployments/build/api/Dockerfile .

load-api: ## Load web3-account-api image into Minikube
	minikube -p $(MINIKUBE_PROFILE) image load $(API_IMAGE)

deploy-api: kustomize-update-api ## Deploy backend API
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/api/overlays/minikube/

kustomize-update-api: ## Update api kustomize image tag to current VERSION
	cd deployments/kustomize/api/overlays/minikube && kustomize edit set image web3-account-api=$(API_IMAGE)

delete-api: ## Delete backend API
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/api/overlays/minikube/ --ignore-not-found

restart-api: ## Rollout restart web3-api
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) rollout restart deployment/web3-api

# --- APISIX Auth Gateway ---

apisix-install: ## Install APISIX via Helm (etcd + gateway + ingress controller)
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kubernetes/minikube/storage/apisix-storage.yaml
	helm install apisix apisix/apisix --namespace apisix --create-namespace --kube-context $(MINIKUBE_PROFILE) -f deployments/helm/apisix-values.yaml
	@echo "⏳ Waiting for APISIX pods..."
	kubectl --context $(MINIKUBE_PROFILE) -n apisix wait --for=condition=ready pod -l app.kubernetes.io/name=etcd --timeout=180s
	kubectl --context $(MINIKUBE_PROFILE) -n apisix wait --for=condition=ready pod -l app.kubernetes.io/name=apisix --timeout=120s
	@echo "🔗 Creating GatewayProxy..."
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kustomize/apisix-auth-gateway/base/gateway-proxy.yaml
	@echo "🔗 Linking IngressClass to GatewayProxy..."
	kubectl --context $(MINIKUBE_PROFILE) patch ingressclass apisix --type merge -p '{"spec":{"parameters":{"apiGroup":"apisix.apache.org","kind":"GatewayProxy","name":"apisix","namespace":"apisix","scope":"Namespace"}}}'
	@echo "✅ APISIX installed and configured!"

apisix-uninstall: ## Uninstall APISIX (Helm + PVCs + PVs + host data)
	helm uninstall apisix -n apisix --kube-context $(MINIKUBE_PROFILE) || true
	kubectl --context $(MINIKUBE_PROFILE) delete pvc --all -n apisix || true
	kubectl --context $(MINIKUBE_PROFILE) delete pv apisix-etcd-0-pv apisix-etcd-1-pv apisix-etcd-2-pv --ignore-not-found
	minikube ssh -p $(MINIKUBE_PROFILE) -- sudo rm -rf /data/apisix
	minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m02 -- sudo rm -rf /data/apisix
	minikube ssh -p $(MINIKUBE_PROFILE) -n $(MINIKUBE_PROFILE)-m03 -- sudo rm -rf /data/apisix

deploy-apisix-auth-gateway: ## Deploy APISIX auth gateway routes (public + admin)
	@echo "📦 Applying routes + consumer (web3 namespace)..."
	kubectl --context $(MINIKUBE_PROFILE) apply -k deployments/kustomize/apisix-auth-gateway/overlays/minikube/
	@echo "🌐 Applying ingress + certificate (apisix namespace)..."
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kustomize/apisix-auth-gateway/overlays/minikube/ingress.yaml
	kubectl --context $(MINIKUBE_PROFILE) apply -f deployments/kustomize/apisix-auth-gateway/overlays/minikube/certificate.yaml
	@echo "✅ APISIX auth gateway deployed!"

delete-apisix-auth-gateway: ## Delete APISIX auth gateway routes
	kubectl --context $(MINIKUBE_PROFILE) delete -k deployments/kustomize/apisix-auth-gateway/overlays/minikube/ --ignore-not-found
	kubectl --context $(MINIKUBE_PROFILE) delete ingress apisix-gateway-ingress -n apisix --ignore-not-found
	kubectl --context $(MINIKUBE_PROFILE) delete certificate gateway-certificate -n apisix --ignore-not-found

# --- Account API Migrations ---

GOBIN ?= $(shell go env GOPATH)/bin
MIGRATE_DSN ?= postgres://postgres:postgres@127.0.0.1:5434/account?sslmode=disable
MIGRATE_PATH ?= migrations/postgres

migrate-up: ## Run all pending Account API migrations (requires port-forward-auth-postgres)
	$(GOBIN)/migrate -path $(MIGRATE_PATH) -database "$(MIGRATE_DSN)" up
	@echo "✅ Migrations applied"

migrate-down: ## Rollback last Account API migration (requires port-forward-auth-postgres)
	$(GOBIN)/migrate -path $(MIGRATE_PATH) -database "$(MIGRATE_DSN)" down 1
	@echo "✅ Rolled back 1 migration"

migrate-status: ## Show current Account API migration version
	$(GOBIN)/migrate -path $(MIGRATE_PATH) -database "$(MIGRATE_DSN)" version

migrate-create: ## Create a new migration file (use: make migrate-create NAME=add_foo)
	$(GOBIN)/migrate create -ext sql -dir $(MIGRATE_PATH) -seq $(NAME)
	@echo "✅ Created migration files for: $(NAME)"

sqlc-generate: ## Generate Go code from SQL queries via sqlc
	cd backend && $(GOBIN)/sqlc generate
	@echo "✅ sqlc generated backend/internal/database/sqlc/"

# --- Auth Aggregate ---

tls-setup: ## Extract local CA and configure macOS Keychain / /etc/hosts for Auth URLs
	@echo "🔐 Extracting Cert-Manager Root CA..."
	@kubectl --context $(MINIKUBE_PROFILE) get secret root-ca-secret -n cert-manager -o jsonpath='{.data.ca\.crt}' | base64 -d > /tmp/root-ca.crt
	@echo "🔑 Trusting Root CA in macOS Keychain (may prompt for password)..."
	@sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain /tmp/root-ca.crt || echo "Warning: Failed to add cert"
	@rm -f /tmp/root-ca.crt
	@echo "🌐 Mapping 127.0.0.1 to /etc/hosts (may prompt for password)..."
	@sudo sh -c "sed -i.bak '/# BEGIN Web3-Lab Local Auth/,/# END Web3-Lab Local Auth/d' /etc/hosts && echo \"\n# BEGIN Web3-Lab Local Auth\n127.0.0.1 gateway.web3-local-dev.com\n127.0.0.1 hydra.web3-local-dev.com\n127.0.0.1 hydra-admin.web3-local-dev.com\n127.0.0.1 kratos.web3-local-dev.com\n127.0.0.1 kratos-admin.web3-local-dev.com\n127.0.0.1 auth.web3-local-dev.com\n127.0.0.1 auth-api.web3-local-dev.com\n127.0.0.1 spicedb.web3-local-dev.com\n127.0.0.1 api.web3-local-dev.com\n127.0.0.1 app.web3-local-dev.com\n127.0.0.1 blockscout.web3-local-dev.com\n127.0.0.1 blockscout-api.web3-local-dev.com\n127.0.0.1 blockscout-stats.web3-local-dev.com\n# END Web3-Lab Local Auth\" >> /etc/hosts && rm -f /etc/hosts.bak"
	@echo "✅ TLS Setup complete! You can now visit https://kratos.web3-local-dev.com without browser warnings."

deploy-auth: deploy-auth-postgres deploy-redis deploy-hydra deploy-kratos deploy-oathkeeper deploy-spicedb deploy-api deploy-apisix-auth-gateway ## Deploy all auth services

delete-auth: delete-apisix-auth-gateway delete-api delete-spicedb delete-oathkeeper delete-kratos delete-hydra delete-redis delete-auth-postgres ## Delete all auth services

restart-auth: restart-hydra restart-kratos restart-oathkeeper restart-spicedb restart-api restart-frontend ## Rollout restart all auth services

cleanup-auth-pvc: ## Delete all Auth PVCs (postgres, redis)
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete pvc -l app=auth-postgres --ignore-not-found=true
	kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) delete pvc -l app=redis --ignore-not-found=true

cleanup-auth-pv: ## Delete all Auth PVs (postgres, redis)
	kubectl --context $(MINIKUBE_PROFILE) delete pv auth-postgres-pv-0 redis-pv-0 --ignore-not-found=true

cleanup-auth-data: ## Wipe Auth data (postgres, redis) from minikube nodes
	minikube ssh -p $(MINIKUBE_PROFILE) -- sudo rm -rf /data/auth/postgres/data0 /data/auth/redis/data0

check-auth-status: ## Check status of all auth service pods
	@echo "=== Auth Services Status ==="
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get pods -l 'app in (auth-postgres,redis,hydra,kratos,oathkeeper,spicedb,web3-api)' -o wide 2>/dev/null || echo "No auth pods found"
	@echo ""
	@echo "=== Auth Services ==="
	@kubectl --context $(MINIKUBE_PROFILE) -n $(NAMESPACE) get svc -l 'component in (database,cache,oauth2-server,identity,api-gateway,authorization,backend)' 2>/dev/null || echo "No auth services found"
