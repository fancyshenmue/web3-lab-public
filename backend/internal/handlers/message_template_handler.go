package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/web3-lab/backend/internal/database"
	"github.com/web3-lab/backend/internal/services"
)

// MessageTemplateHandler handles admin CRUD for message templates.
type MessageTemplateHandler struct {
	templateService *services.MessageTemplateService
}

func NewMessageTemplateHandler(templateService *services.MessageTemplateService) *MessageTemplateHandler {
	return &MessageTemplateHandler{templateService: templateService}
}

type createTemplateRequest struct {
	Name         string `json:"name" binding:"required"`
	Protocol     string `json:"protocol" binding:"required"`
	Statement    string `json:"statement" binding:"required"`
	Domain       string `json:"domain" binding:"required"`
	URI          string `json:"uri" binding:"required"`
	ChainID      int    `json:"chain_id"`
	Version      string `json:"version"`
	NonceTTLSecs int    `json:"nonce_ttl_secs"`
}

// CreateTemplate creates a new message template.
// POST /api/v1/admin/message-templates
func (h *MessageTemplateHandler) CreateTemplate(c *gin.Context) {
	var req createTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	if req.Protocol != "siwe" && req.Protocol != "eip712" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "protocol must be 'siwe' or 'eip712'"})
		return
	}

	tmpl := &database.MessageTemplate{
		Name:         req.Name,
		Protocol:     req.Protocol,
		Statement:    req.Statement,
		Domain:       req.Domain,
		URI:          req.URI,
		ChainID:      req.ChainID,
		Version:      req.Version,
		NonceTTLSecs: req.NonceTTLSecs,
	}

	if err := h.templateService.Create(c.Request.Context(), tmpl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tmpl)
}

// ListTemplates returns all message templates.
// GET /api/v1/admin/message-templates
func (h *MessageTemplateHandler) ListTemplates(c *gin.Context) {
	templates, err := h.templateService.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// GetTemplate returns a single message template.
// GET /api/v1/admin/message-templates/:id
func (h *MessageTemplateHandler) GetTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid template ID"})
		return
	}

	tmpl, err := h.templateService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	if tmpl == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "template not found"})
		return
	}
	c.JSON(http.StatusOK, tmpl)
}

type updateTemplateRequest struct {
	Name         string `json:"name"`
	Statement    string `json:"statement"`
	Domain       string `json:"domain"`
	URI          string `json:"uri"`
	ChainID      int    `json:"chain_id"`
	Version      string `json:"version"`
	NonceTTLSecs int    `json:"nonce_ttl_secs"`
}

// UpdateTemplate updates a message template.
// PUT /api/v1/admin/message-templates/:id
func (h *MessageTemplateHandler) UpdateTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid template ID"})
		return
	}

	existing, err := h.templateService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "template not found"})
		return
	}

	var req updateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	// Apply partial updates
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Statement != "" {
		existing.Statement = req.Statement
	}
	if req.Domain != "" {
		existing.Domain = req.Domain
	}
	if req.URI != "" {
		existing.URI = req.URI
	}
	if req.ChainID > 0 {
		existing.ChainID = req.ChainID
	}
	if req.Version != "" {
		existing.Version = req.Version
	}
	if req.NonceTTLSecs > 0 {
		existing.NonceTTLSecs = req.NonceTTLSecs
	}

	if err := h.templateService.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteTemplate deletes a message template.
// DELETE /api/v1/admin/message-templates/:id
func (h *MessageTemplateHandler) DeleteTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid template ID"})
		return
	}

	if err := h.templateService.Delete(c.Request.Context(), id); err != nil {
		if err.Error() == "template is still referenced by app client(s)" {
			c.JSON(http.StatusConflict, gin.H{"code": "CONFLICT", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
