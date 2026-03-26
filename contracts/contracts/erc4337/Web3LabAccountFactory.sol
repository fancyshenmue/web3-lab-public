// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/utils/Create2.sol";
import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import "@account-abstraction/contracts/core/EntryPoint.sol";
import "./Web3LabAccount.sol";

/**
 * @title Web3LabAccountFactory
 * @dev A factory that deploys Web3LabAccount proxies using CREATE2, allowing 
 * deterministic addressing based on the owner's Session Key / Identity.
 */
contract Web3LabAccountFactory {
    Web3LabAccount public immutable accountImplementation;

    constructor(IEntryPoint _entryPoint) {
        accountImplementation = new Web3LabAccount(_entryPoint);
    }

    /**
     * @dev Create an account, and return its address.
     * Returns the address even if the account is already deployed.
     * Note that during UserOperation execution, this method is called only if the account is not deployed.
     */
    function createAccount(address owner, uint256 salt) public returns (Web3LabAccount ret) {
        address addr = getAddress(owner, salt);
        uint codeSize = addr.code.length;
        if (codeSize > 0) {
            return Web3LabAccount(payable(addr));
        }
        ret = Web3LabAccount(payable(new ERC1967Proxy{salt : bytes32(salt)}(
            address(accountImplementation),
            abi.encodeCall(SimpleAccount.initialize, (owner))
        )));
    }

    /**
     * @dev Calculate the counterfactual address of this account as it would be returned by createAccount()
     */
    function getAddress(address owner, uint256 salt) public view returns (address) {
        return Create2.computeAddress(bytes32(salt), keccak256(abi.encodePacked(
            type(ERC1967Proxy).creationCode,
            abi.encode(
                address(accountImplementation),
                abi.encodeCall(SimpleAccount.initialize, (owner))
            )
        )));
    }
}
