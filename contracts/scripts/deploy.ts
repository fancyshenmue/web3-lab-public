import hre from "hardhat";

async function main() {
  const { ethers } = hre;
  console.log("Starting Web3Lab smart contract deployment...");

  const [deployer] = await ethers.getSigners();
  console.log("Deploying contracts with the account:", deployer.address);

  // 1. EntryPoint
  const EntryPoint = await ethers.getContractFactory("EntryPoint");
  const entryPoint = await EntryPoint.deploy();
  await entryPoint.waitForDeployment();
  const entryPointAddr = await entryPoint.getAddress();
  console.log(`EntryPoint deployed to: ${entryPointAddr}`);

  // 2. Factory
  const Factory = await ethers.getContractFactory("Web3LabAccountFactory");
  const factory = await Factory.deploy(entryPointAddr);
  await factory.waitForDeployment();
  console.log(`Web3LabAccountFactory deployed to: ${await factory.getAddress()}`);

  // 3. Mock Tokens
  const MockUSDC = await ethers.getContractFactory("MockUSDC");
  const usdc = await MockUSDC.deploy();
  await usdc.waitForDeployment();
  console.log(`MockUSDC deployed to: ${await usdc.getAddress()}`);

  const MockMembership = await ethers.getContractFactory("MockMembership");
  const membership = await MockMembership.deploy();
  await membership.waitForDeployment();
  console.log(`MockMembership deployed to: ${await membership.getAddress()}`);

  const MockLoyalty = await ethers.getContractFactory("MockLoyalty");
  const loyalty = await MockLoyalty.deploy();
  await loyalty.waitForDeployment();
  console.log(`MockLoyalty deployed to: ${await loyalty.getAddress()}`);

  // 4. Paymaster
  const Paymaster = await ethers.getContractFactory("Web3LabPaymaster");
  const paymaster = await Paymaster.deploy(entryPointAddr, deployer.address);
  await paymaster.waitForDeployment();
  console.log(`Web3LabPaymaster deployed to: ${await paymaster.getAddress()}`);

  console.log("\nDeployment Complete!");
  console.log("Remember to save these addresses to your frontend/backend configurations!");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
