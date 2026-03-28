package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"

	"github.com/web3-lab/backend/internal/config"
	"github.com/web3-lab/backend/pkg/logs"
)

// StorageService handles MinIO object storage operations.
type StorageService struct {
	client          *minio.Client // Internal operations
	presignClient   *minio.Client // Public operations (presigning URLs)
	bucketName      string
	publicBaseURL   string
	internalBaseURL string
	presignedExpiry time.Duration
}

// NFTMetadata represents the standard ERC-721/1155 metadata JSON.
type NFTMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

// NewStorageService creates a new MinIO storage service.
func NewStorageService(cfg config.MinIOConfig) (*StorageService, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	expiry := 15 // default 15 minutes
	if cfg.PresignedExpiry > 0 {
		expiry = cfg.PresignedExpiry
	}

	// Extract host from public base URL for correct presigned URL signature generation
	var publicHost string
	var publicSecure bool = cfg.UseSSL
	if u, err := url.Parse(cfg.PublicBaseURL); err == nil {
		publicHost = u.Host
		publicSecure = u.Scheme == "https"
	}

	// Create a separate client initialized with the public host simply for presigning
	// This ensures the AWS v4 signature signs the exact Host header the browser will send.
	// We MUST provide the Region explicitly. Otherwise, minio-go will attempt to make a
	// network request (Get /?location=) to the public host to find the region, which will
	// fail (connection refused) from inside the Kubernetes pod.
	presignClient, err := minio.New(publicHost, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: publicSecure,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("create minio presign client: %w", err)
	}

	svc := &StorageService{
		client:          client,
		presignClient:   presignClient,
		bucketName:      cfg.BucketName,
		publicBaseURL:   strings.TrimRight(cfg.PublicBaseURL, "/"),
		internalBaseURL: strings.TrimRight(cfg.InternalBaseURL, "/"),
		presignedExpiry: time.Duration(expiry) * time.Minute,
	}

	logs.Logger.Info("MinIO storage service initialized",
		zap.String("endpoint", cfg.Endpoint),
		zap.String("bucket", cfg.BucketName),
	)

	return svc, nil
}

// GeneratePresignedPutURL creates a presigned PUT URL for direct browser uploads.
// The returned URL has the correct public signature and points to the external Ingress.
func (s *StorageService) GeneratePresignedPutURL(ctx context.Context, objectKey, contentType string) (string, error) {
	presignedURL, err := s.presignClient.PresignedPutObject(ctx, s.bucketName, objectKey, s.presignedExpiry)
	if err != nil {
		return "", fmt.Errorf("generate presigned put url: %w", err)
	}

	return presignedURL.String(), nil
}

// UploadMetadataJSON uploads a metadata JSON object to MinIO.
func (s *StorageService) UploadMetadataJSON(ctx context.Context, objectKey string, metadata *NFTMetadata) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	reader := bytes.NewReader(data)
	_, err = s.client.PutObject(ctx, s.bucketName, objectKey, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("upload metadata: %w", err)
	}

	logs.Logger.Info("Metadata JSON uploaded",
		zap.String("key", objectKey),
		zap.Int("size", len(data)),
	)

	return nil
}

// BuildObjectKey constructs the MinIO object key based on token type and identifiers.
func BuildObjectKey(tokenType, contractAddress, tokenID, fileType string) string {
	addr := strings.ToLower(contractAddress)
	switch strings.ToUpper(tokenType) {
	case "ERC20":
		return fmt.Sprintf("erc20/%s.%s", addr, fileType)
	case "ERC721":
		if fileType == "json" || fileType == "metadata" {
			return fmt.Sprintf("erc721/%s/metadata/%s", addr, tokenID)
		}
		return fmt.Sprintf("erc721/%s/images/%s.%s", addr, tokenID, fileType)
	case "ERC1155":
		paddedTokenID := tokenID
		if parsedID, ok := new(big.Int).SetString(tokenID, 10); ok {
			paddedTokenID = fmt.Sprintf("%064x", parsedID)
		}
		if fileType == "json" || fileType == "metadata" {
			return fmt.Sprintf("erc1155/%s/metadata/%s.json", addr, paddedTokenID)
		}
		return fmt.Sprintf("erc1155/%s/images/%s.%s", addr, paddedTokenID, fileType)
	default:
		return fmt.Sprintf("misc/%s/%s.%s", addr, tokenID, fileType)
	}
}

// BuildPublicURL constructs the external (browser-accessible) URL for an object.
func (s *StorageService) BuildPublicURL(objectKey string) string {
	return fmt.Sprintf("%s/%s", s.publicBaseURL, objectKey)
}

// BuildInternalURL constructs the internal (k8s DNS) URL for an object.
func (s *StorageService) BuildInternalURL(objectKey string) string {
	return fmt.Sprintf("%s/%s", s.internalBaseURL, objectKey)
}

// BuildERC721BaseURI constructs the internal base URI for ERC-721 tokenURI resolution.
func (s *StorageService) BuildERC721BaseURI(contractAddress string) string {
	addr := strings.ToLower(contractAddress)
	return fmt.Sprintf("%s/erc721/%s/metadata/", s.internalBaseURL, addr)
}

// BuildERC1155URI constructs the internal URI template for ERC-1155 uri() resolution.
// Uses the {id} placeholder as per ERC-1155 standard.
func (s *StorageService) BuildERC1155URI(contractAddress string) string {
	addr := strings.ToLower(contractAddress)
	return fmt.Sprintf("%s/erc1155/%s/metadata/{id}.json", s.internalBaseURL, addr)
}
