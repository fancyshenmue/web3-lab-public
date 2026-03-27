package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/web3-lab/backend/internal/services"
)

// SIWEHandler handles wallet authentication endpoints.
type SIWEHandler struct {
	siweService *services.SIWEService
}

func NewSIWEHandler(siweService *services.SIWEService) *SIWEHandler {
	return &SIWEHandler{siweService: siweService}
}

// GetNonce generates a nonce and returns a pre-formatted sign-in message.
// GET /api/v1/siwe/nonce?address=0x...&protocol=siwe&client_id=...
func (h *SIWEHandler) GetNonce(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "address is required"})
		return
	}

	protocol := c.DefaultQuery("protocol", "siwe")
	if protocol != "siwe" && protocol != "eip712" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "protocol must be 'siwe' or 'eip712'"})
		return
	}

	var clientID *uuid.UUID
	if cid := c.Query("client_id"); cid != "" {
		parsed, err := uuid.Parse(cid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid client_id"})
			return
		}
		clientID = &parsed
	}

	result, err := h.siweService.GenerateNonce(c.Request.Context(), address, protocol, clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

type verifyRequest struct {
	Message   string `json:"message" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	Protocol  string `json:"protocol" binding:"required"`
}

// Verify verifies a wallet signature and returns a Kratos session (standalone).
// POST /api/v1/siwe/verify
func (h *SIWEHandler) Verify(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	result, err := h.siweService.Verify(c.Request.Context(), req.Message, req.Signature, req.Protocol)
	if err != nil {
		if isVerificationError(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_SIGNATURE", "message": err.Error()})
			return
		}
		if isNonceError(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "NONCE_EXPIRED", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

type authenticateRequest struct {
	Message        string `json:"message" binding:"required"`
	Signature      string `json:"signature" binding:"required"`
	Protocol       string `json:"protocol" binding:"required"`
	LoginChallenge string `json:"login_challenge" binding:"required"`
}

// Authenticate verifies a wallet signature and completes the Hydra OAuth2 flow.
// POST /api/v1/siwe/authenticate
func (h *SIWEHandler) Authenticate(c *gin.Context) {
	var req authenticateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	result, err := h.siweService.Authenticate(c.Request.Context(), req.Message, req.Signature, req.Protocol, req.LoginChallenge)
	if err != nil {
		if isVerificationError(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_SIGNATURE", "message": err.Error()})
			return
		}
		if isNonceError(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "NONCE_EXPIRED", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// --- Error classification helpers ---

func isVerificationError(err error) bool {
	msg := err.Error()
	return msg == "signature does not match address" || msg == "verify signature: signature does not match address"
}

func isNonceError(err error) bool {
	msg := err.Error()
	return msg == "invalid or expired nonce" || msg == "verify nonce: nonce not found"
}

type linkRequest struct {
	Message   string `json:"message" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	Protocol  string `json:"protocol" binding:"required"`
}

// LinkEOA links a new EOA wallet to the currently authenticated account.
// POST /api/v1/auth/siwe/link
func (h *SIWEHandler) LinkEOA(c *gin.Context) {
	accountID, exists := c.Get("account_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "account_id not found in context"})
		return
	}

	var req linkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	result, err := h.siweService.LinkEOA(c.Request.Context(), accountID.(uuid.UUID), req.Message, req.Signature, req.Protocol)
	if err != nil {
		if isVerificationError(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_SIGNATURE", "message": err.Error()})
			return
		}
		if isNonceError(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "NONCE_EXPIRED", "message": err.Error()})
			return
		}
		msg := err.Error()
		if msg == "this wallet is already linked to an account" {
			c.JSON(http.StatusConflict, gin.H{"code": "ALREADY_LINKED", "message": msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
