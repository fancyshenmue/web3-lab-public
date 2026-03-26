package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/web3-lab/backend/internal/services"
)

// AuthHandler handles wallet authentication endpoints.
type AuthHandler struct {
	walletAuth *services.WalletAuthService
	nonces     *services.NonceService

	domain  string
	version string
	chainID int
}

func NewAuthHandler(walletAuth *services.WalletAuthService, nonces *services.NonceService, domain, version string, chainID int) *AuthHandler {
	return &AuthHandler{
		walletAuth: walletAuth,
		nonces:     nonces,
		domain:     domain,
		version:    version,
		chainID:    chainID,
	}
}

// GetChallenge returns a nonce challenge for wallet signing.
// GET /api/v1/auth/challenge?address=0x...
func (h *AuthHandler) GetChallenge(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "address query parameter is required"))
		return
	}

	result, err := h.walletAuth.GenerateChallenge(c.Request.Context(), address, h.domain, h.version, h.chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, result)
}

// VerifyWalletSignature verifies a wallet signature and logs the user in.
// POST /api/v1/auth/verify
func (h *AuthHandler) VerifyWalletSignature(c *gin.Context) {
	var req struct {
		Address   string `json:"address" binding:"required"`
		Signature string `json:"signature" binding:"required"`
		Nonce     string `json:"nonce" binding:"required"`
		Message   string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	result, err := h.walletAuth.VerifyAndLogin(c.Request.Context(), req.Address, req.Signature, req.Nonce, req.Message)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("INVALID_SIGNATURE", err.Error()))
		return
	}

	c.JSON(http.StatusOK, result)
}
