import fs from "fs";
import { ethers } from "ethers";
import hre from "hardhat";

async function main() {
  if (!fs.existsSync("deployments.json")) {
    throw new Error("deployments.json not found! Please run 'make deploy-contracts' first.");
  }
  const ADDRESSES = JSON.parse(fs.readFileSync("deployments.json", "utf-8"));
  
  const ENTRY_POINT_ADDRESS = process.env.ENTRY_POINT_ADDRESS || ADDRESSES.EntryPoint;
  const PAYMASTER_ADDRESS = process.env.PAYMASTER_ADDRESS || ADDRESSES.Web3LabPaymaster;

  const url = process.env.GETH_RPC_URL || "http://127.0.0.1:8545";
  const provider = new ethers.JsonRpcProvider(url);
  let signer;

  if (process.env.PRIVATE_KEY) {
    signer = new ethers.Wallet(process.env.PRIVATE_KEY, provider);
  } else {
    signer = await provider.getSigner(0);
  }

  console.log("Funding Paymaster using account:", signer.address);

  const epArtifact = await hre.artifacts.readArtifact("Web3LabEntryPoint");
  const entryPoint = new ethers.Contract(ENTRY_POINT_ADDRESS, epArtifact.abi, signer);
  
  const tx = await entryPoint.depositTo(PAYMASTER_ADDRESS, { value: ethers.parseEther("100") });
  await tx.wait();

  console.log("Successfully deposited 100 ETH to Paymaster in EntryPoint!");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
