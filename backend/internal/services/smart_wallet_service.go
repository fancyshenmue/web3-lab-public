package services

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/web3-lab/backend/internal/config"
	"github.com/web3-lab/backend/pkg/logs"
)

// SmartWalletService handles interactions relating to the ERC-4337 Smart Contract Wallet.
// This includes deriving deterministic CREATE2 addresses and mocking ZK Proof generation.
type SmartWalletService struct {
	rpcURL         string
	factoryAddress common.Address
	client         *ethclient.Client
	cfg            config.Web3Config
}

// NewSmartWalletService establishes a connection to the Web3 RPC and initializes the service.
func NewSmartWalletService(cfg config.Web3Config) (*SmartWalletService, error) {
	client, err := ethclient.Dial(cfg.GethRPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	return &SmartWalletService{
		rpcURL:         cfg.GethRPCUrl,
		factoryAddress: common.HexToAddress(cfg.AccountFactoryAddr),
		client:         client,
		cfg:            cfg,
	}, nil
}

// DeriveWalletAddress calls the `getAddress(address,uint256)` view function on the Factory contract.
// We use the Global Account ID (UUID) as the integer salt to ensure 1 unique SCW per human user.
func (s *SmartWalletService) DeriveWalletAddress(ctx context.Context, ownerAddr string, accountID uuid.UUID) (string, error) {
	// Convert UUID to a big.Int salt
	uuidBytes, _ := accountID.MarshalBinary() // 16 bytes
	salt := new(big.Int).SetBytes(uuidBytes)

	// Function selector for getAddress(address,uint256) -> 0x8cb84e18
	// Encoded arguments: address (padded to 32 bytes) + uint256 salt (padded to 32 bytes)
	methodSelector, _ := hex.DecodeString("8cb84e18")
	ownerCommon := common.HexToAddress(ownerAddr)

	paddedOwner := common.LeftPadBytes(ownerCommon.Bytes(), 32)
	paddedSalt := common.LeftPadBytes(salt.Bytes(), 32)

	var payload []byte
	payload = append(payload, methodSelector...)
	payload = append(payload, paddedOwner...)
	payload = append(payload, paddedSalt...)

	// Perform eth_call
	msg := ethereum.CallMsg{
		To:   &s.factoryAddress,
		Data: payload,
	}

	result, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		logs.FromContext(ctx).Error("Failed to call getAddress on factory", zap.Error(err))
		return "", fmt.Errorf("factory call failed: %w", err)
	}

	if len(result) != 32 {
		return "", fmt.Errorf("unexpected result length from factory: %d", len(result))
	}

	// Address is the last 20 bytes of the 32-byte returned word
	derivedAddress := common.BytesToAddress(result[12:]).Hex()
	return strings.ToLower(derivedAddress), nil
}

// DeriveWalletAddressByIdentity derives a wallet address using identity_id as the CREATE2 salt.
// Each identity gets its own mathematically distinct SCW address.
func (s *SmartWalletService) DeriveWalletAddressByIdentity(ctx context.Context, identityID uuid.UUID) (string, error) {
	// Use a fixed genesis account as the owner (index 0) for identity-based derivation
	ownerAddr, _ := s.GetDeterministicAccount(identityID)
	return s.DeriveWalletAddress(ctx, ownerAddr, identityID)
}

// GetInitCode returns the encoded factory payload (factory address + createAccount calldata) required by EntryPoint to deploy a new account.
func (s *SmartWalletService) GetInitCode(ownerAddr string, accountID uuid.UUID) []byte {
	uuidBytes, _ := accountID.MarshalBinary()
	salt := new(big.Int).SetBytes(uuidBytes)
	
	// Function selector for createAccount(address,uint256) -> 0x5fbfb9cf
	methodSelector, _ := hex.DecodeString("5fbfb9cf")
	ownerCommon := common.HexToAddress(ownerAddr)

	payload := make([]byte, 0, 20+4+32+32)
	payload = append(payload, s.factoryAddress.Bytes()...)
	payload = append(payload, methodSelector...)
	payload = append(payload, common.LeftPadBytes(ownerCommon.Bytes(), 32)...)
	payload = append(payload, common.LeftPadBytes(salt.Bytes(), 32)...)

	return payload
}

// GetDeterministicAccount derives one of the 10 Genesis EOAs based on the AccountID UUID.
// It returns the owner address hex and the corresponding private key hex.
func (s *SmartWalletService) GetDeterministicAccount(accountID uuid.UUID) (string, string) {
	// 10 Genesis EOAs mapped to Web2 identity are loaded from config.
	genesisAddresses := s.cfg.GenesisAddresses
	genesisKeys := s.cfg.GenesisKeys

	if len(genesisAddresses) == 0 || len(genesisKeys) == 0 {
		return "", ""
	}

	// Simple hash slice for modulo uniqueness
	var sum uint32
	for _, b := range accountID {
		sum += uint32(b)
	}
	index := sum % uint32(len(genesisAddresses))

	return genesisAddresses[index], genesisKeys[index]
}

// GenerateZKProof is a mock implementation of the TEE / Prover microservice call.
// It generates a perfectly valid ECDSA signature mapping to Hardhat Test Key 1, passing normal SimpleAccount validation.
func (s *SmartWalletService) GenerateZKProof(ctx context.Context, accountID uuid.UUID, userOpHash string) (string, error) {
	logs.FromContext(ctx).Info("Generating Native Protocol ECDSA Signature for UserOperation",
		zap.String("account_id", accountID.String()),
		zap.String("userop_hash", userOpHash),
	)

	// Resolve the deterministic proxy key for the Web2 User
	_, privKeyHex := s.GetDeterministicAccount(accountID)
	mockPrivKey, _ := crypto.HexToECDSA(privKeyHex)
	
	hashBytes := common.FromHex(userOpHash)
	
	// ERC-4337 v0.8.0 SimpleAccount no longer computes toEthSignedMessageHash internally.
	// The provided userOpHash is a correctly EIP-712-typed hash evaluated from EntryPoint.
	sig, err := crypto.Sign(hashBytes, mockPrivKey)
	if err != nil {
		return "", err
	}
	sig[64] += 27 // v offset
	
	return "0x" + common.Bytes2Hex(sig), nil
}
