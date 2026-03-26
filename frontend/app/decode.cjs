const { ethers } = require('ethers');
const data = "0x00000000000000000000000000000000000000000000000000000000000000070000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008403dee4c5000000000000000000000000e127d08ae30cf7bcce946edd424a957cdc3b3a580000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000";

const decoded = ethers.AbiCoder.defaultAbiCoder().decode(["uint256", "bytes"], data);
console.log("Nonce:", decoded[0].toString());

const revertReasonBytes = decoded[1];
console.log("Revert Reason Bytes:", revertReasonBytes);

if (revertReasonBytes.startsWith('0x08c379a0')) {
    // Error(string)
    const reasonStr = ethers.AbiCoder.defaultAbiCoder().decode(["string"], '0x' + revertReasonBytes.slice(10));
    console.log("String Revert:", reasonStr[0]);
} else if (revertReasonBytes.startsWith('0x4e487b71')) {
    // Panic(uint256)
    const panicCode = ethers.AbiCoder.defaultAbiCoder().decode(["uint256"], '0x' + revertReasonBytes.slice(10));
    console.log("Panic Revert:", panicCode[0].toString());
} else {
    console.log("Custom Error bytes. Selector:", revertReasonBytes.slice(0, 10));
}
