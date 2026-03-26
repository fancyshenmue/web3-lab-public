package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/web3-lab/backend/internal/database"
	"github.com/web3-lab/backend/internal/services"
)

type ClientHandler struct {
	clientService *services.AppClientService
}

func NewClientHandler(clientService *services.AppClientService) *ClientHandler {
	return &ClientHandler{clientService: clientService}
}

type CreateClientRequest struct {
	Name               string   `json:"name" binding:"required"`
	FrontendURL        string   `json:"frontend_url" binding:"required,url"`
	LoginPath          string   `json:"login_path"`
	LogoutURL          string   `json:"logout_url"`
	AllowedCORSOrigins []string `json:"allowed_cors_origins"`
}

func (h *ClientHandler) CreateClient(c *gin.Context) {
	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if req.LoginPath == "" {
		req.LoginPath = "/login"
	}

	client := &database.AppClient{
		Name:               req.Name,
		FrontendURL:        req.FrontendURL,
		LoginPath:          req.LoginPath,
		LogoutURL:          req.LogoutURL,
		AllowedCORSOrigins: req.AllowedCORSOrigins,
	}

	if err := h.clientService.CreateClient(c.Request.Context(), client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create client", "details": err.Error()})
		return
	}

	// For security, although returned here once so the admin can copy the secret, we usually omit JWTSecret from responses.
	c.JSON(http.StatusCreated, gin.H{"client": client})
}

func (h *ClientHandler) ListClients(c *gin.Context) {
	clients, err := h.clientService.ListClients(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

func (h *ClientHandler) GetClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	client, err := h.clientService.GetClient(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get client"})
		return
	}
	if client == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"client": client})
}

func (h *ClientHandler) UpdateClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if req.LoginPath == "" {
		req.LoginPath = "/login"
	}

	existing, err := h.clientService.GetClient(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get client"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	existing.Name = req.Name
	existing.FrontendURL = req.FrontendURL
	existing.LoginPath = req.LoginPath
	existing.LogoutURL = req.LogoutURL
	existing.AllowedCORSOrigins = req.AllowedCORSOrigins

	if err := h.clientService.UpdateClient(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"client": existing})
}

func (h *ClientHandler) DeleteClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	if err := h.clientService.DeleteClient(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete client", "details": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
