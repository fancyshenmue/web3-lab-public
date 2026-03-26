package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/web3-lab/backend/internal/database"
	"github.com/web3-lab/backend/pkg/logs"
)

// SIWEService orchestrates the SIWE (+ EIP-712) wallet authentication flow:
// template resolution → message generation → signature verification → identity management.
type SIWEService struct {
	templates  *MessageTemplateService
	nonces     *NonceService
	auth       *AuthService
	accounts   *AccountService
	kratos     *KratosAdminService
	hydra      *HydraClientService

	// Defaults (used when no template is resolved)
	defaultDomain    string
	defaultURI       string
	defaultStatement string
	defaultChainID   int
	defaultVersion   string
}

func NewSIWEService(
	templates *MessageTemplateService,
	nonces *NonceService,
	auth *AuthService,
	accounts *AccountService,
	kratos *KratosAdminService,
	hydra *HydraClientService,
	domain, uri, statement, version string,
	chainID int,
) *SIWEService {
	return &SIWEService{
		templates:        templates,
		nonces:           nonces,
		auth:             auth,
		accounts:         accounts,
		kratos:           kratos,
		hydra:            hydra,
		defaultDomain:    domain,
		defaultURI:       uri,
		defaultStatement: statement,
		defaultChainID:   chainID,
		defaultVersion:   version,
	}
}

// NonceResult is returned by GenerateNonce.
type NonceResult struct {
	Nonce     string    `json:"nonce"`
	Message   string    `json:"message"`
	Protocol  string    `json:"protocol"`
	ExpiresAt time.Time `json:"expires_at"`
	Domain    string    `json:"domain"`
	ChainID   int       `json:"chain_id"`
}

// SIWEVerifyResult is returned by Verify.
type SIWEVerifyResult struct {
	AccountID    uuid.UUID `json:"account_id"`
	SessionToken string    `json:"session_token"`
	IsNew        bool      `json:"is_new"`
}

// SIWEAuthenticateResult is returned by Authenticate.
type SIWEAuthenticateResult struct {
	RedirectTo string    `json:"redirect_to"`
	AccountID  uuid.UUID `json:"account_id"`
	IdentityID string    `json:"identity_id"`
	IsNewUser  bool      `json:"is_new_user"`
}

// GenerateNonce resolves the message template, generates a nonce, and returns
// a pre-formatted message for the wallet to sign.
func (s *SIWEService) GenerateNonce(ctx context.Context, address, protocol string, clientID *uuid.UUID) (*NonceResult, error) {
	if protocol == "" {
		protocol = "siwe"
	}

	// Resolve template
	tmpl, err := s.templates.ResolveTemplate(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("resolve template: %w", err)
	}

	domain := s.defaultDomain
	uri := s.defaultURI
	statement := s.defaultStatement
	chainID := s.defaultChainID
	version := s.defaultVersion
	nonceTTLSecs := 300

	if tmpl != nil {
		domain = tmpl.Domain
		uri = tmpl.URI
		statement = tmpl.Statement
		chainID = tmpl.ChainID
		version = tmpl.Version
		nonceTTLSecs = tmpl.NonceTTLSecs
	}

	// Variable substitution in statement
	statement = strings.ReplaceAll(statement, "{service_name}", "Web3 Lab")

	// Generate nonce
	nonce, err := s.nonces.Generate(ctx, strings.ToLower(address))
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(nonceTTLSecs) * time.Second)

	var message string
	switch protocol {
	case "siwe":
		message = s.generateSIWEMessage(domain, address, statement, uri, version, chainID, nonce, now, expiresAt)
	case "eip712":
		message, err = s.generateEIP712Message(domain, address, statement, version, chainID, nonce, now, expiresAt)
		if err != nil {
			return nil, fmt.Errorf("generate eip712 message: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	return &NonceResult{
		Nonce:     nonce,
		Message:   message,
		Protocol:  protocol,
		ExpiresAt: expiresAt,
		Domain:    domain,
		ChainID:   chainID,
	}, nil
}

// Verify verifies the wallet signature, finds/creates an identity, and returns a session token.
func (s *SIWEService) Verify(ctx context.Context, message, signature, protocol string) (*SIWEVerifyResult, error) {
	// 1. Parse message to extract address and nonce
	address, nonce, err := s.parseMessage(message, protocol)
	if err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}

	addr := strings.ToLower(address)

	// 2. Verify nonce
	valid, err := s.nonces.Verify(ctx, addr, nonce)
	if err != nil {
		return nil, fmt.Errorf("verify nonce: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid or expired nonce")
	}

	// 3. Verify signature
	ok, err := s.verifySignature(address, signature, message, protocol)
	if err != nil {
		return nil, fmt.Errorf("verify signature: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("signature does not match address")
	}

	// 4. Find or create identity
	acct, identity, isNew, err := s.findOrCreateAccount(ctx, addr)
	if err != nil {
		return nil, err
	}

	// 5. Create Kratos session (use identity's KratosIdentityID)
	sessionToken, err := s.kratos.CreateSession(ctx, identity.KratosIdentityID)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &SIWEVerifyResult{
		AccountID:    acct.AccountID,
		SessionToken: sessionToken,
		IsNew:        isNew,
	}, nil
}

// Authenticate verifies the signature and completes the Hydra OAuth2 flow.
func (s *SIWEService) Authenticate(ctx context.Context, message, signature, protocol, loginChallenge string) (*SIWEAuthenticateResult, error) {
	// 1. Parse, verify nonce + signature (same as Verify)
	address, nonce, err := s.parseMessage(message, protocol)
	if err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}

	addr := strings.ToLower(address)

	valid, err := s.nonces.Verify(ctx, addr, nonce)
	if err != nil {
		return nil, fmt.Errorf("verify nonce: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid or expired nonce")
	}

	ok, err := s.verifySignature(address, signature, message, protocol)
	if err != nil {
		return nil, fmt.Errorf("verify signature: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("signature does not match address")
	}

	// 2. Find or create identity
	acct, identity, isNew, err := s.findOrCreateAccount(ctx, addr)
	if err != nil {
		return nil, err
	}

	// 3. Accept Hydra login (use identity's KratosIdentityID as subject)
	loginRedirect, err := s.hydra.AcceptLoginRequest(ctx, loginChallenge, identity.KratosIdentityID.String())
	if err != nil {
		return nil, fmt.Errorf("accept hydra login: %w", err)
	}

	// 5. Extract consent_challenge from the login redirect URL,
	//    then auto-accept the consent to complete the OAuth2 flow.
	//    This makes SIWE behave identically to the browser-based email/Google login.
	finalRedirect, err := s.autoAcceptConsent(ctx, loginRedirect)
	if err != nil {
		logs.FromContext(ctx).Warn("SIWE: consent auto-accept failed, returning login redirect",
			zap.Error(err),
			zap.String("login_redirect", loginRedirect),
		)
		// Fallback: return the login redirect and let the browser handle consent
		finalRedirect = loginRedirect
	}

	return &SIWEAuthenticateResult{
		RedirectTo: finalRedirect,
		AccountID:  acct.AccountID,
		IdentityID: identity.KratosIdentityID.String(),
		IsNewUser:  isNew,
	}, nil
}

// autoAcceptConsent follows the Hydra login redirect to extract the consent
// challenge, fetches the consent request, and auto-accepts it. Returns the
// final redirect URL containing the authorization code.
func (s *SIWEService) autoAcceptConsent(ctx context.Context, loginRedirect string) (string, error) {
	// The loginRedirect is a Hydra URL like:
	// https://gateway.web3-local-dev.com/oauth2/auth?...&consent_challenge=xxx
	// We need to follow it to get the consent challenge.
	// Hydra responds with 302 → consent URL (which has consent_challenge param).

	// First, try to get the consent challenge by making a GET to the loginRedirect
	// with a non-following HTTP client. Hydra will 302 to the consent URL.
	consentChallenge, err := s.extractConsentChallenge(ctx, loginRedirect)
	if err != nil {
		return "", fmt.Errorf("extract consent challenge: %w", err)
	}

	// Get the consent request to retrieve requested scopes
	consentReq, err := s.hydra.GetConsentRequest(ctx, consentChallenge)
	if err != nil {
		return "", fmt.Errorf("get consent request: %w", err)
	}

	// Extract scopes from consent request
	scopes, _ := consentReq["requested_scope"].([]interface{})
	scopeStrs := make([]string, len(scopes))
	for i, scope := range scopes {
		scopeStrs[i], _ = scope.(string)
	}

	appIdentityID := ""
	subject, _ := consentReq["subject"].(string)
	if subject != "" {
		if kratosUUID, err := uuid.Parse(subject); err == nil {
			if ident, err := s.accounts.GetAccountIdentityByKratosID(ctx, kratosUUID); err == nil && ident != nil {
				appIdentityID = ident.IdentityID.String()
			}
		}
	}

	// Auto-accept the consent
	finalRedirect, err := s.hydra.AcceptConsentRequest(ctx, consentChallenge, scopeStrs, appIdentityID)
	if err != nil {
		return "", fmt.Errorf("accept consent: %w", err)
	}

	return finalRedirect, nil
}

// extractConsentChallenge follows the Hydra login redirect (which 302s to the
// consent URL) and extracts the consent_challenge query parameter.
func (s *SIWEService) extractConsentChallenge(ctx context.Context, loginRedirect string) (string, error) {
	// The loginRedirect URL uses Hydra's public issuer URL (e.g. https://gateway.web3-local-dev.com)
	// which is not reachable from inside the cluster. Rewrite it to use the
	// internal Hydra public service URL.
	internalURL := s.rewriteToInternalURL(loginRedirect)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", internalURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("follow login redirect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("unexpected status %d from login redirect", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("no Location header in login redirect response")
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("parse consent redirect URL: %w", err)
	}

	challenge := parsed.Query().Get("consent_challenge")
	if challenge == "" {
		return "", fmt.Errorf("consent_challenge not found in redirect: %s", location)
	}

	return challenge, nil
}

// rewriteToInternalURL replaces the external gateway hostname in Hydra redirect
// URLs with the internal Hydra public service URL reachable from within the cluster.
func (s *SIWEService) rewriteToInternalURL(externalURL string) string {
	hydraPublicURL := s.hydra.PublicURL()
	if hydraPublicURL == "" {
		return externalURL
	}

	// Parse both URLs to extract scheme+host
	extParsed, err := url.Parse(externalURL)
	if err != nil {
		return externalURL
	}
	intParsed, err := url.Parse(hydraPublicURL)
	if err != nil {
		return externalURL
	}

	// Replace scheme and host with internal values
	extParsed.Scheme = intParsed.Scheme
	extParsed.Host = intParsed.Host

	return extParsed.String()
}

// --- Internal helpers ---

// generateSIWEMessage creates an EIP-4361 compliant message.
func (s *SIWEService) generateSIWEMessage(domain, address, statement, uri, version string, chainID int, nonce string, issuedAt, expiresAt time.Time) string {
	return fmt.Sprintf(`%s wants you to sign in with your Ethereum account:
%s

%s

URI: %s
Version: %s
Chain ID: %d
Nonce: %s
Issued At: %s
Expiration Time: %s`,
		domain, address, statement, uri, version, chainID, nonce,
		issuedAt.Format(time.RFC3339),
		expiresAt.Format(time.RFC3339),
	)
}

// generateEIP712Message creates EIP-712 typed data JSON.
func (s *SIWEService) generateEIP712Message(domain, address, statement, version string, chainID int, nonce string, issuedAt, expiresAt time.Time) (string, error) {
	typedData := map[string]interface{}{
		"types": map[string]interface{}{
			"EIP712Domain": []map[string]string{
				{"name": "name", "type": "string"},
				{"name": "version", "type": "string"},
				{"name": "chainId", "type": "uint256"},
			},
			"AuthMessage": []map[string]string{
				{"name": "address", "type": "address"},
				{"name": "statement", "type": "string"},
				{"name": "nonce", "type": "string"},
				{"name": "issuedAt", "type": "string"},
				{"name": "expiresAt", "type": "string"},
			},
		},
		"primaryType": "AuthMessage",
		"domain": map[string]interface{}{
			"name":    domain,
			"version": version,
			"chainId": chainID,
		},
		"message": map[string]interface{}{
			"address":   address,
			"statement": statement,
			"nonce":     nonce,
			"issuedAt":  issuedAt.Format(time.RFC3339),
			"expiresAt": expiresAt.Format(time.RFC3339),
		},
	}

	data, err := json.Marshal(typedData)
	if err != nil {
		return "", fmt.Errorf("marshal eip712: %w", err)
	}
	return string(data), nil
}

// parseMessage extracts address and nonce from the signed message.
func (s *SIWEService) parseMessage(message, protocol string) (address, nonce string, err error) {
	switch protocol {
	case "siwe":
		return s.parseSIWEMessage(message)
	case "eip712":
		return s.parseEIP712Message(message)
	default:
		return "", "", fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// parseSIWEMessage extracts address and nonce from an EIP-4361 message.
func (s *SIWEService) parseSIWEMessage(message string) (address, nonce string, err error) {
	lines := strings.Split(message, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		// Address is on line 2 (0-indexed: line 1)
		if i == 1 && strings.HasPrefix(line, "0x") {
			address = line
		}
		// Nonce line: "Nonce: xxx"
		if strings.HasPrefix(line, "Nonce: ") {
			nonce = strings.TrimPrefix(line, "Nonce: ")
		}
	}
	if address == "" {
		return "", "", fmt.Errorf("address not found in SIWE message")
	}
	if nonce == "" {
		return "", "", fmt.Errorf("nonce not found in SIWE message")
	}
	return address, nonce, nil
}

// parseEIP712Message extracts address and nonce from EIP-712 JSON.
func (s *SIWEService) parseEIP712Message(message string) (address, nonce string, err error) {
	var typedData struct {
		Message struct {
			Address string `json:"address"`
			Nonce   string `json:"nonce"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(message), &typedData); err != nil {
		return "", "", fmt.Errorf("parse eip712 json: %w", err)
	}
	if typedData.Message.Address == "" {
		return "", "", fmt.Errorf("address not found in EIP-712 message")
	}
	if typedData.Message.Nonce == "" {
		return "", "", fmt.Errorf("nonce not found in EIP-712 message")
	}
	return typedData.Message.Address, typedData.Message.Nonce, nil
}

// verifySignature delegates to the appropriate verification method.
func (s *SIWEService) verifySignature(address, signature, message, protocol string) (bool, error) {
	switch protocol {
	case "siwe":
		// SIWE uses personal_sign (EIP-191)
		return s.auth.VerifySignature(address, signature, message)
	case "eip712":
		// EIP-712 typed data signing uses standard HashStruct reconstruction
		return s.auth.VerifyEIP712Signature(address, signature, message)
	default:
		return false, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// findOrCreateAccount looks up or creates a Kratos identity and local account.
func (s *SIWEService) findOrCreateAccount(ctx context.Context, addr string) (*database.Account, *database.AccountIdentity, bool, error) {
	isNew := false
	identity, err := s.accounts.GetIdentityByProviderUserID(ctx, database.ProviderEOA, addr)
	if err != nil {
		return nil, nil, false, fmt.Errorf("lookup identity: %w", err)
	}

	var acct *database.Account
	if identity == nil {
		// New user — create Kratos identity, then local account
		kratosID, err := s.kratos.CreateIdentityWithWallet(ctx, addr)
		if err != nil {
			return nil, nil, false, fmt.Errorf("create kratos identity: %w", err)
		}

		attrs, _ := json.Marshal(map[string]string{"eoa_address": addr})
		acct, identity, err = s.accounts.CreateAccountWithIdentity(ctx, kratosID, database.ProviderEOA, addr, attrs)
		if err != nil {
			return nil, nil, false, fmt.Errorf("create account: %w", err)
		}
		isNew = true
		logs.FromContext(ctx).Info("SIWE: new wallet account created",
			zap.String("address", addr),
			zap.String("account_id", acct.AccountID.String()),
		)
	} else {
		acct, err = s.accounts.GetAccountByID(ctx, identity.AccountID)
		if err != nil {
			return nil, nil, false, fmt.Errorf("get account: %w", err)
		}
		_ = s.accounts.UpdateLastLogin(ctx, acct.AccountID)
	}

	return acct, identity, isNew, nil
}
