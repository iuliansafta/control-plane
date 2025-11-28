package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType type of JWT token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// JWTClaims custom claims for JWT tokens
type JWTClaims struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations
type JWTService struct {
	secretKey            []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

// NewJWTService creates a new JWT service
// secretKey should be at least 32 bytes for HS256
func NewJWTService(secretKey string, accessTokenDuration, refreshTokenDuration time.Duration) *JWTService {
	return &JWTService{
		secretKey:            []byte(secretKey),
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
	}
}

// GenerateAccessToken generates a new JWT access token
func (s *JWTService) GenerateAccessToken(userID uuid.UUID, email, name string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    userID.String(),
		Email:     email,
		Name:      name,
		TokenType: AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "control-plane",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken generates a new random refresh token string
func (s *JWTService) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GetRefreshTokenExpiry returns the expiry time for refresh tokens
func (s *JWTService) GetRefreshTokenExpiry() time.Time {
	return time.Now().Add(s.refreshTokenDuration)
}

// ValidateToken validates and parses a JWT token
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.TokenType != AccessToken {
		return nil, fmt.Errorf("invalid token type: expected access token")
	}

	return claims, nil
}

// ExtractUserID extracts the user ID from JWT claims
func (s *JWTService) ExtractUserID(claims *JWTClaims) (uuid.UUID, error) {
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid user ID in token: %w", err)
	}
	return userID, nil
}

// IsTokenExpired checks if a token is expired
func (s *JWTService) IsTokenExpired(claims *JWTClaims) bool {
	if claims.ExpiresAt == nil {
		return true
	}
	return claims.ExpiresAt.Time.Before(time.Now())
}
