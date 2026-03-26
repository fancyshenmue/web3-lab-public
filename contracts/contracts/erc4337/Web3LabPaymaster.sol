// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@account-abstraction/contracts/core/BasePaymaster.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

contract Web3LabPaymaster is BasePaymaster {
    address public verifyingSigner;

    constructor(IEntryPoint _entryPoint, address _verifyingSigner) BasePaymaster(_entryPoint) {
        verifyingSigner = _verifyingSigner;
    }

    function setVerifyingSigner(address _signer) external onlyOwner {
        verifyingSigner = _signer;
    }

    function getHash(PackedUserOperation calldata userOp) public view returns (bytes32) {
        // Construct the hash over the relevant UserOp fields to be signed by the backend Paymaster API.
        // Using block.chainid to prevent cross-chain replay attacks.
        return keccak256(abi.encode(
            userOp.sender,
            userOp.nonce,
            keccak256(userOp.initCode),
            keccak256(userOp.callData),
            userOp.accountGasLimits,
            userOp.preVerificationGas,
            userOp.gasFees,
            block.chainid,
            address(this)
        ));
    }

    function _validatePaymasterUserOp(
        PackedUserOperation calldata userOp,
        bytes32 /*userOpHash*/,
        uint256 /*requiredPreFund*/
    ) internal view override returns (bytes memory context, uint256 validationData) {
        // v0.7 paymasterAndData format: [paymasterAddress(20)][paymasterVerificationGasLimit(16)][paymasterPostOpGasLimit(16)][signature]
        require(userOp.paymasterAndData.length >= 52, "Web3LabPaymaster: paymasterAndData too short");
        
        bytes calldata signature = userOp.paymasterAndData[52:];
        require(signature.length == 65, "Web3LabPaymaster: invalid signature length");

        bytes32 hash = MessageHashUtils.toEthSignedMessageHash(getHash(userOp));
        address recoveredSigner = ECDSA.recover(hash, signature);
        
        if (recoveredSigner != verifyingSigner) {
            return ("", 1); // 1 is SIG_VALIDATION_FAILED
        }

        return ("", 0); // 0 is SIG_VALIDATION_SUCCESS
    }
}
