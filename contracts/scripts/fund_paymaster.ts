import hre from "hardhat";

async function main() {
  const { ethers } = hre;
  const ENTRY_POINT_ADDRESS = "0x5FbDB2315678afecb367f032d93F642f64180aa3";
  const PAYMASTER_ADDRESS = "0x5FC8d32690cc91D4c39d9d3abcBD16989F875707";

  const [signer] = await ethers.getSigners();
  console.log("Funding Paymaster using account:", signer.address);

  const entryPoint = await ethers.getContractAt("EntryPoint", ENTRY_POINT_ADDRESS);
  
  const tx = await entryPoint.depositTo(PAYMASTER_ADDRESS, { value: ethers.parseEther("100") });
  await tx.wait();

  console.log("Successfully deposited 100 ETH to Paymaster in EntryPoint!");
  const depositInfo = await entryPoint.deposits(PAYMASTER_ADDRESS);
  console.log("Current deposit:", ethers.formatEther(depositInfo.deposit), "ETH");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
