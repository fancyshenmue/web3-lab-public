# OpenSpec: Minikube Cluster

## Status
Implemented

## Context
The web3-lab requires a local Kubernetes environment to host a multi-node Ethereum PoS cluster. Minikube provides a lightweight multi-node Kubernetes cluster running on Docker, suitable for development and experimentation. The cluster layout mirrors the e-commerce-lab pattern (3 nodes, profile-based isolation, hostPath PVs with nodeAffinity).

## Requirements

### Requirement: Multi-Node Cluster
The cluster SHALL run 3 Minikube nodes using the Docker driver to simulate a distributed environment.
- **Profile**: `web3-lab`
- **Nodes**: 3 (`web3-lab`, `web3-lab-m02`, `web3-lab-m03`)
- **Driver**: Docker
- **Resources**: Configurable memory and CPU via Makefile variables

### Requirement: Namespace Isolation
All web3 workloads SHALL be deployed into a dedicated `web3` namespace.

### Requirement: Persistent Storage
The cluster SHALL use hostPath PersistentVolumes with nodeAffinity to pin storage to specific nodes.
- **Geth (EL)**: 3 PVs × 5Gi at `/data/geth/data{0,1,2}`
- **Beacon (CL)**: 3 PVs × 5Gi at `/data/beacon/data{0,1,2}`
- **Validator**: 3 PVs × 1Gi at `/data/validator/data{0,1,2}`

### Requirement: Makefile Lifecycle
The cluster SHALL be managed via Makefile targets (`minikube-start`, `minikube-stop`, `create-namespace`, `apply-pv`).
