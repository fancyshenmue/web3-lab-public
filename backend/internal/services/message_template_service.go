package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/web3-lab/backend/internal/database"
	"github.com/web3-lab/backend/pkg/logs"
)

const (
	msgTmplCachePrefix = "msg_tmpl"
	msgTmplCacheTTL    = 1 * time.Hour
)

// MessageTemplateService handles CRUD for message templates with Redis caching.
type MessageTemplateService struct {
	repo      database.MessageTemplateRepository
	redis     *redis.Client
	keyPrefix string
}

func NewMessageTemplateService(
	repo database.MessageTemplateRepository,
	redisClient *redis.Client,
	keyPrefix string,
) *MessageTemplateService {
	return &MessageTemplateService{
		repo:      repo,
		redis:     redisClient,
		keyPrefix: keyPrefix,
	}
}

// Create creates a new message template and returns it.
func (s *MessageTemplateService) Create(ctx context.Context, tmpl *database.MessageTemplate) error {
	if tmpl.NonceTTLSecs <= 0 {
		tmpl.NonceTTLSecs = 300
	}
	if tmpl.ChainID <= 0 {
		tmpl.ChainID = 1
	}
	if tmpl.Version == "" {
		tmpl.Version = "1"
	}
	return s.repo.CreateMessageTemplate(ctx, tmpl)
}

// Get retrieves a template by ID, using Redis cache.
func (s *MessageTemplateService) Get(ctx context.Context, id uuid.UUID) (*database.MessageTemplate, error) {
	// Try cache first
	cacheKey := s.cacheKey(id.String())
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var tmpl database.MessageTemplate
		if json.Unmarshal([]byte(cached), &tmpl) == nil {
			return &tmpl, nil
		}
	}

	// Cache miss — fetch from DB
	tmpl, err := s.repo.GetMessageTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, nil
	}

	// Populate cache
	s.setCache(ctx, id.String(), tmpl)
	return tmpl, nil
}

// GetByName retrieves a template by name (no cache — used for lookups like "default").
func (s *MessageTemplateService) GetByName(ctx context.Context, name string) (*database.MessageTemplate, error) {
	return s.repo.GetMessageTemplateByName(ctx, name)
}

// GetDefault returns the template named "default", or nil if not found.
func (s *MessageTemplateService) GetDefault(ctx context.Context) (*database.MessageTemplate, error) {
	return s.repo.GetMessageTemplateByName(ctx, "default")
}

// List returns all templates.
func (s *MessageTemplateService) List(ctx context.Context) ([]*database.MessageTemplate, error) {
	return s.repo.ListMessageTemplates(ctx)
}

// Update updates a template and invalidates the cache.
func (s *MessageTemplateService) Update(ctx context.Context, tmpl *database.MessageTemplate) error {
	if err := s.repo.UpdateMessageTemplate(ctx, tmpl); err != nil {
		return err
	}
	s.invalidateCache(ctx, tmpl.ID.String())
	return nil
}

// Delete deletes a template if no app_clients reference it.
func (s *MessageTemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	count, err := s.repo.CountAppClientsByTemplateID(ctx, id)
	if err != nil {
		return fmt.Errorf("check references: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("template is still referenced by %d app client(s)", count)
	}
	if err := s.repo.DeleteMessageTemplate(ctx, id); err != nil {
		return err
	}
	s.invalidateCache(ctx, id.String())
	return nil
}

// ResolveTemplate returns the best template for the given client_id.
// Priority: app_client.message_template_id → "default" template → nil.
func (s *MessageTemplateService) ResolveTemplate(ctx context.Context, clientID *uuid.UUID) (*database.MessageTemplate, error) {
	if clientID != nil {
		// Look up the app_client's linked template
		// (We rely on the caller to pass the template_id directly for now)
		// For a full implementation, we'd join app_client → template here
	}

	// Fallback to default
	return s.GetDefault(ctx)
}

func (s *MessageTemplateService) cacheKey(id string) string {
	return s.keyPrefix + ":" + msgTmplCachePrefix + ":" + id
}

func (s *MessageTemplateService) setCache(ctx context.Context, id string, tmpl *database.MessageTemplate) {
	data, err := json.Marshal(tmpl)
	if err != nil {
		logs.Logger.Warn("failed to marshal template for cache", zap.Error(err))
		return
	}
	if err := s.redis.Set(ctx, s.cacheKey(id), data, msgTmplCacheTTL).Err(); err != nil {
		logs.Logger.Warn("failed to set template cache", zap.Error(err))
	}
}

func (s *MessageTemplateService) invalidateCache(ctx context.Context, id string) {
	if err := s.redis.Del(ctx, s.cacheKey(id)).Err(); err != nil {
		logs.Logger.Warn("failed to invalidate template cache", zap.Error(err))
	}
}
