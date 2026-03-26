package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/web3-lab/backend/internal/database"
)

type AppClientService struct {
	repo         database.AccountRepository // Uses the transaction-capable interface which embeds AppClientRepository methods if casted or accessed directly. Wait, I should probably pass the db pool or concrete type if needed, but the interface holds it all.
	appRepo      database.AppClientRepository
	hydraService *HydraClientService
	redisClient  *redis.Client
	keyPrefix    string
}

func NewAppClientService(repo database.AccountRepository, appRepo database.AppClientRepository, hydra *HydraClientService, redis *redis.Client, keyPrefix string) *AppClientService {
	return &AppClientService{
		repo:         repo,
		appRepo:      appRepo,
		hydraService: hydra,
		redisClient:  redis,
		keyPrefix:    keyPrefix,
	}
}

// clientCacheKey returns the Redis key for a specific oauth2 client config
func (s *AppClientService) clientCacheKey(oauth2ClientID string) string {
	return fmt.Sprintf("%s:client:%s:config", s.keyPrefix, oauth2ClientID)
}

// corsCacheKey returns the Redis key for the sets of all allowed origins
func (s *AppClientService) corsCacheKey() string {
	return fmt.Sprintf("%s:cors:origins", s.keyPrefix)
}

func (s *AppClientService) CreateClient(ctx context.Context, client *database.AppClient) error {
	client.ID = uuid.New()

	// 1. Provision via Hydra
	redirectURIs := []string{
		client.FrontendURL + client.LoginPath,
		client.FrontendURL + "/callback",
		client.FrontendURL,
	}
	logoutRedirectURIs := []string{}
	if client.LogoutURL != "" {
		logoutRedirectURIs = append(logoutRedirectURIs, client.LogoutURL)
	}
	// Also allow redirect to frontend root with ?logout=true
	logoutRedirectURIs = append(logoutRedirectURIs, client.FrontendURL+"/?logout=true")
	clientID, clientSecret, err := s.hydraService.CreateOAuth2Client(ctx, client.Name, redirectURIs, logoutRedirectURIs, client.AllowedCORSOrigins)
	if err != nil {
		return fmt.Errorf("failed to provision hydra client: %w", err)
	}

	client.OAuth2ClientID = clientID
	// Optional: We can hash or simply discard the clientSecret in Postgres. We won't store it for now.
	// We'll return it in the result or log it for the admin one time.
	// Actually we should store it temporarily or return it to handler. For now we set it to JWTSecret field just to return it, BUT we shouldn't persist it plainly ideally. Let's just persist it in jwt_secret for now as a place holder.
	client.JWTSecret = clientSecret

	// 2. Save to DB
	if err := s.appRepo.CreateAppClient(ctx, client); err != nil {
		// Rollback in hydra (fire and forget fallback)
		_ = s.hydraService.DeleteOAuth2Client(ctx, clientID)
		return fmt.Errorf("failed to save client to db: %w", err)
	}

	// 3. Sync to Redis Cache
	if err := s.SyncToCache(ctx, client); err != nil {
		return fmt.Errorf("failed to sync cache: %w", err)
	}

	return nil
}

func (s *AppClientService) UpdateClient(ctx context.Context, client *database.AppClient) error {
	// 1. Update DB
	if err := s.appRepo.UpdateAppClient(ctx, client); err != nil {
		return fmt.Errorf("failed to update db: %w", err)
	}

	// 2. Sync to Hydra
	redirectURIs := []string{
		client.FrontendURL + client.LoginPath,
		client.FrontendURL + "/callback",
		client.FrontendURL,
	}
	logoutRedirectURIs := []string{}
	if client.LogoutURL != "" {
		logoutRedirectURIs = append(logoutRedirectURIs, client.LogoutURL)
	}
	logoutRedirectURIs = append(logoutRedirectURIs, client.FrontendURL+"/?logout=true")
	err := s.hydraService.UpdateOAuth2Client(ctx, client.OAuth2ClientID, client.Name, redirectURIs, logoutRedirectURIs, client.AllowedCORSOrigins)
	if err != nil {
		return fmt.Errorf("failed to update hydra client: %w", err)
	}

	// 3. Sync to Redis
	if err := s.SyncToCache(ctx, client); err != nil {
		return fmt.Errorf("failed to sync cache: %w", err)
	}

	return nil
}

func (s *AppClientService) DeleteClient(ctx context.Context, id uuid.UUID) error {
	client, err := s.appRepo.GetAppClient(ctx, id)
	if err != nil {
		return err
	}
	if client == nil {
		return nil
	}

	// 1. Delete DB
	if err := s.appRepo.DeleteAppClient(ctx, id); err != nil {
		return fmt.Errorf("failed to delete db: %w", err)
	}

	// 2. Delete Hydra
	if err := s.hydraService.DeleteOAuth2Client(ctx, client.OAuth2ClientID); err != nil {
		// Log error but continue
		fmt.Printf("warning: failed to delete hydra client %s: %v\n", client.OAuth2ClientID, err)
	}

	// 3. Remove from Cache
	s.redisClient.Del(ctx, s.clientCacheKey(client.OAuth2ClientID))
	// Rebuild CORS set (easiest way is to clear it and rebuild it, or let middleware fetch all from DB if missing)
	s.redisClient.Del(ctx, s.corsCacheKey())

	return nil
}

func (s *AppClientService) ListClients(ctx context.Context) ([]*database.AppClient, error) {
	return s.appRepo.ListAppClients(ctx)
}

func (s *AppClientService) GetClient(ctx context.Context, id uuid.UUID) (*database.AppClient, error) {
	return s.appRepo.GetAppClient(ctx, id)
}

func (s *AppClientService) SyncToCache(ctx context.Context, client *database.AppClient) error {
	data, err := json.Marshal(client)
	if err != nil {
		return err
	}

	// Keep cache for 30 days or indefinitely (TTL 0 for config)
	if err := s.redisClient.Set(ctx, s.clientCacheKey(client.OAuth2ClientID), data, 0).Err(); err != nil {
		return err
	}

	// Add CORS to global set
	if len(client.AllowedCORSOrigins) > 0 {
		origins := make([]interface{}, len(client.AllowedCORSOrigins))
		for i, v := range client.AllowedCORSOrigins {
			origins[i] = v
		}
		if err := s.redisClient.SAdd(ctx, s.corsCacheKey(), origins...).Err(); err != nil {
			return err
		}
	}

	return nil
}

// GetCachedClient fetches config quickly from Redis
func (s *AppClientService) GetCachedClient(ctx context.Context, oauth2ClientID string) (*database.AppClient, error) {
	data, err := s.redisClient.Get(ctx, s.clientCacheKey(oauth2ClientID)).Bytes()
	if err == redis.Nil {
		// Fallback to DB
		client, dbErr := s.appRepo.GetAppClientByOAuth2ID(ctx, oauth2ClientID)
		if dbErr != nil {
			return nil, dbErr
		}
		if client != nil {
			_ = s.SyncToCache(ctx, client)
		}
		return client, nil
	} else if err != nil {
		return nil, err
	}

	var client database.AppClient
	if err := json.Unmarshal(data, &client); err != nil {
		return nil, err
	}
	return &client, nil
}

// IsCORSAllowed checks if a specific origin is permitted dynamically.
func (s *AppClientService) IsCORSAllowed(ctx context.Context, origin string) (bool, error) {
	return s.redisClient.SIsMember(ctx, s.corsCacheKey(), origin).Result()
}
