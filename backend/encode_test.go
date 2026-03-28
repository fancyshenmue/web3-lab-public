package main

import (
"fmt"
"math/big"
"strings"
"github.com/ethereum/go-ethereum/accounts/abi"
"github.com/ethereum/go-ethereum/common"
"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	uriString := "http://minio.web3.svc.cluster.local:9000/web3lab-assets/erc721/0x2340e2c1fd4370ff362e6567818c7330e3d9cb63/metadata/"
	
	// Manual encoding
	var manualData []byte
	methodSelector := crypto.Keccak256Hash([]byte("setBaseURI(string)")).Bytes()[:4]
	manualData = append(manualData, methodSelector...)
	offset := big.NewInt(32)
	length := big.NewInt(int64(len(uriString)))
	strPadding := ((len(uriString) + 31) / 32) * 32
	paddedStr := make([]byte, strPadding)
	copy(paddedStr, uriString)
	manualData = append(manualData, common.LeftPadBytes(offset.Bytes(), 32)...)
	manualData = append(manualData, common.LeftPadBytes(length.Bytes(), 32)...)
	manualData = append(manualData, paddedStr...)

	// Official ABI encoding
	parsedABI, _ := abi.JSON(strings.NewReader(`[{"inputs":[{"internalType":"string","name":"newBaseURI","type":"string"}],"name":"setBaseURI","outputs":[],"stateMutability":"nonpayable","type":"function"}]`))
	officialData, _ := parsedABI.Pack("setBaseURI", uriString)

	fmt.Printf("Manual: %x\n", manualData)
	fmt.Printf("Officl: %x\n", officialData)
    fmt.Printf("Match:  %v\n", string(manualData) == string(officialData))
}
