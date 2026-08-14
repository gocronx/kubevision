package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
)

const apiKeyPrefix = "kv_"

// APIKeyService handles business logic for API key management.
type APIKeyService struct {
	repo     repository.APIKeyRepo
	userRepo repository.UserRepo
}

// NewAPIKeyService creates a new APIKeyService.
func NewAPIKeyService(repo repository.APIKeyRepo, userRepo repository.UserRepo) *APIKeyService {
	return &APIKeyService{
		repo:     repo,
		userRepo: userRepo,
	}
}

// GenerateKeyRequest holds parameters for creating a new API key.
type GenerateKeyRequest struct {
	Name      string     `json:"name" binding:"required"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

// GenerateKeyResponse contains the newly created API key.
// The plain-text key is only returned here — it cannot be retrieved later.
type GenerateKeyResponse struct {
	ID        uint       `json:"id"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"keyPrefix"`
	PlainKey  string     `json:"key"` // shown only once
	ExpiresAt *time.Time `json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

// Generate creates a new API key for the given user. It returns the plain-text
// key once; only the SHA-256 hash is stored in the database.
//
// Key format: kv_ + 64 hex characters (32 random bytes).
func (s *APIKeyService) Generate(ctx context.Context, userID uint, name string, expiresAt *time.Time) (*GenerateKeyResponse, error) {
	// Verify user exists.
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, "user not found")
	}

	// Generate 32 random bytes and encode as hex.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate key material")
	}
	plainKey := apiKeyPrefix + hex.EncodeToString(raw)

	keyHash := hashAPIKey(plainKey)

	// KeyPrefix is the first 10 characters shown in listings.
	keyPrefix := plainKey[:min(10, len(plainKey))]

	record := &model.APIKey{
		UserID:    userID,
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.Create(ctx, record); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, fmt.Sprintf("failed to store API key: %v", err))
	}

	return &GenerateKeyResponse{
		ID:        record.ID,
		Name:      record.Name,
		KeyPrefix: record.KeyPrefix,
		PlainKey:  plainKey,
		ExpiresAt: record.ExpiresAt,
		CreatedAt: record.CreatedAt,
	}, nil
}

// Validate looks up a plain-text API key, verifies it has not expired, and
// returns the matching model.APIKey on success.
func (s *APIKeyService) Validate(ctx context.Context, plainKey string) (*model.APIKey, error) {
	keyHash := hashAPIKey(plainKey)

	record, err := s.repo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "invalid API key")
	}

	if record.ExpiresAt != nil && time.Now().After(*record.ExpiresAt) {
		return nil, bizerr.New(bizerr.CodeTokenExpired, "API key has expired")
	}

	return record, nil
}

func hashAPIKey(key string) string {
	// API keys contain 256 bits from crypto/rand, so a fast one-way lookup hash
	// is not vulnerable to the low-entropy password attacks covered by this rule.
	sum := sha256.Sum256([]byte(key)) // codeql[go/weak-sensitive-data-hashing]
	return hex.EncodeToString(sum[:])
}

// APIKeyInfo is a safe view of an API key that omits the hash.
type APIKeyInfo struct {
	ID        uint       `json:"id"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"keyPrefix"`
	ExpiresAt *time.Time `json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

// ListByUser returns all API keys for the given user without exposing hashes.
func (s *APIKeyService) ListByUser(ctx context.Context, userID uint) ([]APIKeyInfo, error) {
	records, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to list API keys")
	}

	infos := make([]APIKeyInfo, len(records))
	for i, r := range records {
		infos[i] = APIKeyInfo{
			ID:        r.ID,
			Name:      r.Name,
			KeyPrefix: r.KeyPrefix,
			ExpiresAt: r.ExpiresAt,
			CreatedAt: r.CreatedAt,
		}
	}
	return infos, nil
}

// Revoke deletes an API key after verifying that the requesting user owns it.
func (s *APIKeyService) Revoke(ctx context.Context, keyID, userID uint) error {
	keys, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to verify key ownership")
	}

	found := false
	for _, k := range keys {
		if k.ID == keyID {
			found = true
			break
		}
	}
	if !found {
		return bizerr.New(bizerr.CodeNotFound, "API key not found or not owned by user")
	}

	if err := s.repo.Delete(ctx, keyID); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to delete API key")
	}
	return nil
}
