package services

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/web3-lab/backend/internal/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/web3-lab/backend/pkg/logs"
	"go.uber.org/zap"
)

// UserOperation represents an EIP-4337 transaction syntax mapping (frontend payload).
type UserOperation struct {
	Sender               string `json:"sender"`
	Nonce                string `json:"nonce"`
	InitCode             string `json:"initCode"`
	CallData             string `json:"callData"`
	CallGasLimit         string `json:"callGasLimit"`
	VerificationGasLimit string `json:"verificationGasLimit"`
	PreVerificationGas   string `json:"preVerificationGas"`
	MaxFeePerGas         string `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
	PaymasterAndData     string `json:"paymasterAndData"`
	Signature            string `json:"signature"`
}

// PackedUserOperation is the strictly typed definition expected by the EntryPoint ABI parameter pack routines.
type PackedUserOperation struct {
	Sender             common.Address
	Nonce              *big.Int
	InitCode           []byte
	CallData           []byte
	AccountGasLimits   [32]byte
	PreVerificationGas *big.Int
	GasFees            [32]byte
	PaymasterAndData   []byte
	Signature          []byte
}

const EntryPointABIStr = `[{"inputs":[{"components":[{"internalType":"address","name":"sender","type":"address"},{"internalType":"uint256","name":"nonce","type":"uint256"},{"internalType":"bytes","name":"initCode","type":"bytes"},{"internalType":"bytes","name":"callData","type":"bytes"},{"internalType":"bytes32","name":"accountGasLimits","type":"bytes32"},{"internalType":"uint256","name":"preVerificationGas","type":"uint256"},{"internalType":"bytes32","name":"gasFees","type":"bytes32"},{"internalType":"bytes","name":"paymasterAndData","type":"bytes"},{"internalType":"bytes","name":"signature","type":"bytes"}],"internalType":"struct PackedUserOperation[]","name":"ops","type":"tuple[]"},{"internalType":"address payable","name":"beneficiary","type":"address"}],"name":"handleOps","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

// BundlerService handles the construction, paymaster sponsorship, and submission of UserOps.
type BundlerService struct {
	bundlerRPCURL    string
	paymasterPrivKey *ecdsa.PrivateKey
	paymasterAddress string
	client           *ethclient.Client
	cfg              config.Web3Config
}

// NewBundlerService initializes the Bundler client and local Paymaster signer.
func NewBundlerService(cfg config.Web3Config) (*BundlerService, error) {
	pk, err := crypto.HexToECDSA(cfg.PaymasterPriv)
	if err != nil {
		return nil, fmt.Errorf("invalid paymaster private key: %w", err)
	}

	client, err := ethclient.Dial(cfg.GethRPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to geth provider for bundler: %w", err)
	}

	return &BundlerService{
		bundlerRPCURL:    cfg.GethRPCUrl,
		paymasterPrivKey: pk,
		paymasterAddress: cfg.PaymasterAddr,
		client:           client,
		cfg:              cfg,
	}, nil
}

// GetClient exposes the underlying standard Geth client
func (s *BundlerService) GetClient() *ethclient.Client {
	return s.client
}

// BuildUserOperation creates a raw UserOp from a high-level intent.
func (s *BundlerService) BuildUserOperation(ctx context.Context, senderAddr, callDataHex string, initCode string) (*UserOperation, error) {
	logs.FromContext(ctx).Info("Building UserOperation with dynamic execution calldata", zap.String("callData", callDataHex))

	// Fetch dynamic Nonce from EntryPoint
	// Function selector for getNonce(address,uint192) -> 0x35567e1a
	methodSelector, _ := hex.DecodeString("35567e1a")
	senderCommon := common.HexToAddress(senderAddr)
	paddedSender := common.LeftPadBytes(senderCommon.Bytes(), 32)
	paddedKey := common.LeftPadBytes([]byte{0}, 32) // Key 0

	var payload []byte
	payload = append(payload, methodSelector...)
	payload = append(payload, paddedSender...)
	payload = append(payload, paddedKey...)

	epAddr := common.HexToAddress(s.cfg.EntryPointAddr)
	msg := ethereum.CallMsg{
		To:   &epAddr,
		Data: payload,
	}

	nonceHex := "0x0"
	result, err := s.client.CallContract(ctx, msg, nil)
	if err == nil && len(result) >= 32 {
		nonceVal := new(big.Int).SetBytes(result)
		nonceHex = "0x" + fmt.Sprintf("%x", nonceVal)
	}

	// Mocked basic UserOp structure
	return &UserOperation{
		Sender:               senderAddr,
		Nonce:                nonceHex, // Dynamically computed
		InitCode:             initCode, // Computed explicitly to allow auto-deployment
		CallData:             callDataHex, // Nested SimpleAccount execute payload
		CallGasLimit:         "0x2DC6C0",    // 3,000,000 gas (ERC721/1155 CREATE needs more gas)
		VerificationGasLimit: "0xF4240",    // 1,000,000 gas (supports initial CREATE2 factory deployment)
		PreVerificationGas:   "0x5208",     // 21000
		MaxFeePerGas:         "0x3B9ACA00", // 1 Gwei
		MaxPriorityFeePerGas: "0x3B9ACA00",
		PaymasterAndData:     "0x",
		Signature:            "0x",
	}, nil
}

// HashUserOp returns the keccak256 hash of the UserOperation, required for signing.
func (s *BundlerService) HashUserOp(ctx context.Context, userOp *UserOperation) ([]byte, error) {
	sender := common.HexToAddress(userOp.Sender)
	nonce := parseBigInt(userOp.Nonce)
	initCode := decodeHex(userOp.InitCode)
	callData := decodeHex(userOp.CallData)
	accountGasLimits := packBytes32(userOp.VerificationGasLimit, userOp.CallGasLimit)
	preVerifGas := parseBigInt(userOp.PreVerificationGas)
	gasFees := packBytes32(userOp.MaxPriorityFeePerGas, userOp.MaxFeePerGas)
	paymasterAndData := decodeHex(userOp.PaymasterAndData)

	UserOpTupleABI := `[{"inputs":[{"components":[{"internalType":"address","name":"sender","type":"address"},{"internalType":"uint256","name":"nonce","type":"uint256"},{"internalType":"bytes","name":"initCode","type":"bytes"},{"internalType":"bytes","name":"callData","type":"bytes"},{"internalType":"bytes32","name":"accountGasLimits","type":"bytes32"},{"internalType":"uint256","name":"preVerificationGas","type":"uint256"},{"internalType":"bytes32","name":"gasFees","type":"bytes32"},{"internalType":"bytes","name":"paymasterAndData","type":"bytes"},{"internalType":"bytes","name":"signature","type":"bytes"}],"internalType":"struct PackedUserOperation","name":"userOp","type":"tuple"}],"name":"getUserOpHash","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"}]`

	parsedABI, err := abi.JSON(strings.NewReader(UserOpTupleABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	type PackedUserOperation struct {
		Sender             common.Address
		Nonce              *big.Int
		InitCode           []byte
		CallData           []byte
		AccountGasLimits   [32]byte
		PreVerificationGas *big.Int
		GasFees            [32]byte
		PaymasterAndData   []byte
		Signature          []byte
	}

	op := PackedUserOperation{
		Sender:             sender,
		Nonce:              nonce,
		InitCode:           initCode,
		CallData:           callData,
		AccountGasLimits:   accountGasLimits,
		PreVerificationGas: preVerifGas,
		GasFees:            gasFees,
		PaymasterAndData:   paymasterAndData,
		Signature:          []byte{},
	}

	encoded, err := parsedABI.Pack("getUserOpHash", op)
	if err != nil {
		return nil, fmt.Errorf("failed to encode call: %w", err)
	}

	epAddr := common.HexToAddress(s.cfg.EntryPointAddr)
	msg := ethereum.CallMsg{
		To:   &epAddr,
		Data: encoded,
	}

	result, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed calling getUserOpHash: %w", err)
	}

	if len(result) < 32 {
		return nil, fmt.Errorf("unexpected return length from getUserOpHash: %d", len(result))
	}

	return result[:32], nil
}

// EncodeExecutionCall constructs the bytes payload for SimpleAccount.execute(target, value, innerCallData)
func (s *BundlerService) EncodeExecutionCall(action, tokenType, toStr, amountStr, tokenIDStr, sender, tokenAddrStr, name, symbol, decimalsStr, initialSupplyStr string) (string, error) {
	// Parse basic parameters
	tokenID := big.NewInt(0)
	if tokenIDStr != "" {
		tokenID.SetString(tokenIDStr, 10)
	}

	amount := big.NewInt(0)
	if amountStr != "" {
		amount.SetString(amountStr, 10)
	}

	toAddr := common.HexToAddress(toStr)
	if toStr == "" {
		toAddr = common.HexToAddress(sender) // Default to self for minting
	}

	targetAddress := common.Address{}

	if action == "deploy_contract" {
		switch tokenType {
		case "ERC20":
			targetAddress = common.HexToAddress(s.cfg.ERC20FactoryAddr)
		case "ERC721":
			targetAddress = common.HexToAddress(s.cfg.ERC721FactoryAddr)
		case "ERC1155":
			targetAddress = common.HexToAddress(s.cfg.ERC1155FactoryAddr)
		default:
			return "", fmt.Errorf("unsupported token type for deploy: %s", tokenType)
		}
	} else {
		if tokenAddrStr == "" {
			return "", fmt.Errorf("token_address is legally required for mint and transfer actions")
		}
		targetAddress = common.HexToAddress(tokenAddrStr)
	}

	var innerCallData []byte

	if action == "mint" {
		if tokenType == "ERC20" {
			// mint(address,uint256)
			amountWei := new(big.Int).Mul(amount, big.NewInt(1e18))
			methodSelector := crypto.Keccak256Hash([]byte("mint(address,uint256)")).Bytes()[:4]
			innerCallData = append(innerCallData, methodSelector...)
			innerCallData = append(innerCallData, common.LeftPadBytes(toAddr.Bytes(), 32)...)
			innerCallData = append(innerCallData, common.LeftPadBytes(amountWei.Bytes(), 32)...)
		} else if tokenType == "ERC721" {
			// mint(address)
			methodSelector := crypto.Keccak256Hash([]byte("mint(address)")).Bytes()[:4]
			innerCallData = append(innerCallData, methodSelector...)
			innerCallData = append(innerCallData, common.LeftPadBytes(toAddr.Bytes(), 32)...)
		} else if tokenType == "ERC1155" {
			// mint(address,uint256,uint256,bytes)
			methodSelector := crypto.Keccak256Hash([]byte("mint(address,uint256,uint256,bytes)")).Bytes()[:4]
			innerCallData = append(innerCallData, methodSelector...)
			innerCallData = append(innerCallData, common.LeftPadBytes(toAddr.Bytes(), 32)...)
			innerCallData = append(innerCallData, common.LeftPadBytes(tokenID.Bytes(), 32)...)
			innerCallData = append(innerCallData, common.LeftPadBytes(amount.Bytes(), 32)...)
			// bytes data (offset=128, length=0)
			innerCallData = append(innerCallData, common.LeftPadBytes(big.NewInt(128).Bytes(), 32)...)
			innerCallData = append(innerCallData, common.LeftPadBytes(big.NewInt(0).Bytes(), 32)...)
		}
	} else if action == "transfer" {
		if tokenType == "ERC20" {
			// transfer(address,uint256)
			amountWei := new(big.Int).Mul(amount, big.NewInt(1e18))
			methodSelector := crypto.Keccak256Hash([]byte("transfer(address,uint256)")).Bytes()[:4]
			innerCallData = append(innerCallData, methodSelector...)
			innerCallData = append(innerCallData, common.LeftPadBytes(toAddr.Bytes(), 32)...)
			innerCallData = append(innerCallData, common.LeftPadBytes(amountWei.Bytes(), 32)...)
		} else if tokenType == "ERC721" {
			// transferFrom(address,address,uint256)
			methodSelector := crypto.Keccak256Hash([]byte("transferFrom(address,address,uint256)")).Bytes()[:4]
			innerCallData = append(innerCallData, methodSelector...)
			innerCallData = append(innerCallData, common.LeftPadBytes(common.HexToAddress(sender).Bytes(), 32)...) // from
			innerCallData = append(innerCallData, common.LeftPadBytes(toAddr.Bytes(), 32)...) // to
			innerCallData = append(innerCallData, common.LeftPadBytes(tokenID.Bytes(), 32)...) // ID
		} else if tokenType == "ERC1155" {
			// safeTransferFrom(address,address,uint256,uint256,bytes)
			methodSelector := crypto.Keccak256Hash([]byte("safeTransferFrom(address,address,uint256,uint256,bytes)")).Bytes()[:4]
			innerCallData = append(innerCallData, methodSelector...)
			innerCallData = append(innerCallData, common.LeftPadBytes(common.HexToAddress(sender).Bytes(), 32)...)
			innerCallData = append(innerCallData, common.LeftPadBytes(toAddr.Bytes(), 32)...)
			innerCallData = append(innerCallData, common.LeftPadBytes(tokenID.Bytes(), 32)...) // ID
			innerCallData = append(innerCallData, common.LeftPadBytes(amount.Bytes(), 32)...) // Amount
			// bytes data (offset=160, length=0)
			innerCallData = append(innerCallData, common.LeftPadBytes(big.NewInt(160).Bytes(), 32)...)
			innerCallData = append(innerCallData, common.LeftPadBytes(big.NewInt(0).Bytes(), 32)...)
		}
	} else if action == "deploy_contract" {
		if name == "" { name = "Web3Lab Token" }
		if symbol == "" { symbol = "W3L" }
		
		encodeStringParamsTriple := func(s1, s2, s3 string) []byte {
			var params []byte
			offset1 := big.NewInt(96)
			len1Padding := ((len(s1) + 31) / 32) * 32
			offset2 := big.NewInt(int64(96 + 32 + len1Padding))
			len2Padding := ((len(s2) + 31) / 32) * 32
			offset3 := big.NewInt(int64(96 + 32 + len1Padding + 32 + len2Padding))
			
			params = append(params, common.LeftPadBytes(offset1.Bytes(), 32)...)
			params = append(params, common.LeftPadBytes(offset2.Bytes(), 32)...)
			params = append(params, common.LeftPadBytes(offset3.Bytes(), 32)...)
			
			params = append(params, common.LeftPadBytes(big.NewInt(int64(len(s1))).Bytes(), 32)...)
			s1Padded := make([]byte, len1Padding)
			copy(s1Padded, []byte(s1))
			params = append(params, s1Padded...)
			
			params = append(params, common.LeftPadBytes(big.NewInt(int64(len(s2))).Bytes(), 32)...)
			s2Padded := make([]byte, len2Padding)
			copy(s2Padded, []byte(s2))
			params = append(params, s2Padded...)
			
			len3Padding := ((len(s3) + 31) / 32) * 32
			params = append(params, common.LeftPadBytes(big.NewInt(int64(len(s3))).Bytes(), 32)...)
			s3Padded := make([]byte, len3Padding)
			copy(s3Padded, []byte(s3))
			params = append(params, s3Padded...)
			
			return params
		}

		if tokenType == "ERC20" {
			decimals := uint8(18)
			if decimalsStr != "" {
				if d, err := strconv.ParseUint(decimalsStr, 10, 8); err == nil {
					decimals = uint8(d)
				}
			}

			initialSupply := big.NewInt(0)
			if initialSupplyStr != "" {
				initialSupply.SetString(initialSupplyStr, 10)
			}

			encodeStringParamsWithUint8AndUint256 := func(s1, s2 string, val1 uint8, val2 *big.Int) []byte {
				var params []byte
				offset1 := big.NewInt(128)
				len1Padding := ((len(s1) + 31) / 32) * 32
				offset2 := big.NewInt(int64(128 + 32 + len1Padding))
				
				params = append(params, common.LeftPadBytes(offset1.Bytes(), 32)...)
				params = append(params, common.LeftPadBytes(offset2.Bytes(), 32)...)
				params = append(params, common.LeftPadBytes(big.NewInt(int64(val1)).Bytes(), 32)...)
				params = append(params, common.LeftPadBytes(val2.Bytes(), 32)...)
				
				params = append(params, common.LeftPadBytes(big.NewInt(int64(len(s1))).Bytes(), 32)...)
				s1Padded := make([]byte, len1Padding)
				copy(s1Padded, []byte(s1))
				params = append(params, s1Padded...)
				
				len2Padding := ((len(s2) + 31) / 32) * 32
				params = append(params, common.LeftPadBytes(big.NewInt(int64(len(s2))).Bytes(), 32)...)
				s2Padded := make([]byte, len2Padding)
				copy(s2Padded, []byte(s2))
				params = append(params, s2Padded...)
				
				return params
			}

			methodSelector := crypto.Keccak256Hash([]byte("createToken(string,string,uint8,uint256)")).Bytes()[:4]
			innerCallData = append(innerCallData, methodSelector...)
			innerCallData = append(innerCallData, encodeStringParamsWithUint8AndUint256(name, symbol, decimals, initialSupply)...)
		} else if tokenType == "ERC721" {
			methodSelector := crypto.Keccak256Hash([]byte("createNFT(string,string,string)")).Bytes()[:4]
			innerCallData = append(innerCallData, methodSelector...)
			innerCallData = append(innerCallData, encodeStringParamsTriple(name, symbol, "https://api.web3lab.com/nft/")...)
		} else if tokenType == "ERC1155" {
			methodSelector := crypto.Keccak256Hash([]byte("createMultiToken(string,string,string)")).Bytes()[:4]
			innerCallData = append(innerCallData, methodSelector...)
			innerCallData = append(innerCallData, encodeStringParamsTriple(name, symbol, "https://api.web3lab.com/erc1155/")...)
		}
	} else {
		return "", fmt.Errorf("unsupported action: %s", action)
	}

	execSelector := crypto.Keccak256Hash([]byte("execute(address,uint256,bytes)")).Bytes()[:4]
	var payload []byte
	payload = append(payload, execSelector...)
	payload = append(payload, common.LeftPadBytes(targetAddress.Bytes(), 32)...)
	payload = append(payload, common.LeftPadBytes(big.NewInt(0).Bytes(), 32)...) // value=0
	
	payload = append(payload, common.LeftPadBytes(big.NewInt(96).Bytes(), 32)...)
	payload = append(payload, common.LeftPadBytes(big.NewInt(int64(len(innerCallData))).Bytes(), 32)...)
	
	paddedLen := ((len(innerCallData) + 31) / 32) * 32
	paddedInnerData := make([]byte, paddedLen)
	copy(paddedInnerData, innerCallData)
	payload = append(payload, paddedInnerData...)

	return "0x" + common.Bytes2Hex(payload), nil
}

// SignPaymasterData signs the UserOp using the backend's designated Paymaster key.
func (s *BundlerService) SignPaymasterData(ctx context.Context, userOp *UserOperation) error {
	logs.FromContext(ctx).Info("Signing UserOperation natively with backend Paymaster key")
	
	pmVerifGas := "000000000000000000000000000f4240" // 32 hex chars: 1,000,000 gas
	pmPostOpGas := "00000000000000000000000000000000" // 32 hex chars: 0 gas

	sender := common.HexToAddress(userOp.Sender)
	nonce := parseBigInt(userOp.Nonce)
	initCode := decodeHex(userOp.InitCode)
	callData := decodeHex(userOp.CallData)
	accountGasLimits := packBytes32(userOp.VerificationGasLimit, userOp.CallGasLimit)
	preVerifGas := parseBigInt(userOp.PreVerificationGas)
	gasFees := packBytes32(userOp.MaxPriorityFeePerGas, userOp.MaxFeePerGas)
	
	chainID, _ := s.client.ChainID(ctx)
	if chainID == nil {
		chainID = big.NewInt(31337) // Fallback for hardhat/local network
	}

	PaymasterHashABI := `[{"inputs":[{"internalType":"address","name":"sender","type":"address"},{"internalType":"uint256","name":"nonce","type":"uint256"},{"internalType":"bytes32","name":"initCodeHash","type":"bytes32"},{"internalType":"bytes32","name":"callDataHash","type":"bytes32"},{"internalType":"bytes32","name":"accountGasLimits","type":"bytes32"},{"internalType":"uint256","name":"preVerificationGas","type":"uint256"},{"internalType":"bytes32","name":"gasFees","type":"bytes32"},{"internalType":"uint256","name":"chainid","type":"uint256"},{"internalType":"address","name":"paymaster","type":"address"}],"name":"getHash","outputs":[],"type":"function"}]`

	parsedABI, _ := abi.JSON(strings.NewReader(PaymasterHashABI))
	
	initCodeHash := crypto.Keccak256Hash(initCode)
	callDataHash := crypto.Keccak256Hash(callData)

	encoded, _ := parsedABI.Pack("getHash", sender, nonce, initCodeHash, callDataHash, accountGasLimits, preVerifGas, gasFees, chainID, common.HexToAddress(s.paymasterAddress))
	
	hash := crypto.Keccak256Hash(encoded[4:])

	// Append typical Ethereum signed message prefix
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(hash.Bytes()))
	signedHash := crypto.Keccak256Hash(append([]byte(prefix), hash.Bytes()...))

	sig, err := crypto.Sign(signedHash.Bytes(), s.paymasterPrivKey)
	if err != nil {
		return err
	}
	sig[64] += 27 // Convert recovery ID to 27/28

	sigHex := hexutil.Encode(sig)[2:]

	paymasterAddr := strings.TrimPrefix(s.paymasterAddress, "0x")
	userOp.PaymasterAndData = fmt.Sprintf("0x%s%s%s%s", paymasterAddr, pmVerifGas, pmPostOpGas, sigHex)

	return nil
}

// decodeHex cleanly parses 0x strings handling empty returns
func decodeHex(in string) []byte {
	b, _ := hexutil.Decode(in)
	return b
}

func parseBigInt(in string) *big.Int {
	if strings.HasPrefix(in, "0x") {
		v, _ := new(big.Int).SetString(in[2:], 16)
		return v
	}
	v, _ := new(big.Int).SetString(in, 10)
	return v
}

// packBytes32 Combines two UINT128 limits into a single bytes32 parameter
func packBytes32(a, b string) [32]byte {
	var result [32]byte
	aInt := parseBigInt(a)
	bInt := parseBigInt(b)
	
	if aInt != nil {
		copy(result[0:16], common.LeftPadBytes(aInt.Bytes(), 16))
	}
	if bInt != nil {
		copy(result[16:32], common.LeftPadBytes(bInt.Bytes(), 16))
	}
	return result
}

// SubmitToBundler forwards the fully constructed UserOp to the remote Bundler node.
func (s *BundlerService) SubmitToBundler(ctx context.Context, userOp *UserOperation) (string, error) {
	logs.FromContext(ctx).Info("Submitting UserOperation to EntryPoint via local EOA Bundler execution")
	
	parsedABI, err := abi.JSON(strings.NewReader(EntryPointABIStr))
	if err != nil {
		return "", fmt.Errorf("failed to parse EntryPoint ABI: %w", err)
	}

	// 1. Convert JSON UserOp to strongly typed Packable representation
	packedOp := PackedUserOperation{
		Sender:             common.HexToAddress(userOp.Sender),
		Nonce:              parseBigInt(userOp.Nonce),
		InitCode:           decodeHex(userOp.InitCode),
		CallData:           decodeHex(userOp.CallData),
		AccountGasLimits:   packBytes32(userOp.VerificationGasLimit, userOp.CallGasLimit),
		PreVerificationGas: parseBigInt(userOp.PreVerificationGas),
		GasFees:            packBytes32(userOp.MaxPriorityFeePerGas, userOp.MaxFeePerGas),
		PaymasterAndData:   decodeHex(userOp.PaymasterAndData),
		Signature:          decodeHex(userOp.Signature),
	}

	beneficiary := crypto.PubkeyToAddress(s.paymasterPrivKey.PublicKey)

	// 2. Pack the data to call handleOps(ops[], beneficiary)
	callData, err := parsedABI.Pack("handleOps", []PackedUserOperation{packedOp}, beneficiary)
	if err != nil {
		return "", fmt.Errorf("failed to ABI pack handleOps: %w", err)
	}

	entryPointAddress := common.HexToAddress(s.cfg.EntryPointAddr)
	
	// 3. Obtain EOA State (Nonce, Gas Price)
	chainID, err := s.client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get chain ID: %w", err)
	}

	bundlerNonce, err := s.client.PendingNonceAt(ctx, beneficiary)
	if err != nil {
		return "", fmt.Errorf("failed to get bundler nonce: %w", err)
	}

	gasPrice, err := s.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to suggest gas price: %w", err)
	}

	// 4. Build and Broadcast the native transaction mapping to handleOps
	// We'll hardcode an adequately high execution gas limit for the smart wallet overhead wrapper.
	tx := types.NewTransaction(bundlerNonce, entryPointAddress, big.NewInt(0), 10000000, gasPrice, callData)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), s.paymasterPrivKey)
	if err != nil {
		return "", fmt.Errorf("failed signing tx: %w", err)
	}

	err = s.client.SendTransaction(ctx, signedTx)
	if err != nil {
		logs.FromContext(ctx).Error("Failed to broadcast bundle transaction", zap.Error(err))
		return "", fmt.Errorf("broadcast failed: %w", err)
	}

	txHash := signedTx.Hash().Hex()
	logs.FromContext(ctx).Info("Successfully submitted ERC-4337 UserOperation via EOA Bundler", zap.String("txHash", txHash))
	return txHash, nil
}
