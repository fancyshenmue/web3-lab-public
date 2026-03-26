package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// KratosAdminService communicates with the Ory Kratos Admin API.
type KratosAdminService struct {
	adminURL  string
	publicURL string
	client    *http.Client
}

func NewKratosAdminService(adminURL, publicURL string) *KratosAdminService {
	return &KratosAdminService{adminURL: adminURL, publicURL: publicURL, client: &http.Client{}}
}

// CheckSessionWithCookies checks if the user has a valid Kratos session by forwarding
// their browser cookies to the Kratos public /sessions/whoami endpoint.
// Returns the Kratos identity ID (subject) if authenticated, or empty string if not.
func (s *KratosAdminService) CheckSessionWithCookies(ctx context.Context, cookies []*http.Cookie) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.publicURL+"/sessions/whoami", nil)
	if err != nil {
		return "", err
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("kratos whoami: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil // No valid session
	}

	var session struct {
		Identity struct {
			ID string `json:"id"`
		} `json:"identity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", fmt.Errorf("decode session: %w", err)
	}

	return session.Identity.ID, nil
}

// RevokeIdentitySessions revokes ALL sessions for a given Kratos identity.
// This invalidates the ory_kratos_session cookie.
func (s *KratosAdminService) RevokeIdentitySessions(ctx context.Context, identityID string) error {
	url := fmt.Sprintf("%s/admin/identities/%s/sessions", s.adminURL, identityID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("kratos revoke sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("kratos revoke sessions: status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// CreateIdentityWithWallet creates a Kratos identity with wallet_address trait.
// Returns the Kratos identity UUID.
func (s *KratosAdminService) CreateIdentityWithWallet(ctx context.Context, walletAddress string) (uuid.UUID, error) {
	body := map[string]any{
		"schema_id": "default",
		"traits": map[string]any{
			"wallet_address": walletAddress,
		},
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", s.adminURL+"/admin/identities", bytes.NewReader(data))
	if err != nil {
		return uuid.Nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("kratos create identity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return uuid.Nil, fmt.Errorf("kratos create identity: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return uuid.Nil, fmt.Errorf("decode kratos response: %w", err)
	}

	return uuid.Parse(result.ID)
}

// CreateSession creates a Kratos session for an identity and returns the session token.
func (s *KratosAdminService) CreateSession(ctx context.Context, identityID uuid.UUID) (string, error) {
	body := map[string]any{
		"identity_id": identityID.String(),
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", s.adminURL+"/admin/sessions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("kratos create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("kratos create session: status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		SessionToken string `json:"session_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode session response: %w", err)
	}

	return result.SessionToken, nil
}

// GetIdentityByWallet looks up a Kratos identity by wallet_address trait.
func (s *KratosAdminService) GetIdentityByWallet(ctx context.Context, walletAddress string) (uuid.UUID, error) {
	url := fmt.Sprintf("%s/admin/identities?credentials_identifier=%s", s.adminURL, walletAddress)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return uuid.Nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("kratos get identity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return uuid.Nil, nil
	}

	var identities []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&identities); err != nil {
		return uuid.Nil, fmt.Errorf("decode identities: %w", err)
	}
	if len(identities) == 0 {
		return uuid.Nil, nil
	}

	return uuid.Parse(identities[0].ID)
}
