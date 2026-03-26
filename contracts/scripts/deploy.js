import { ethers } from "ethers";
import hre from "hardhat";

async function main() {
  console.log("Starting Web3Lab smart contract deployment...");

  // Connect securely to the active RPC endpoint
  const url = process.env.GETH_RPC_URL || "http://127.0.0.1:8545";
  const provider = new ethers.JsonRpcProvider(url);
  let deployer;

  if (process.env.PRIVATE_KEY) {
    console.log("Using ENV injected PRIVATE_KEY.");
    deployer = new ethers.Wallet(process.env.PRIVATE_KEY, provider);
  } else {
    console.log("No PRIVATE_KEY provided. Using default Hardhat Signer 0.");
    deployer = await provider.getSigner(0);
  }

  console.log("Deploying contracts with the account:", deployer.address);

  // 1. EntryPoint
  const epArtifact = await hre.artifacts.readArtifact("Web3LabEntryPoint");
  const epFactory = new ethers.ContractFactory(epArtifact.abi, epArtifact.bytecode, deployer);
  const entryPoint = await epFactory.deploy();
  await entryPoint.waitForDeployment();
  const entryPointAddr = await entryPoint.getAddress();
  console.log(`EntryPoint deployed to: ${entryPointAddr}`);

  // 2. Factory
  const facArtifact = await hre.artifacts.readArtifact("Web3LabAccountFactory");
  const factoryFact = new ethers.ContractFactory(facArtifact.abi, facArtifact.bytecode, deployer);
  const factory = await factoryFact.deploy(entryPointAddr);
  await factory.waitForDeployment();
  console.log(`Web3LabAccountFactory deployed to: ${await factory.getAddress()}`);

  const erc20Art = await hre.artifacts.readArtifact("Web3LabERC20Factory");
  const erc20FactoryFact = new ethers.ContractFactory(erc20Art.abi, erc20Art.bytecode, deployer);
  const erc20Factory = await erc20FactoryFact.deploy();
  await erc20Factory.waitForDeployment();
  console.log(`Web3LabERC20Factory deployed to: ${await erc20Factory.getAddress()}`);

  const erc721Art = await hre.artifacts.readArtifact("Web3LabERC721Factory");
  const erc721FactoryFact = new ethers.ContractFactory(erc721Art.abi, erc721Art.bytecode, deployer);
  const erc721Factory = await erc721FactoryFact.deploy();
  await erc721Factory.waitForDeployment();
  console.log(`Web3LabERC721Factory deployed to: ${await erc721Factory.getAddress()}`);

  const erc1155Art = await hre.artifacts.readArtifact("Web3LabERC1155Factory");
  const erc1155FactoryFact = new ethers.ContractFactory(erc1155Art.abi, erc1155Art.bytecode, deployer);
  const erc1155Factory = await erc1155FactoryFact.deploy();
  await erc1155Factory.waitForDeployment();
  console.log(`Web3LabERC1155Factory deployed to: ${await erc1155Factory.getAddress()}`);



  // 4. Paymaster
  const payArt = await hre.artifacts.readArtifact("Web3LabPaymaster");
  const paymasterFact = new ethers.ContractFactory(payArt.abi, payArt.bytecode, deployer);
  // Backend signer is typically a dedicated key. Using deployer for mock tests.
  const paymaster = await paymasterFact.deploy(entryPointAddr, deployer.address);
  await paymaster.waitForDeployment();
  console.log(`Web3LabPaymaster deployed to: ${await paymaster.getAddress()}`);

  const deployments = {
    EntryPoint: entryPointAddr,
    Web3LabAccountFactory: await factory.getAddress(),
    Web3LabERC20Factory: await erc20Factory.getAddress(),
    Web3LabERC721Factory: await erc721Factory.getAddress(),
    Web3LabERC1155Factory: await erc1155Factory.getAddress(),
    Web3LabPaymaster: await paymaster.getAddress()
  };

  import("fs").then(fs => {
    fs.writeFileSync("deployments.json", JSON.stringify(deployments, null, 2));
    console.log("\n✅ Deployment Complete!");
    console.log("Contract addresses have been automatically saved to contracts/deployments.json!");
  });
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
