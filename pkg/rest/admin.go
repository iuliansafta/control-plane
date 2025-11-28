package rest

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// CreateKeyRequest represents the JSON request for creating an API key
type CreateKeyRequest struct {
	Name string `json:"name"`
}

// CreateKeyResponse represents the JSON response for creating an API key
type CreateKeyResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"` // Returned once on creation
	CreatedAt string `json:"created_at"`
}

// APIKeyInfo represents an API key in list responses
type APIKeyInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

// createAPIKey handles POST /api/v1/admin/keys
// @Summary Create an API key
// @Description Create a new API key (Admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security ApiKeyAuth
// @Param request body CreateKeyRequest true "Create Key Request"
// @Success 201 {object} CreateKeyResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/keys [post]
func (s *Server) createAPIKey(c echo.Context) error {
	var req CreateKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "name is required",
		})
	}

	// Create the API key
	plainKey, apiKey, err := s.authSvc.CreateKey(req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, CreateKeyResponse{
		ID:        apiKey.ID.String(),
		Name:      apiKey.Name,
		Key:       plainKey,
		CreatedAt: apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// listAPIKeys handles GET /api/v1/admin/keys
// @Summary List API keys
// @Description List all API keys (Admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security ApiKeyAuth
// @Success 200 {object} []APIKeyInfo
// @Failure 500 {object} map[string]string
// @Router /admin/keys [get]
func (s *Server) listAPIKeys(c echo.Context) error {
	keys, err := s.authSvc.ListKeys()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	result := make([]APIKeyInfo, len(keys))
	for i, key := range keys {
		result[i] = APIKeyInfo{
			ID:        key.ID.String(),
			Name:      key.Name,
			IsActive:  key.IsActive,
			CreatedAt: key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, result)
}

// deleteAPIKey handles DELETE /api/v1/admin/keys/:id
// @Summary Delete an API key
// @Description Delete an API key by ID (Admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security ApiKeyAuth
// @Param id path string true "Key ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/keys/{id} [delete]
func (s *Server) deleteAPIKey(c echo.Context) error {
	keyID := c.Param("id")
	if keyID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "key id is required",
		})
	}

	err := s.authSvc.DeleteKey(keyID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "API key deleted successfully",
	})
}
