// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "./Web3LabERC1155.sol";

contract Web3LabERC1155Factory {
    event ERC1155Created(address indexed tokenAddress, string name, string symbol, string uri, address indexed creator);

    mapping(address => address[]) public userDeployedContracts;

    function getContracts(address owner) public view returns (address[] memory) {
        return userDeployedContracts[owner];
    }

    function createMultiToken(string memory name, string memory symbol, string memory uri) public returns (address) {
        Web3LabERC1155 newMultiToken = new Web3LabERC1155(name, symbol, uri, msg.sender);
        userDeployedContracts[msg.sender].push(address(newMultiToken));
        emit ERC1155Created(address(newMultiToken), name, symbol, uri, msg.sender);
        return address(newMultiToken);
    }
}
