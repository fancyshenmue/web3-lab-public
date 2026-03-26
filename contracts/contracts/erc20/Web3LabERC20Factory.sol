// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "./Web3LabERC20.sol";

contract Web3LabERC20Factory {
    event ERC20Created(address indexed tokenAddress, string name, string symbol, address indexed creator);

    mapping(address => address[]) public userDeployedContracts;

    function getContracts(address owner) public view returns (address[] memory) {
        return userDeployedContracts[owner];
    }

    function createToken(string memory name, string memory symbol, uint8 decimals, uint256 initialSupply) public returns (address) {
        Web3LabERC20 newToken = new Web3LabERC20(name, symbol, decimals, initialSupply, msg.sender);
        userDeployedContracts[msg.sender].push(address(newToken));
        emit ERC20Created(address(newToken), name, symbol, msg.sender);
        return address(newToken);
    }
}
