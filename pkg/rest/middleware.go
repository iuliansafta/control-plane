package rest

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	APIKeyHeader     = "X-API-Key"
	APIKeyContextKey = "api_key"
	AuthTypeKey      = "auth_type"
)

// authMiddleware validates both API key and JWT authentication
func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate JWT token
			claims, err := s.userSvc.ValidateAccessToken(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid or expired token",
				})
			}

			c.Set(UserContextKey, claims.UserID)
			c.Set(AuthTypeKey, "jwt")

			return next(c)
		}

		apiKey := c.Request().Header.Get(APIKeyHeader)
		if apiKey != "" {
			keyInfo, err := s.authSvc.ValidateKey(apiKey)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid or inactive API key",
				})
			}

			c.Set(APIKeyContextKey, keyInfo)
			c.Set(AuthTypeKey, "api_key")

			if keyInfo.UserID != nil {
				c.Set(UserContextKey, keyInfo.UserID.String())
			}

			return next(c)
		}

		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "missing authentication credentials",
		})
	}
}

// jwtOnlyMiddleware requires JWT authentication only (for user-specific endpoints)
func (s *Server) jwtOnlyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "missing or invalid authorization header",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := s.userSvc.ValidateAccessToken(token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "invalid or expired token",
			})
		}

		c.Set(UserContextKey, claims.UserID)
		c.Set(AuthTypeKey, "jwt")

		return next(c)
	}
}
