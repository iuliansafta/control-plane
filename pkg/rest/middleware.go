package rest

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	APIKeyHeader     = "X-API-Key"
	APIKeyContextKey = "api_key"
)

// authMiddleware validates API key authentication
func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		apiKey := c.Request().Header.Get(APIKeyHeader)
		if apiKey == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "missing API key",
			})
		}

		keyInfo, err := s.authSvc.ValidateKey(apiKey)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "invalid or inactive API key",
			})
		}

		c.Set(APIKeyContextKey, keyInfo)

		return next(c)
	}
}
