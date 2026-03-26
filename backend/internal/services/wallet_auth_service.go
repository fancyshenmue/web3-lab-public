package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/web3-lab/backend/internal/database"
	"github.com/web3-lab/backend/pkg/logs"
)

// WalletAuthService orchestrates the wallet authentication flow:
// challenge → verify → find-or-create account → create Kratos session.
type WalletAuthService struct {
	accounts *AccountService
	nonces   *NonceService
	auth     *AuthService
	kratos   *KratosAdminService
}

func NewWalletAuthService(
	accounts *AccountService,
	nonces *NonceService,
	auth *AuthService,
	kratos *KratosAdminService,
) *WalletAuthService {
	return &WalletAuthService{
		accounts: accounts,
		nonces:   nonces,
		auth:     auth,
		kratos:   kratos,
	}
}

// ChallengeResult is returned by GenerateChallenge.
type ChallengeResult struct {
	Nonce   string `json:"nonce"`
	Message string `json:"message"`
}

// VerifyResult is returned by VerifyAndLogin.
type VerifyResult struct {
	AccountID    uuid.UUID `json:"account_id"`
	SessionToken string    `json:"session_token"`
	IsNew        bool      `json:"is_new"`
}

// GenerateChallenge creates a nonce and a human-readable sign message.
func (s *WalletAuthService) GenerateChallenge(ctx context.Context, address, domain, version string, chainID int) (*ChallengeResult, error) {
	nonce, err := s.nonces.Generate(ctx, strings.ToLower(address))
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	msg := fmt.Sprintf(
		"%s wants you to sign in with your Ethereum account:\n%s\n\nSign this message to authenticate.\n\nURI: https://%s\nVersion: %s\nChain ID: %d\nNonce: %s\nIssued At: %s",
		domain, address, domain, version, chainID, nonce, time.Now().UTC().Format(time.RFC3339),
	)

	return &ChallengeResult{Nonce: nonce, Message: msg}, nil
}

// VerifyAndLogin verifies the wallet signature and returns (or creates) the account.
func (s *WalletAuthService) VerifyAndLogin(ctx context.Context, address, signature, nonce, message string) (*VerifyResult, error) {
	addr := strings.ToLower(address)

	// 1. Verify nonce
	valid, err := s.nonces.Verify(ctx, addr, nonce)
	if err != nil {
		return nil, fmt.Errorf("verify nonce: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid or expired nonce")
	}

	// 2. Verify signature
	ok, err := s.auth.VerifySignature(address, signature, message)
	if err != nil {
		return nil, fmt.Errorf("verify signature: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("signature does not match address")
	}

	// 3. Find or create account
	isNew := false
	identity, err := s.accounts.GetIdentityByProviderUserID(ctx, database.ProviderEOA, addr)
	if err != nil {
		return nil, fmt.Errorf("lookup identity: %w", err)
	}

	var acct *database.Account
	if identity == nil {
		// New user — create Kratos identity, then local account
		kratosID, err := s.kratos.CreateIdentityWithWallet(ctx, addr)
		if err != nil {
			return nil, fmt.Errorf("create kratos identity: %w", err)
		}

		attrs, _ := json.Marshal(map[string]string{"eoa_address": addr})
		acct, identity, err = s.accounts.CreateAccountWithIdentity(ctx, kratosID, database.ProviderEOA, addr, attrs)
		if err != nil {
			return nil, fmt.Errorf("create account: %w", err)
		}
		isNew = true
		logs.FromContext(ctx).Info("New wallet account created",
			zap.String("address", addr),
			zap.String("account_id", acct.AccountID.String()),
		)
	} else {
		acct, err = s.accounts.GetAccountByID(ctx, identity.AccountID)
		if err != nil {
			return nil, fmt.Errorf("get account: %w", err)
		}
		_ = s.accounts.UpdateLastLogin(ctx, acct.AccountID)
	}

	// 4. Create Kratos session (use identity's KratosIdentityID, not account)
	sessionToken, err := s.kratos.CreateSession(ctx, identity.KratosIdentityID)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &VerifyResult{
		AccountID:    acct.AccountID,
		SessionToken: sessionToken,
		IsNew:        isNew,
	}, nil
}
