// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "./Web3LabERC721.sol";

contract Web3LabERC721Factory {
    event ERC721Created(address indexed tokenAddress, string name, string symbol, address indexed creator);

    mapping(address => address[]) public userDeployedContracts;

    function getContracts(address owner) public view returns (address[] memory) {
        return userDeployedContracts[owner];
    }

    function createNFT(string memory name, string memory symbol, string memory baseURI) public returns (address) {
        Web3LabERC721 newNFT = new Web3LabERC721(name, symbol, baseURI, msg.sender);
        userDeployedContracts[msg.sender].push(address(newNFT));
        emit ERC721Created(address(newNFT), name, symbol, msg.sender);
        return address(newNFT);
    }
}
