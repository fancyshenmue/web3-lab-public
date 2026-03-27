package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/web3-lab/backend/internal/services"
)

// AccountHandler handles account and identity CRUD endpoints.
type AccountHandler struct {
	accounts *services.AccountService
	wallet   *services.SmartWalletService
}

func NewAccountHandler(accounts *services.AccountService, wallet *services.SmartWalletService) *AccountHandler {
	return &AccountHandler{accounts: accounts, wallet: wallet}
}

// GetAccount returns an account by ID.
// GET /api/v1/accounts/:account_id
func (h *AccountHandler) GetAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "invalid account_id"))
		return
	}

	acct, err := h.accounts.GetAccountByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "account not found"))
		return
	}

	c.JSON(http.StatusOK, acct)
}

// GetAccountByEOA returns an account by its Ethereum address.
// GET /api/v1/accounts/eoa/:eoa_address
func (h *AccountHandler) GetAccountByEOA(c *gin.Context) {
	address := c.Param("eoa_address")
	if address == "" {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "eoa_address is required"))
		return
	}

	acct, err := h.accounts.FindAccountByEOA(c.Request.Context(), address)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "account not found"))
		return
	}

	c.JSON(http.StatusOK, acct)
}

// GetAccountIdentities returns all identities linked to an account.
// GET /api/v1/accounts/:account_id/identities
func (h *AccountHandler) GetAccountIdentities(c *gin.Context) {
	id, err := uuid.Parse(c.Param("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "invalid account_id"))
		return
	}

	idents, err := h.accounts.GetIdentitiesByAccountID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"identities": idents})
}

// GetAccountSessions returns active sessions for an account.
// GET /api/v1/accounts/:account_id/sessions
func (h *AccountHandler) GetAccountSessions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "invalid account_id"))
		return
	}

	sessions, err := h.accounts.GetActiveSessionsByAccountID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// UnlinkIdentity soft-deletes an identity.
// DELETE /api/v1/identities/:identity_id
func (h *AccountHandler) UnlinkIdentity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("identity_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "invalid identity_id"))
		return
	}

	if err := h.accounts.SoftDeleteIdentity(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "unlinked"})
}

// identityResponse is the JSON structure returned for each identity in GetMyIdentities.
type identityResponse struct {
	IdentityID     uuid.UUID `json:"identity_id"`
	ProviderID     string    `json:"provider_id"`
	ProviderUserID string    `json:"provider_user_id"`
	DisplayName    *string   `json:"display_name,omitempty"`
	IsPrimary      bool      `json:"is_primary"`
	LinkedAt       string    `json:"linked_at"`
	SCWAddress     string    `json:"scw_address,omitempty"`
}

// GetMyIdentities returns all identities for the currently authenticated account,
// enriched with the per-identity SCW address.
// GET /api/v1/accounts/me/identities
func (h *AccountHandler) GetMyIdentities(c *gin.Context) {
	accountID, exists := c.Get("account_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errorResponse("UNAUTHORIZED", "account_id not found in context"))
		return
	}

	idents, err := h.accounts.GetIdentitiesByAccountID(c.Request.Context(), accountID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	result := make([]identityResponse, 0, len(idents))
	for _, ident := range idents {
		resp := identityResponse{
			IdentityID:     ident.IdentityID,
			ProviderID:     ident.ProviderID,
			ProviderUserID: ident.ProviderUserID,
			DisplayName:    ident.DisplayName,
			IsPrimary:      ident.IsPrimary,
			LinkedAt:       ident.LinkedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Derive per-identity SCW address if wallet service is available
		if h.wallet != nil {
			scw, err := h.wallet.DeriveWalletAddressByIdentity(c.Request.Context(), ident.IdentityID)
			if err == nil {
				resp.SCWAddress = scw
			}
		}

		result = append(result, resp)
	}

	c.JSON(http.StatusOK, gin.H{"identities": result})
}

// UnlinkMyIdentity soft-deletes an identity owned by the authenticated user.
// Guards against unlinking the last remaining identity.
// DELETE /api/v1/accounts/me/identities/:identity_id
func (h *AccountHandler) UnlinkMyIdentity(c *gin.Context) {
	accountID, exists := c.Get("account_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errorResponse("UNAUTHORIZED", "account_id not found in context"))
		return
	}

	identityID, err := uuid.Parse(c.Param("identity_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "invalid identity_id"))
		return
	}

	if err := h.accounts.SafeUnlinkIdentity(c.Request.Context(), accountID.(uuid.UUID), identityID); err != nil {
		// Differentiate between known errors
		msg := err.Error()
		if msg == "cannot unlink: this is the only remaining identity" {
			c.JSON(http.StatusConflict, errorResponse("LAST_IDENTITY", msg))
			return
		}
		if msg == "identity not found" || msg == "identity does not belong to this account" {
			c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", msg))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", msg))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "unlinked"})
}
