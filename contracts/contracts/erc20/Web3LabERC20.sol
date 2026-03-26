// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract Web3LabERC20 is ERC20, Ownable {
    uint8 private _customDecimals;

    constructor(
        string memory name_, 
        string memory symbol_, 
        uint8 decimals_,
        uint256 initialSupply_,
        address initialOwner
    ) 
        ERC20(name_, symbol_) 
        Ownable(initialOwner) 
    {
        _customDecimals = decimals_;
        if (initialSupply_ > 0) {
            _mint(initialOwner, initialSupply_ * 10 ** decimals_);
        }
    }

    function decimals() public view virtual override returns (uint8) {
        return _customDecimals;
    }

    function mint(address to, uint256 amount) public onlyOwner {
        _mint(to, amount);
    }
}
