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
}

func NewAccountHandler(accounts *services.AccountService) *AccountHandler {
	return &AccountHandler{accounts: accounts}
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
