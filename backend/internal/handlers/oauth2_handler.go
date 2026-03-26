package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/web3-lab/backend/internal/services"
	"github.com/web3-lab/backend/pkg/logs"
	"go.uber.org/zap"
)

// OAuth2Handler handles Hydra login/consent webhooks.
type OAuth2Handler struct {
	hydra         *services.HydraClientService
	kratos        *services.KratosAdminService
	clientService *services.AppClientService
	accounts      *services.AccountService
}

func NewOAuth2Handler(hydra *services.HydraClientService, kratos *services.KratosAdminService, clientService *services.AppClientService, accounts *services.AccountService) *OAuth2Handler {
	return &OAuth2Handler{hydra: hydra, kratos: kratos, clientService: clientService, accounts: accounts}
}

// HandleLogin processes the Hydra login challenge.
// GET /api/v1/oauth2/login?login_challenge=xxx
func (h *OAuth2Handler) HandleLogin(c *gin.Context) {
	challenge := c.Query("login_challenge")
	if challenge == "" {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "login_challenge is required"))
		return
	}

	loginReq, err := h.hydra.GetLoginRequest(c.Request.Context(), challenge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	// If the user already has a Hydra session (skip=true), accept immediately
	if skip, _ := loginReq["skip"].(bool); skip {
		subject, _ := loginReq["subject"].(string)
		redirectTo, err := h.hydra.AcceptLoginRequest(c.Request.Context(), challenge, subject)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
			return
		}
		c.Redirect(http.StatusFound, redirectTo)
		return
	}

	// No Hydra session — check if user has a Kratos session (e.g., just registered)
	cookies := c.Request.Cookies()
	cookieNames := make([]string, len(cookies))
	for i, ck := range cookies {
		cookieNames[i] = ck.Name
	}
	logs.Logger.Debug("HandleLogin: checking Kratos session",
		zap.Int("cookie_count", len(cookies)),
		zap.Strings("cookie_names", cookieNames),
	)

	identityID, err := h.kratos.CheckSessionWithCookies(c.Request.Context(), cookies)
	logs.Logger.Debug("HandleLogin: Kratos session check result",
		zap.String("identity_id", identityID),
		zap.Error(err),
	)
	if err == nil && identityID != "" {
		// User has a valid Kratos session — accept the Hydra login
		redirectTo, err := h.hydra.AcceptLoginRequest(c.Request.Context(), challenge, identityID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
			return
		}
		logs.Logger.Info("HandleLogin: auto-accepted via Kratos session",
			zap.String("identity_id", identityID),
		)
		c.Redirect(http.StatusFound, redirectTo)
		return
	}

	// No session at all — redirect to the login page
	clientInfo, ok := loginReq["client"].(map[string]any)
	if !ok {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "missing client info in hydra login request"))
		return
	}
	clientID, _ := clientInfo["client_id"].(string)

	appClient, err := h.clientService.GetCachedClient(c.Request.Context(), clientID)
	if err != nil || appClient == nil {
		// Fallback to returning JSON if client config is missing
		c.JSON(http.StatusOK, loginReq)
		return
	}

	loginURL := fmt.Sprintf("%s%s?login_challenge=%s", appClient.FrontendURL, appClient.LoginPath, challenge)
	c.Redirect(http.StatusFound, loginURL)
}

// HandleConsent processes the Hydra consent challenge.
// GET /api/v1/oauth2/consent?consent_challenge=xxx
func (h *OAuth2Handler) HandleConsent(c *gin.Context) {
	challenge := c.Query("consent_challenge")
	if challenge == "" {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "consent_challenge is required"))
		return
	}

	consentReq, err := h.hydra.GetConsentRequest(c.Request.Context(), challenge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	// Auto-approve consent for first-party clients.
	// For first-party apps we control, we skip the consent screen entirely.
	scopes, _ := consentReq["requested_scope"].([]interface{})
	scopeStrs := make([]string, len(scopes))
	for i, s := range scopes {
		scopeStrs[i], _ = s.(string)
	}

	appIdentityID := ""
	subject, _ := consentReq["subject"].(string)
	if subject != "" {
		if kratosUUID, err := uuid.Parse(subject); err == nil {
			if ident, err := h.accounts.GetAccountIdentityByKratosID(c.Request.Context(), kratosUUID); err == nil && ident != nil {
				appIdentityID = ident.IdentityID.String()
			}
		}
	}

	redirectTo, err := h.hydra.AcceptConsentRequest(c.Request.Context(), challenge, scopeStrs, appIdentityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}
	c.Redirect(http.StatusFound, redirectTo)
}

// HandleLogout processes the Hydra logout challenge.
// GET /api/v1/oauth2/logout?logout_challenge=xxx
func (h *OAuth2Handler) HandleLogout(c *gin.Context) {
	challenge := c.Query("logout_challenge")
	if challenge == "" {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "logout_challenge is required"))
		return
	}

	// Get logout request to find the subject (Kratos identity ID)
	logoutReq, err := h.hydra.GetLogoutRequest(c.Request.Context(), challenge)
	if err != nil {
		logs.Logger.Warn("HandleLogout: failed to get logout request from Hydra",
			zap.Error(err),
		)
	} else {
		logs.Logger.Debug("HandleLogout: got logout request",
			zap.Any("logout_request", logoutReq),
		)
		subject, _ := logoutReq["subject"].(string)
		if subject == "" {
			logs.Logger.Warn("HandleLogout: no subject in logout request")
		} else {
			// Revoke all Kratos sessions for this identity
			if revokeErr := h.kratos.RevokeIdentitySessions(c.Request.Context(), subject); revokeErr != nil {
				logs.Logger.Warn("HandleLogout: failed to revoke Kratos sessions",
					zap.String("subject", subject),
					zap.Error(revokeErr),
				)
			} else {
				logs.Logger.Info("HandleLogout: revoked Kratos sessions",
					zap.String("subject", subject),
				)
			}
		}
	}

	redirectTo, err := h.hydra.AcceptLogoutRequest(c.Request.Context(), challenge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}
	c.Redirect(http.StatusFound, redirectTo)
}

// HandleRegistrationWebhook is called by Kratos after a successful registration.
// It creates the account and account_identity records in our database.
// POST /api/v1/oauth2/registration-webhook
func (h *OAuth2Handler) HandleRegistrationWebhook(c *gin.Context) {
	// Parse the webhook payload from Kratos
	var payload struct {
		IdentityID string `json:"identity_id"`
		Email      string `json:"email"`
		Provider   string `json:"provider"` // "email" or "google"
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		logs.Logger.Error("registration webhook: invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	kratosUUID, err := uuid.Parse(payload.IdentityID)
	if err != nil {
		logs.Logger.Error("registration webhook: invalid identity_id", zap.String("identity_id", payload.IdentityID))
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "invalid identity_id"))
		return
	}

	// Determine provider
	providerID := payload.Provider
	if providerID == "" {
		providerID = "email" // default to email
	}

	// provider_user_id is typically the email for email/google, or kratos identity ID as fallback
	providerUserID := payload.Email
	if providerUserID == "" {
		providerUserID = payload.IdentityID
	}

	// Build attributes JSON
	attrs := map[string]string{"email": payload.Email}
	attrsJSON, _ := json.Marshal(attrs)

	// Check if account already exists for this Kratos identity
	existing, _ := h.accounts.GetAccountByKratosIdentityID(c.Request.Context(), kratosUUID)
	if existing != nil {
		logs.Logger.Info("registration webhook: account already exists",
			zap.String("kratos_identity_id", kratosUUID.String()),
			zap.String("account_id", existing.AccountID.String()),
		)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "account_id": existing.AccountID.String()})
		return
	}

	acct, ident, err := h.accounts.CreateAccountWithIdentity(
		c.Request.Context(), kratosUUID, providerID, providerUserID, attrsJSON,
	)
	if err != nil {
		logs.Logger.Error("registration webhook: create account failed",
			zap.Error(err),
			zap.String("kratos_identity_id", kratosUUID.String()),
		)
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	logs.Logger.Info("registration webhook: account created",
		zap.String("account_id", acct.AccountID.String()),
		zap.String("identity_id", ident.IdentityID.String()),
		zap.String("provider", providerID),
	)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "account_id": acct.AccountID.String()})
}
