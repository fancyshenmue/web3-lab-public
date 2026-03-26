// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "@account-abstraction/contracts/core/EntryPoint.sol";

/**
 * @title Web3LabEntryPoint
 * @dev A simple wrapper around the standard ERC-4337 EntryPoint.
 * This ensures Hardhat compiles and emits a deployable Artifact in our workspace
 * rather than pruning it from the node_modules AST.
 */
contract Web3LabEntryPoint is EntryPoint {}
