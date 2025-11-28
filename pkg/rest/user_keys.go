package rest

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// CreateUserKeyRequest represents the JSON request for creating a user API key
type CreateUserKeyRequest struct {
	Name string `json:"name" validate:"required"`
}

// CreateUserKeyResponse represents the JSON response for creating a user API key
type CreateUserKeyResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"` // Returned once on creation
	CreatedAt string `json:"created_at"`
	Message   string `json:"message"`
}

// UserAPIKeyInfo represents a user API key in list responses
type UserAPIKeyInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

// createUserAPIKey handles POST /api/v1/user/keys (requires JWT authentication)
// @Summary Create a user API key
// @Description Create a new API key for the current user
// @Tags user_keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserKeyRequest true "Create User Key Request"
// @Success 201 {object} CreateUserKeyResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/keys [post]
func (s *Server) createUserAPIKey(c echo.Context) error {
	// Get user ID from context (set by JWT middleware)
	userIDStr, ok := c.Get(UserContextKey).(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	userID, err := parseUserID(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID",
		})
	}

	var req CreateUserKeyRequest
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

	// Create the API key for the user
	plainKey, apiKey, err := s.authSvc.CreateKeyForUser(c.Request().Context(), userID, req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, CreateUserKeyResponse{
		ID:        apiKey.ID.String(),
		Name:      apiKey.Name,
		Key:       plainKey,
		CreatedAt: apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Message:   "Store this API key securely. It will not be shown again.",
	})
}

// listUserAPIKeys handles GET /api/v1/user/keys (requires JWT authentication)
// @Summary List user API keys
// @Description List all API keys for the current user
// @Tags user_keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} []UserAPIKeyInfo
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/keys [get]
func (s *Server) listUserAPIKeys(c echo.Context) error {
	// Get user ID from context (set by JWT middleware)
	userIDStr, ok := c.Get(UserContextKey).(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	userID, err := parseUserID(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID",
		})
	}

	// Get user's API keys
	keys, err := s.authSvc.ListKeysByUser(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	result := make([]UserAPIKeyInfo, len(keys))
	for i, key := range keys {
		result[i] = UserAPIKeyInfo{
			ID:        key.ID.String(),
			Name:      key.Name,
			IsActive:  key.IsActive,
			CreatedAt: key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, result)
}

// deleteUserAPIKey handles DELETE /api/v1/user/keys/:id (requires JWT authentication)
// @Summary Delete a user API key
// @Description Delete a user API key by ID
// @Tags user_keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Key ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/keys/{id} [delete]
func (s *Server) deleteUserAPIKey(c echo.Context) error {
	// Get user ID from context (set by JWT middleware)
	userIDStr, ok := c.Get(UserContextKey).(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	userID, err := parseUserID(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID",
		})
	}

	keyID := c.Param("id")
	if keyID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "key id is required",
		})
	}

	keyUUID, err := parseUserID(keyID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid key id",
		})
	}

	// Get the key to verify ownership
	apiKey, err := s.authSvc.GetKeyByID(keyUUID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get API key",
		})
	}

	if apiKey == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "API key not found",
		})
	}

	// Verify the key belongs to the user
	if apiKey.UserID == nil || *apiKey.UserID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "you do not have permission to delete this API key",
		})
	}

	// Delete the key
	err = s.authSvc.DeleteKey(keyID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "API key deleted successfully",
	})
}
