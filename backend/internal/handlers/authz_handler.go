package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/web3-lab/backend/internal/services"
)

// AuthzHandler handles SpiceDB authorization endpoints.
type AuthzHandler struct {
	authz *services.AuthzService
}

func NewAuthzHandler(authz *services.AuthzService) *AuthzHandler {
	return &AuthzHandler{authz: authz}
}

// CheckPermission checks a SpiceDB permission.
// POST /api/v1/authz/check
func (h *AuthzHandler) CheckPermission(c *gin.Context) {
	var req struct {
		ResourceType string `json:"resource_type" binding:"required"`
		ResourceID   string `json:"resource_id" binding:"required"`
		Permission   string `json:"permission" binding:"required"`
		SubjectType  string `json:"subject_type" binding:"required"`
		SubjectID    string `json:"subject_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	allowed, err := h.authz.CheckPermission(c.Request.Context(),
		req.ResourceType, req.ResourceID, req.Permission, req.SubjectType, req.SubjectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"allowed": allowed})
}

// CreateRelationship writes a SpiceDB relationship.
// POST /internal/spicedb/relationships
func (h *AuthzHandler) CreateRelationship(c *gin.Context) {
	var req struct {
		ResourceType string `json:"resource_type" binding:"required"`
		ResourceID   string `json:"resource_id" binding:"required"`
		Relation     string `json:"relation" binding:"required"`
		SubjectType  string `json:"subject_type" binding:"required"`
		SubjectID    string `json:"subject_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	if err := h.authz.CreateRelationship(c.Request.Context(),
		req.ResourceType, req.ResourceID, req.Relation, req.SubjectType, req.SubjectID); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}

// DeleteRelationship removes a SpiceDB relationship.
// DELETE /internal/spicedb/relationships
func (h *AuthzHandler) DeleteRelationship(c *gin.Context) {
	var req struct {
		ResourceType string `json:"resource_type" binding:"required"`
		ResourceID   string `json:"resource_id" binding:"required"`
		Relation     string `json:"relation" binding:"required"`
		SubjectType  string `json:"subject_type" binding:"required"`
		SubjectID    string `json:"subject_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	if err := h.authz.DeleteRelationship(c.Request.Context(),
		req.ResourceType, req.ResourceID, req.Relation, req.SubjectType, req.SubjectID); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// HealthCheck returns authorization service health.
// GET /api/v1/authz/health
func (h *AuthzHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "spicedb-authz"})
}
