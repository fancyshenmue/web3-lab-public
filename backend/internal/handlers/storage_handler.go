package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/web3-lab/backend/internal/services"
	"github.com/web3-lab/backend/pkg/logs"
)

// StorageHandler handles storage-related HTTP requests.
type StorageHandler struct {
	storageSvc *services.StorageService
}

// NewStorageHandler creates a new storage handler.
func NewStorageHandler(storageSvc *services.StorageService) *StorageHandler {
	return &StorageHandler{storageSvc: storageSvc}
}

// PresignedURLRequest is the request body for generating a presigned PUT URL.
type PresignedURLRequest struct {
	TokenType       string `json:"token_type" binding:"required"`       // ERC20, ERC721, ERC1155
	ContractAddress string `json:"contract_address" binding:"required"` // 0x...
	TokenID         string `json:"token_id"`                            // optional for ERC-20
	FileExtension   string `json:"file_extension" binding:"required"`   // png, jpg, webp
}

// PresignedURLResponse is the response for presigned URL generation.
type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"` // Presigned PUT URL (via Ingress)
	ObjectKey string `json:"object_key"` // MinIO object key
	PublicURL string `json:"public_url"` // Public URL for the uploaded file
}

// GeneratePresignedURL generates a presigned PUT URL for direct browser upload.
func (h *StorageHandler) GeneratePresignedURL(c *gin.Context) {
	var req PresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	objectKey := services.BuildObjectKey(req.TokenType, req.ContractAddress, req.TokenID, req.FileExtension)

	contentType := "image/png"
	switch req.FileExtension {
	case "jpg", "jpeg":
		contentType = "image/jpeg"
	case "webp":
		contentType = "image/webp"
	case "gif":
		contentType = "image/gif"
	case "svg":
		contentType = "image/svg+xml"
	}

	uploadURL, err := h.storageSvc.GeneratePresignedPutURL(c.Request.Context(), objectKey, contentType)
	if err != nil {
		logs.Logger.Error("Failed to generate presigned URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate upload URL"})
		return
	}

	publicURL := h.storageSvc.BuildPublicURL(objectKey)

	c.JSON(http.StatusOK, PresignedURLResponse{
		UploadURL: uploadURL,
		ObjectKey: objectKey,
		PublicURL: publicURL,
	})
}

// MetadataRequest is the request body for generating and uploading metadata JSON.
type MetadataRequest struct {
	TokenType       string `json:"token_type" binding:"required"`       // ERC721, ERC1155
	ContractAddress string `json:"contract_address" binding:"required"` // 0x...
	TokenID         string `json:"token_id" binding:"required"`         // e.g. "0", "1"
	Name            string `json:"name" binding:"required"`             // NFT name
	Description     string `json:"description"`                        // NFT description
	ImageURL        string `json:"image_url"`                          // pre-computed public URL for the image
}

// MetadataResponse is the response for metadata creation.
type MetadataResponse struct {
	MetadataURL string `json:"metadata_url"` // Public URL for the metadata JSON
	InternalURL string `json:"internal_url"` // Internal k8s URL for on-chain reference
}

// GenerateMetadata creates and uploads a metadata JSON file for ERC-721/1155 tokens.
func (h *StorageHandler) GenerateMetadata(c *gin.Context) {
	var req MetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If no image URL provided, construct it from the contract address and token ID
	imageURL := req.ImageURL
	if imageURL == "" {
		imageKey := services.BuildObjectKey(req.TokenType, req.ContractAddress, req.TokenID, "png")
		imageURL = h.storageSvc.BuildPublicURL(imageKey)
	}

	metadata := &services.NFTMetadata{
		Name:        req.Name,
		Description: req.Description,
		Image:       imageURL,
	}

	metadataKey := services.BuildObjectKey(req.TokenType, req.ContractAddress, req.TokenID, "metadata")

	if err := h.storageSvc.UploadMetadataJSON(c.Request.Context(), metadataKey, metadata); err != nil {
		logs.Logger.Error("Failed to upload metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload metadata"})
		return
	}

	c.JSON(http.StatusOK, MetadataResponse{
		MetadataURL: h.storageSvc.BuildPublicURL(metadataKey),
		InternalURL: h.storageSvc.BuildInternalURL(metadataKey),
	})
}

// ERC20IconRequest is the request body for uploading an ERC-20 token icon.
type ERC20IconRequest struct {
	ContractAddress string `json:"contract_address" binding:"required"` // 0x...
	IconURL         string `json:"icon_url" binding:"required"`         // Public URL of the uploaded icon
}

// UploadERC20Icon records an ERC-20 token icon URL.
// Note: Blockscout DB update (UPDATE tokens SET icon_url = ...) should be handled
// by a separate script or integrated when we have Blockscout DB access from the backend.
func (h *StorageHandler) UploadERC20Icon(c *gin.Context) {
	var req ERC20IconRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logs.Logger.Info("ERC-20 icon registered",
		zap.String("contract", req.ContractAddress),
		zap.String("icon_url", req.IconURL),
	)

	// TODO: Direct Blockscout DB update via pgx connection
	// For now, return the icon URL for the frontend to display
	c.JSON(http.StatusOK, gin.H{
		"contract_address": req.ContractAddress,
		"icon_url":         req.IconURL,
		"message":          "Icon URL registered. Run 'make seed-update-icons' to update Blockscout DB.",
	})
}
