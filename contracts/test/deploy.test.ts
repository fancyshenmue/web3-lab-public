import { expect } from "chai";
import hre from "hardhat";

describe("Web3Lab Smart Contract Assets", function () {
  it("Should deploy the EntryPoint and Web3LabAccountFactory", async function () {
    const { ethers } = hre;
    const EntryPoint = await ethers.getContractFactory("EntryPoint");
    const entryPoint = await EntryPoint.deploy();
    const entryPointAddress = await entryPoint.getAddress();

    const Factory = await ethers.getContractFactory("Web3LabAccountFactory");
    const factory = await Factory.deploy(entryPointAddress);

    expect(await factory.getAddress()).to.not.equal(ethers.ZeroAddress);
  });

  it("Should deploy the Mock Tokens (ERC-20, ERC-721, ERC-1155)", async function () {
    const MockUSDC = await ethers.getContractFactory("MockUSDC");
    const usdc = await MockUSDC.deploy();
    expect(await usdc.name()).to.equal("Mock USDC");

    const MockMembership = await ethers.getContractFactory("MockMembership");
    const membership = await MockMembership.deploy();
    expect(await membership.name()).to.equal("Web3Lab Membership");

    const MockLoyalty = await ethers.getContractFactory("MockLoyalty");
    const loyalty = await MockLoyalty.deploy();
    expect(await loyalty.getAddress()).to.not.equal(ethers.ZeroAddress);
  });

  it("Should deploy the Web3LabPaymaster", async function () {
    const EntryPoint = await ethers.getContractFactory("EntryPoint");
    const entryPoint = await EntryPoint.deploy();

    const [owner, backendSigner] = await ethers.getSigners();
    
    const Paymaster = await ethers.getContractFactory("Web3LabPaymaster");
    const paymaster = await Paymaster.deploy(await entryPoint.getAddress(), backendSigner.address);
    
    expect(await paymaster.verifyingSigner()).to.equal(backendSigner.address);
  });
});
