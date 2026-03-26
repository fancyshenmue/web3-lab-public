package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// AuthService provides cryptographic signature verification for EOA wallets.
type AuthService struct{}

func NewAuthService() *AuthService { return &AuthService{} }

// VerifySignature checks that the signature was produced by the given address
// for the specified message (EIP-191 personal_sign).
func (s *AuthService) VerifySignature(address, signature, message string) (bool, error) {
	// Decode signature
	sigBytes, err := hexutil.Decode(signature)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}
	if len(sigBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	// Adjust v value (MetaMask returns 27/28, go-ethereum expects 0/1)
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	// Hash the message with EIP-191 prefix
	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixedMsg))

	// Recover public key
	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false, fmt.Errorf("recover public key: %w", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	expected := common.HexToAddress(address)

	return strings.EqualFold(recoveredAddr.Hex(), expected.Hex()), nil
}

// RecoverAddress recovers the signer address from a signed message (EIP-191).
func (s *AuthService) RecoverAddress(signature, message string) (string, error) {
	sigBytes, err := hexutil.Decode(signature)
	if err != nil {
		return "", fmt.Errorf("decode signature: %w", err)
	}
	if len(sigBytes) != 65 {
		return "", fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixedMsg))

	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return "", fmt.Errorf("recover public key: %w", err)
	}

	return crypto.PubkeyToAddress(*pubKey).Hex(), nil
}

// VerifyEIP712Signature checks that the EIP-712 structured data signature
// was produced by the given address.
func (s *AuthService) VerifyEIP712Signature(address, signature, messageJSON string) (bool, error) {
	recoveredAddr, err := s.RecoverEIP712Address(signature, messageJSON)
	if err != nil {
		return false, err
	}

	expected := common.HexToAddress(address)
	return strings.EqualFold(recoveredAddr, expected.Hex()), nil
}

// RecoverEIP712Address recovers the signer address from an EIP-712 signed typed data JSON.
func (s *AuthService) RecoverEIP712Address(signature, messageJSON string) (string, error) {
	// Parse the JSON into standard typed data
	var typedData apitypes.TypedData
	if err := json.Unmarshal([]byte(messageJSON), &typedData); err != nil {
		return "", fmt.Errorf("unmarshal EIP-712 typed data: %w", err)
	}

	// Compute EIP-712 compliant hash (HashStruct + DomainSeparator + Keccak256)
	hashBytes, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return "", fmt.Errorf("compute EIP-712 hash: %w", err)
	}

	sigBytes, err := hexutil.Decode(signature)
	if err != nil {
		return "", fmt.Errorf("decode signature: %w", err)
	}
	if len(sigBytes) != 65 {
		return "", fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	// Adjust v value (MetaMask returns 27/28, go-ethereum expects 0/1)
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	pubKey, err := crypto.SigToPub(hashBytes, sigBytes)
	if err != nil {
		return "", fmt.Errorf("recover public key: %w", err)
	}

	return crypto.PubkeyToAddress(*pubKey).Hex(), nil
}
