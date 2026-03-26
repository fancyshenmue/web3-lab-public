import { ethers } from "hardhat";

async function main() {
  try {
    const provider = new ethers.JsonRpcProvider("http://127.0.0.1:8545");
    const txHash = "0x0358e8d8741d30c369e81d069a74f4c7580383679f28ad1c2d67d36a64821603";
    
    console.log(`Analyzing transaction: ${txHash}`);
    const receipt = await provider.getTransactionReceipt(txHash);
    
    if (!receipt) {
      console.log("Transaction not found. Is the chain synced?");
      process.exit(1);
    }
    
    console.log(`Status: ${receipt.status === 1 ? "Success" : "Reverted (0)"}`);
    console.log(`Gas Used: ${receipt.gasUsed.toString()}`);
    
    // Simulate the exact call to get the specific revert string
    const tx = await provider.getTransaction(txHash);
    if (!tx) {
        console.log("Tx body not found");
        return;
    }
    
    console.log("Simulating eth_call to extract revert reason...");
    await provider.call({
      to: tx.to,
      data: tx.data,
      value: tx.value,
      from: tx.from,
      gasLimit: tx.gasLimit,
      gasPrice: tx.gasPrice
    });
    
    console.log("Simulation succeeded unexpectedly (no revert!)");
  } catch (error: any) {
    console.log("\n--- REVERT REASON EXTRACTED ---");
    if (error.data) {
        console.log("Raw Error Data:", error.data);
    }
    console.error(error.message);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
