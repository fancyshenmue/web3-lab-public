package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NonceService manages wallet authentication nonces stored in Redis.
type NonceService struct {
	client    *redis.Client
	keyPrefix string
	ttl       time.Duration
}

func NewNonceService(client *redis.Client, keyPrefix string, ttl time.Duration) *NonceService {
	return &NonceService{client: client, keyPrefix: keyPrefix, ttl: ttl}
}

// Generate creates a cryptographically random nonce and stores it for the given address.
func (s *NonceService) Generate(ctx context.Context, address string) (string, error) {
	nonce, err := randomHex(16)
	if err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	key := s.key(address)
	if err := s.client.Set(ctx, key, nonce, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("store nonce: %w", err)
	}
	return nonce, nil
}

// Verify retrieves and deletes (consume) the nonce for the given address.
func (s *NonceService) Verify(ctx context.Context, address, nonce string) (bool, error) {
	key := s.key(address)
	stored, err := s.client.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("verify nonce: %w", err)
	}
	return stored == nonce, nil
}

func (s *NonceService) key(address string) string {
	return s.keyPrefix + ":" + address
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
