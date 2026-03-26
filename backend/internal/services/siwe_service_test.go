package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSIWEService creates a minimal SIWEService for testing pure functions.
func newTestSIWEService() *SIWEService {
	hydra := &HydraClientService{
		publicURL: "http://hydra-public.web3.svc.cluster.local:4444",
	}
	return &SIWEService{
		hydra:            hydra,
		defaultDomain:    "app.web3-local-dev.com",
		defaultURI:       "https://app.web3-local-dev.com",
		defaultStatement: "Sign in to Web3 Lab",
		defaultChainID:   1337,
		defaultVersion:   "1",
	}
}

// --- SIWE Message Generation ---

func TestGenerateSIWEMessage(t *testing.T) {
	s := newTestSIWEService()
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	expires := now.Add(5 * time.Minute)

	msg := s.generateSIWEMessage(
		"app.web3-local-dev.com",
		"0x1234567890abcdef1234567890abcdef12345678",
		"Sign in to Web3 Lab",
		"https://app.web3-local-dev.com",
		"1",
		1337,
		"abc123nonce",
		now,
		expires,
	)

	// EIP-4361 format assertions
	assert.Contains(t, msg, "app.web3-local-dev.com wants you to sign in with your Ethereum account:")
	assert.Contains(t, msg, "0x1234567890abcdef1234567890abcdef12345678")
	assert.Contains(t, msg, "Sign in to Web3 Lab")
	assert.Contains(t, msg, "URI: https://app.web3-local-dev.com")
	assert.Contains(t, msg, "Version: 1")
	assert.Contains(t, msg, "Chain ID: 1337")
	assert.Contains(t, msg, "Nonce: abc123nonce")
	assert.Contains(t, msg, "Issued At: 2026-03-24T12:00:00Z")
	assert.Contains(t, msg, "Expiration Time: 2026-03-24T12:05:00Z")

	// Verify line order (EIP-4361 ABNF)
	lines := strings.Split(msg, "\n")
	assert.True(t, strings.HasSuffix(lines[0], "wants you to sign in with your Ethereum account:"), "line 0: domain header")
	assert.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", strings.TrimSpace(lines[1]), "line 1: address")
	assert.Equal(t, "", strings.TrimSpace(lines[2]), "line 2: empty separator")
	assert.Equal(t, "Sign in to Web3 Lab", strings.TrimSpace(lines[3]), "line 3: statement")
}

// --- SIWE Message Parsing ---

func TestParseSIWEMessage(t *testing.T) {
	s := newTestSIWEService()

	t.Run("valid message", func(t *testing.T) {
		msg := `app.web3-local-dev.com wants you to sign in with your Ethereum account:
0xAbC1234567890abcdef1234567890abcdef12345

Sign in to Web3 Lab

URI: https://app.web3-local-dev.com
Version: 1
Chain ID: 1337
Nonce: testNonce42
Issued At: 2026-03-24T12:00:00Z
Expiration Time: 2026-03-24T12:05:00Z`

		addr, nonce, err := s.parseSIWEMessage(msg)
		require.NoError(t, err)
		assert.Equal(t, "0xAbC1234567890abcdef1234567890abcdef12345", addr)
		assert.Equal(t, "testNonce42", nonce)
	})

	t.Run("missing address", func(t *testing.T) {
		msg := `app.web3-local-dev.com wants you to sign in with your Ethereum account:
not-an-address

Nonce: testNonce42`

		_, _, err := s.parseSIWEMessage(msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "address not found")
	})

	t.Run("missing nonce", func(t *testing.T) {
		msg := `app.web3-local-dev.com wants you to sign in with your Ethereum account:
0x1234567890abcdef1234567890abcdef12345678

Sign in to Web3 Lab

URI: https://app.web3-local-dev.com
Version: 1`

		_, _, err := s.parseSIWEMessage(msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nonce not found")
	})
}

// --- EIP-712 Message Generation ---

func TestGenerateEIP712Message(t *testing.T) {
	s := newTestSIWEService()
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	expires := now.Add(5 * time.Minute)

	msg, err := s.generateEIP712Message(
		"app.web3-local-dev.com",
		"0x1234567890abcdef1234567890abcdef12345678",
		"Sign in to Web3 Lab",
		"1",
		1337,
		"abc123nonce",
		now,
		expires,
	)
	require.NoError(t, err)

	// Must be valid JSON
	var typedData map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(msg), &typedData))

	// Check structure
	assert.Equal(t, "AuthMessage", typedData["primaryType"])

	domain, ok := typedData["domain"].(map[string]interface{})
	require.True(t, ok, "domain should be a map")
	assert.Equal(t, "app.web3-local-dev.com", domain["name"])
	assert.Equal(t, "1", domain["version"])
	assert.Equal(t, float64(1337), domain["chainId"])

	message, ok := typedData["message"].(map[string]interface{})
	require.True(t, ok, "message should be a map")
	assert.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", message["address"])
	assert.Equal(t, "Sign in to Web3 Lab", message["statement"])
	assert.Equal(t, "abc123nonce", message["nonce"])
	assert.Equal(t, "2026-03-24T12:00:00Z", message["issuedAt"])
	assert.Equal(t, "2026-03-24T12:05:00Z", message["expiresAt"])

	// Check types structure
	types, ok := typedData["types"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, types, "EIP712Domain")
	assert.Contains(t, types, "AuthMessage")
}

// --- EIP-712 Message Parsing ---

func TestParseEIP712Message(t *testing.T) {
	s := newTestSIWEService()

	t.Run("valid JSON", func(t *testing.T) {
		msg := `{"types":{},"primaryType":"AuthMessage","domain":{},"message":{"address":"0xABCdef","nonce":"myNonce","statement":"test"}}`

		addr, nonce, err := s.parseEIP712Message(msg)
		require.NoError(t, err)
		assert.Equal(t, "0xABCdef", addr)
		assert.Equal(t, "myNonce", nonce)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, _, err := s.parseEIP712Message("not-json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse eip712 json")
	})

	t.Run("missing address", func(t *testing.T) {
		msg := `{"message":{"nonce":"myNonce"}}`
		_, _, err := s.parseEIP712Message(msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "address not found")
	})

	t.Run("missing nonce", func(t *testing.T) {
		msg := `{"message":{"address":"0xABC"}}`
		_, _, err := s.parseEIP712Message(msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nonce not found")
	})
}

// --- URL Rewriting ---

func TestRewriteToInternalURL(t *testing.T) {
	s := newTestSIWEService()

	t.Run("rewrites external gateway to internal hydra", func(t *testing.T) {
		external := "https://gateway.web3-local-dev.com/oauth2/auth?client_id=test&scope=openid&login_verifier=abc"
		result := s.rewriteToInternalURL(external)

		assert.Equal(t, "http://hydra-public.web3.svc.cluster.local:4444/oauth2/auth?client_id=test&scope=openid&login_verifier=abc", result)
	})

	t.Run("preserves path", func(t *testing.T) {
		external := "https://gateway.web3-local-dev.com/some/deep/path"
		result := s.rewriteToInternalURL(external)

		assert.Contains(t, result, "http://hydra-public.web3.svc.cluster.local:4444/some/deep/path")
	})

	t.Run("empty hydra public URL returns original", func(t *testing.T) {
		s2 := &SIWEService{
			hydra: &HydraClientService{publicURL: ""},
		}
		original := "https://gateway.web3-local-dev.com/test"
		result := s2.rewriteToInternalURL(original)
		assert.Equal(t, original, result)
	})

	t.Run("invalid URL returns original", func(t *testing.T) {
		result := s.rewriteToInternalURL("://not-a-url")
		assert.Equal(t, "://not-a-url", result)
	})
}

// --- parseMessage dispatcher ---

func TestParseMessage(t *testing.T) {
	s := newTestSIWEService()

	t.Run("siwe protocol", func(t *testing.T) {
		msg := `domain wants you to sign in with your Ethereum account:
0xAbCdEf1234567890AbCdEf1234567890AbCdEf12

Statement

Nonce: testNonce`

		addr, nonce, err := s.parseMessage(msg, "siwe")
		require.NoError(t, err)
		assert.Equal(t, "0xAbCdEf1234567890AbCdEf1234567890AbCdEf12", addr)
		assert.Equal(t, "testNonce", nonce)
	})

	t.Run("eip712 protocol", func(t *testing.T) {
		msg := `{"message":{"address":"0xABC","nonce":"n123"}}`
		addr, nonce, err := s.parseMessage(msg, "eip712")
		require.NoError(t, err)
		assert.Equal(t, "0xABC", addr)
		assert.Equal(t, "n123", nonce)
	})

	t.Run("unsupported protocol", func(t *testing.T) {
		_, _, err := s.parseMessage("", "unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported protocol")
	})
}
