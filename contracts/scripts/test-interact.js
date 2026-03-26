import { ethers } from "ethers";
import hre from "hardhat";
import fs from "fs";

async function main() {
  if (!fs.existsSync("deployments.json")) {
    throw new Error("deployments.json not found! Please run 'make deploy-contracts' first.");
  }
  const ADDRESSES = JSON.parse(fs.readFileSync("deployments.json", "utf-8"));
  console.log("Starting Web3Lab Contract Interaction Test...");

  const url = process.env.GETH_RPC_URL || "http://127.0.0.1:8545";
  const provider = new ethers.JsonRpcProvider(url);
  
  if (!process.env.PRIVATE_KEY) {
    throw new Error("Please provide your PRIVATE_KEY environment variable.");
  }
  const signer = new ethers.Wallet(process.env.PRIVATE_KEY, provider);
  console.log("Testing with EOA account (SCW Owner):", signer.address);

  // Utility to generate a 4-char random hex/alphanumeric suffix
  const randStr = () => Math.random().toString(36).substring(2, 6).toUpperCase();

  // ----------------------------------------------------------------------
  // 1. CREATE TWO SMART CONTRACT WALLETS (SCW A & B)
  // ----------------------------------------------------------------------
  console.log("\n--- 1. Deploying Smart Contract Wallets (A & B) ---");
  const factoryArt = await hre.artifacts.readArtifact("Web3LabAccountFactory");
  const factory = new ethers.Contract(ADDRESSES.Web3LabAccountFactory, factoryArt.abi, signer);

  // We use salt=0 for A, salt=1 for B
  const walletAAddr = await factory.getFunction("getAddress(address,uint256)")(signer.address, 0);
  const walletBAddr = await factory.getFunction("getAddress(address,uint256)")(signer.address, 1);
  console.log(`Wallet A (Address): ${walletAAddr}`);
  console.log(`Wallet B (Address): ${walletBAddr}`);

  console.log(`Deploying Wallet A...`);
  await (await factory.createAccount(signer.address, 0)).wait();
  console.log(`Deploying Wallet B...`);
  await (await factory.createAccount(signer.address, 1)).wait();
  
  // Attach the Web3LabAccount interface to Wallet A so we can call its 'execute' function
  const accountArt = await hre.artifacts.readArtifact("Web3LabAccount");
  const scwA = new ethers.Contract(walletAAddr, accountArt.abi, signer);


  // Track deployed token addresses for Blockscout icon updates
  const seedAddresses = { erc20: [], erc721: [], erc1155: [] };

  // ----------------------------------------------------------------------
  // 2. ERC20 FACTORY & SCW TRANSFER (x4)
  // ----------------------------------------------------------------------
  console.log("\n--- 2. Testing ERC20 SCW Transfers (4 Tokens) ---");
  const erc20FactoryArt = await hre.artifacts.readArtifact("Web3LabERC20Factory");
  const erc20Factory = new ethers.Contract(ADDRESSES.Web3LabERC20Factory, erc20FactoryArt.abi, signer);
  const erc20Art = await hre.artifacts.readArtifact("Web3LabERC20");

  for (let i = 1; i <= 4; i++) {
    const erc20Suffix = randStr();
    const erc20Symbol = `LAB-${i}-${erc20Suffix}`;
    
    console.log(`\n  Token ${i}: [${erc20Symbol}]`);
    const createErc20Tx = await erc20Factory.createToken(`Lab Token ${i} ${erc20Suffix}`, erc20Symbol);
    const erc20Log = (await createErc20Tx.wait()).logs.find(l => l.fragment && l.fragment.name === "ERC20Created");
    const erc20Addr = erc20Log.args[0];
    const erc20 = new ethers.Contract(erc20Addr, erc20Art.abi, signer);
    seedAddresses.erc20.push({ index: i, address: erc20Addr, symbol: erc20Symbol });

    await (await erc20.mint(walletAAddr, ethers.parseUnits("100", 18))).wait();

    console.log(`  Wallet A executes 'transfer()' of 50 ${erc20Symbol} to Wallet B...`);
    const transfer20Data = erc20.interface.encodeFunctionData("transfer", [walletBAddr, ethers.parseUnits("50", 18)]);
    await (await scwA.execute(erc20Addr, 0, transfer20Data)).wait();
    console.log(`  ✅ Wallet B Balance: ${ethers.formatUnits(await erc20.balanceOf(walletBAddr), 18)} ${erc20Symbol}`);
  }


  // ----------------------------------------------------------------------
  // 3. ERC721 FACTORY & SCW TRANSFER (x4)
  // ----------------------------------------------------------------------
  console.log(`\n--- 3. Testing ERC721 SCW Transfers (4 NFTs) ---`);
  const erc721FactoryArt = await hre.artifacts.readArtifact("Web3LabERC721Factory");
  const erc721Factory = new ethers.Contract(ADDRESSES.Web3LabERC721Factory, erc721FactoryArt.abi, signer);
  const erc721Art = await hre.artifacts.readArtifact("Web3LabERC721");

  const erc721Suffix = randStr();
  const erc721Symbol = `NFT-${erc721Suffix}`;
  const baseUri = `http://localhost:9000/web3lab-assets/erc721/metadata/`;
  
  console.log(`\n  Deploying NFT Collection: [${erc721Symbol}]`);
  const createErc721Tx = await erc721Factory.createNFT(`Lab Collection ${erc721Suffix}`, erc721Symbol, baseUri);
  const erc721Log = (await createErc721Tx.wait()).logs.find(l => l.fragment && l.fragment.name === "ERC721Created");
  const erc721Addr = erc721Log.args[0];
  const erc721 = new ethers.Contract(erc721Addr, erc721Art.abi, signer);
  seedAddresses.erc721.push({ address: erc721Addr, symbol: erc721Symbol });

  for (let i = 0; i <= 3; i++) {
    console.log(`\n  Minting NFT (Token ID: ${i}) directly to Wallet A...`);
    await (await erc721.mint(walletAAddr)).wait();

    console.log(`  Wallet A executes 'transferFrom()' of NFT(${i}) to Wallet B...`);
    const transfer721Data = erc721.interface.encodeFunctionData("transferFrom", [walletAAddr, walletBAddr, i]);
    await (await scwA.execute(erc721Addr, 0, transfer721Data)).wait();
    console.log(`  ✅ Wallet B NFT Balance: ${await erc721.balanceOf(walletBAddr)} (NFT-${i} Owner: ${await erc721.ownerOf(i)})`);
  }


  // ----------------------------------------------------------------------
  // 4. ERC1155 FACTORY & SCW TRANSFER (x4)
  // ----------------------------------------------------------------------
  console.log(`\n--- 4. Testing ERC1155 SCW Transfers (4 Items) ---`);
  const erc1155FactoryArt = await hre.artifacts.readArtifact("Web3LabERC1155Factory");
  const erc1155Factory = new ethers.Contract(ADDRESSES.Web3LabERC1155Factory, erc1155FactoryArt.abi, signer);
  const erc1155Art = await hre.artifacts.readArtifact("Web3LabERC1155");

  for (let i = 1; i <= 4; i++) {
    const erc1155Suffix = randStr();
    const erc1155Symbol = `ITM-${i}-${erc1155Suffix}`;
    const uri = `http://localhost:9000/web3lab-assets/erc1155/metadata/{id}.json`;
    
    console.log(`\n  Item ${i}: [${erc1155Symbol}]`);
    const createErc1155Tx = await erc1155Factory.createMultiToken(`Lab Items ${i} ${erc1155Suffix}`, erc1155Symbol, uri);
    const erc1155Log = (await createErc1155Tx.wait()).logs.find(l => l.fragment && l.fragment.name === "ERC1155Created");
    const erc1155Addr = erc1155Log.args[0];
    const erc1155 = new ethers.Contract(erc1155Addr, erc1155Art.abi, signer);
    seedAddresses.erc1155.push({ index: i, address: erc1155Addr, symbol: erc1155Symbol });

    await (await erc1155.mint(walletAAddr, i, 500, "0x")).wait();

    console.log(`  Wallet A executes 'safeTransferFrom()' of 200 Items to Wallet B...`);
    const transfer1155Data = erc1155.interface.encodeFunctionData("safeTransferFrom", [walletAAddr, walletBAddr, i, 200, "0x"]);
    await (await scwA.execute(erc1155Addr, 0, transfer1155Data)).wait();
    console.log(`  ✅ Wallet B Item Balance (ID ${i}): ${await erc1155.balanceOf(walletBAddr, i)}`);
  }

  // Save deployed seed addresses for Blockscout icon update
  fs.writeFileSync("seed-addresses.json", JSON.stringify(seedAddresses, null, 2));
  console.log("\n📝 Saved deployed token addresses to seed-addresses.json");
  console.log("\n🎉 All Smart Contract Wallet Execution integration test calls completed successfully!");

}

main().catch((error) => {
  console.error("Test failed:", error);
  process.exitCode = 1;
});
