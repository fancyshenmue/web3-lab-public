# Web2.5 Smart Wallet Bridge Architecture

## Overview

This document outlines the detailed execution steps, off-chain/on-chain component boundaries, and security rationales for the Web2.5 Smart Wallet Bridge. By linking Ory Kratos/Hydra authentication with an ERC-4337 Smart Contract Wallet, we provide frictionless "Web2-style" transaction execution for Web3 assets.

## 1. Complete Workflow

| Step                       | Actor                 | Action                                                                                    |
| :------------------------- | :-------------------- | :---------------------------------------------------------------------------------------- |
| **1. Authentication**      | Kratos/Hydra          | User logs in, frontend receives JWT.                                                      |
| **2. Intent Construction** | Browser/Go Backend    | Converts JWT to ZK Proof, embeds into `UserOperation`.                                    |
| **3. Request Submission**  | Bundler (Role)        | Receives `UserOp`, simulates and validates ZK Proof locally.                              |
| **4. On-chain Execution**  | EntryPoint (Contract) | After Bundler submits transaction, EntryPoint calls wallet contract for final validation. |
| **5. Fee Settlement**      | Paymaster (Contract)  | After successful validation, Paymaster pays Gas fees to Bundler.                          |

## 2. ERC-4337 Components: Off-chain vs On-chain

| Component                | Environment | Role           | Description                                                                                                                                |
| :----------------------- | :---------- | :------------- | :----------------------------------------------------------------------------------------------------------------------------------------- |
| **UserOperation**        | Off-chain   | Data Object    | Just a JSON payload. Exists in Browser or Go Backend and is sent to Bundler. Not an actual Ethereum transaction.                           |
| **Bundler**              | Off-chain   | Node Service   | Program running on a server (e.g., Go-Ethereum variant). Collects UserOps, wraps them into actual transactions, and submits them on-chain. |
| **Paymaster (Service)**  | Off-chain   | API Service    | Go Backend. Validates Hydra JWT and decides whether to sign a Gas sponsorship permission.                                                  |
| **EntryPoint**           | On-chain    | Core Contract  | Official singleton contract. Orchestrates the flow, validates UserOps, and deducts Gas.                                                    |
| **Paymaster (Contract)** | On-chain    | Smart Contract | Deployed on-chain. Holds ETH deposits and pays Gas fees upon successful validation.                                                        |
| **Account Contract**     | On-chain    | User Wallet    | User's smart contract address. Contains ZK Proof validation logic to ensure commands only execute if JWT proof is valid.                   |

## 3. Execution Steps & Security Rationale

| Step                | Component    | Location          | Security Rationale                                                                                                          |
| :------------------ | :----------- | :---------------- | :-------------------------------------------------------------------------------------------------------------------------- |
| **1. Login**        | Kratos/Hydra | Backend (Go)      | Passwords and Sessions securely managed by Kratos. Browser only holds a short-lived Session Cookie.                         |
| **2. Request**      | API Call     | Browser -> Go     | User clicks "Execute Transaction," frontend sends command to Backend.                                                       |
| **3. Validation**   | Auth Check   | Backend (Go)      | Backend verifies if Hydra JWT is valid and user has sufficient permissions.                                                 |
| **4. Proving**      | ZK Prover    | Backend / TEE     | Backend generates ZK Proof. JWT never leaves controlled Backend environment. Backend compute power ensures fast generation. |
| **5. Construction** | UserOp Build | Backend (Go)      | Backend assembles `UserOperation` and embeds ZK Proof.                                                                      |
| **6. Sponsorship**  | Paymaster    | Backend (Go)      | Backend signs Paymaster authorization, as it knows this is a legitimately logged-in user.                                   |
| **7. Broadcast**    | Bundler      | Off-chain Service | Backend submits `UserOp` directly to Bundler.                                                                               |
