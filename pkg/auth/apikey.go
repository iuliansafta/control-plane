package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/iuliansafta/control-plane/pkg/database"
)

// APIKeyService handles API key operations
type APIKeyService struct {
	repo *database.APIKeyRepository
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(repo *database.APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// GenerateKey generates a new random API key
func GenerateKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashKey hashes an API key using SHA-256
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CreateKey creates a new API key with the given name
// Returns the plaintext key (show once) and the created API key record
func (s *APIKeyService) CreateKey(name string) (string, *database.APIKey, error) {
	plainKey, err := GenerateKey()
	if err != nil {
		return "", nil, err
	}

	keyHash := HashKey(plainKey)

	apiKey, err := s.repo.Create(name, keyHash)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return plainKey, apiKey, nil
}

// ValidateKey validates an API key and returns the key record if valid
func (s *APIKeyService) ValidateKey(plainKey string) (*database.APIKey, error) {
	if plainKey == "" {
		return nil, fmt.Errorf("empty API key")
	}

	keyHash := HashKey(plainKey)

	apiKey, err := s.repo.GetByHash(keyHash)
	if err != nil {
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	if apiKey == nil {
		return nil, fmt.Errorf("invalid API key")
	}

	if !apiKey.IsActive {
		return nil, fmt.Errorf("API key is inactive")
	}

	return apiKey, nil
}

// ListKeys returns all API keys
func (s *APIKeyService) ListKeys() ([]database.APIKey, error) {
	return s.repo.List()
}

// DeleteKey deletes an API key by ID
func (s *APIKeyService) DeleteKey(id string) error {
	uuid, err := parseUUID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(uuid)
}

// DeactivateKey deactivates an API key by ID
func (s *APIKeyService) DeactivateKey(id string) error {
	uuid, err := parseUUID(id)
	if err != nil {
		return err
	}
	return s.repo.Deactivate(uuid)
}

// KeyCount returns the total number of API keys
func (s *APIKeyService) KeyCount() (int, error) {
	return s.repo.Count()
}

func parseUUID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid UUID: %w", err)
	}
	return parsed, nil
}
