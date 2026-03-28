// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract Web3LabERC1155 is ERC1155, Ownable {
    string public name;
    string public symbol;

    constructor(string memory name_, string memory symbol_, string memory uri_, address initialOwner) 
        ERC1155(uri_) 
        Ownable(initialOwner) 
    {
        name = name_;
        symbol = symbol_;
    }

    function mint(address account, uint256 id, uint256 amount, bytes memory data) public onlyOwner {
        _mint(account, id, amount, data);
    }

    function mintBatch(address to, uint256[] memory ids, uint256[] memory amounts, bytes memory data) public onlyOwner {
        _mintBatch(to, ids, amounts, data);
    }

    function setURI(string memory newuri) public onlyOwner {
        _setURI(newuri);
    }
}
