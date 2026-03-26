package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HydraClientService communicates with the Ory Hydra Admin API.
type HydraClientService struct {
	adminURL  string
	publicURL string
	client    *http.Client
}

func NewHydraClientService(adminURL, publicURL string) *HydraClientService {
	return &HydraClientService{adminURL: adminURL, publicURL: publicURL, client: &http.Client{}}
}

// PublicURL returns the internal Hydra public URL for server-to-server communication.
func (s *HydraClientService) PublicURL() string {
	return s.publicURL
}

// GetLoginRequest retrieves a Hydra login request by challenge.
func (s *HydraClientService) GetLoginRequest(ctx context.Context, challenge string) (map[string]any, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/login?login_challenge=%s", s.adminURL, challenge)
	return s.getJSON(ctx, url)
}

// AcceptLoginRequest accepts a Hydra login request.
func (s *HydraClientService) AcceptLoginRequest(ctx context.Context, challenge, subject string) (string, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/login/accept?login_challenge=%s", s.adminURL, challenge)
	body := map[string]any{"subject": subject, "remember": true, "remember_for": 3600}
	resp, err := s.putJSON(ctx, url, body)
	if err != nil {
		return "", err
	}
	redirectTo, _ := resp["redirect_to"].(string)
	return redirectTo, nil
}

// GetConsentRequest retrieves a Hydra consent request by challenge.
func (s *HydraClientService) GetConsentRequest(ctx context.Context, challenge string) (map[string]any, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/consent?consent_challenge=%s", s.adminURL, challenge)
	return s.getJSON(ctx, url)
}

// AcceptConsentRequest accepts a Hydra consent request.
func (s *HydraClientService) AcceptConsentRequest(ctx context.Context, challenge string, scopes []string, appIdentityID string) (string, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/consent/accept?consent_challenge=%s", s.adminURL, challenge)
	body := map[string]any{
		"grant_scope":  scopes,
		"remember":     true,
		"remember_for": 3600,
	}

	if appIdentityID != "" {
		body["session"] = map[string]any{
			"id_token": map[string]any{
				"identity_id": appIdentityID,
			},
			"access_token": map[string]any{
				"identity_id": appIdentityID,
			},
		}
	}

	resp, err := s.putJSON(ctx, url, body)
	if err != nil {
		return "", err
	}
	redirectTo, _ := resp["redirect_to"].(string)
	return redirectTo, nil
}

// CreateOAuth2Client provisions a new OAuth2 client in Hydra.
func (s *HydraClientService) CreateOAuth2Client(ctx context.Context, name string, redirectURIs, logoutRedirectURIs, corsOrigins []string) (string, string, error) {
	url := fmt.Sprintf("%s/admin/clients", s.adminURL)
	body := map[string]any{
		"client_name":                  name,
		"redirect_uris":                redirectURIs,
		"post_logout_redirect_uris":    logoutRedirectURIs,
		"allowed_cors_origins":         corsOrigins,
		"response_types":               []string{"code", "id_token", "token"},
		"grant_types":                  []string{"authorization_code", "refresh_token", "implicit"},
		"token_endpoint_auth_method":   "none",
	}
	resp, err := s.postJSON(ctx, url, body)
	if err != nil {
		return "", "", err
	}
	clientID, _ := resp["client_id"].(string)
	clientSecret, _ := resp["client_secret"].(string)
	return clientID, clientSecret, nil
}

// UpdateOAuth2Client updates an existing OAuth2 client in Hydra.
func (s *HydraClientService) UpdateOAuth2Client(ctx context.Context, clientID, name string, redirectURIs, logoutRedirectURIs, corsOrigins []string) error {
	url := fmt.Sprintf("%s/admin/clients/%s", s.adminURL, clientID)
	body := map[string]any{
		"client_name":                  name,
		"redirect_uris":                redirectURIs,
		"post_logout_redirect_uris":    logoutRedirectURIs,
		"allowed_cors_origins":         corsOrigins,
		"response_types":               []string{"code", "id_token", "token"},
		"grant_types":                  []string{"authorization_code", "refresh_token", "implicit"},
		"token_endpoint_auth_method":   "none",
	}
	_, err := s.putJSON(ctx, url, body)
	return err
}

// DeleteOAuth2Client removes an OAuth2 client from Hydra.
func (s *HydraClientService) DeleteOAuth2Client(ctx context.Context, clientID string) error {
	url := fmt.Sprintf("%s/admin/clients/%s", s.adminURL, clientID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hydra DELETE %s: status %d: %s", url, resp.StatusCode, string(b))
	}
	return nil
}

// GetLogoutRequest retrieves a Hydra logout request by challenge.
func (s *HydraClientService) GetLogoutRequest(ctx context.Context, challenge string) (map[string]any, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/logout?logout_challenge=%s", s.adminURL, challenge)
	return s.getJSON(ctx, url)
}

// AcceptLogoutRequest accepts a Hydra logout request and returns the redirect URL.
func (s *HydraClientService) AcceptLogoutRequest(ctx context.Context, challenge string) (string, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/logout/accept?logout_challenge=%s", s.adminURL, challenge)
	resp, err := s.putJSON(ctx, url, map[string]any{})
	if err != nil {
		return "", err
	}
	redirectTo, _ := resp["redirect_to"].(string)
	return redirectTo, nil
}

func (s *HydraClientService) getJSON(ctx context.Context, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hydra %s: status %d: %s", url, resp.StatusCode, string(b))
	}
	var result map[string]any
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

func (s *HydraClientService) putJSON(ctx context.Context, url string, body any) (map[string]any, error) {
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hydra PUT %s: status %d: %s", url, resp.StatusCode, string(b))
	}
	var result map[string]any
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

func (s *HydraClientService) postJSON(ctx context.Context, url string, body any) (map[string]any, error) {
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hydra POST %s: status %d: %s", url, resp.StatusCode, string(b))
	}
	var result map[string]any
	return result, json.NewDecoder(resp.Body).Decode(&result)
}
