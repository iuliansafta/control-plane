package rest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	RefreshTokenCookieName = "refresh_token"
	RefreshTokenMaxAge     = 7 * 24 * 60 * 60
	UserContextKey         = "user_id"
)

// parseUserID parses a user ID string into a UUID
func parseUserID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid UUID: %w", err)
	}
	return parsed, nil
}

// RegisterRequest JSON request for user registration
type RegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterResponse JSON response for user registration
type RegisterResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     string `json:"created_at"`
}

// LoginRequest JSON request for user login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse JSON response for user login
type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int      `json:"expires_in"`
	User         UserInfo `json:"user"`
}

// RefreshTokenRequest JSON request for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

// RefreshTokenResponse JSON response for token refresh
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// UserInfo represents user information in responses
type UserInfo struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Email         string  `json:"email"`
	EmailVerified bool    `json:"email_verified"`
	IsActive      bool    `json:"is_active"`
	LastLogin     *string `json:"last_login,omitempty"`
}

// register handles POST /api/v1/auth/register
func (s *Server) register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "name, email, and password are required",
		})
	}

	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "password must be at least 8 characters",
		})
	}

	user, err := s.userSvc.RegisterUser(c.Request().Context(), req.Name, req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, RegisterResponse{
		ID:            user.ID.String(),
		Name:          user.Name,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format(time.RFC3339),
	})
}

// login handles POST /api/v1/auth/login
func (s *Server) login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "email and password are required",
		})
	}

	accessToken, refreshToken, user, err := s.userSvc.LoginUser(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	if user.EmailVerified {
		cookie := &http.Cookie{
			Name:     RefreshTokenCookieName,
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   RefreshTokenMaxAge,
		}
		c.SetCookie(cookie)
	}

	var lastLogin *string
	if user.LastLogin != nil {
		formatted := user.LastLogin.Format(time.RFC3339)
		lastLogin = &formatted
	}

	response := LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
		User: UserInfo{
			ID:            user.ID.String(),
			Name:          user.Name,
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
			IsActive:      user.IsActive,
			LastLogin:     lastLogin,
		},
	}

	if !user.EmailVerified {
		response.RefreshToken = refreshToken
	}

	return c.JSON(http.StatusOK, response)
}

// refreshToken handles POST /api/v1/auth/refresh
func (s *Server) refreshToken(c echo.Context) error {
	var refreshToken string

	cookie, err := c.Cookie(RefreshTokenCookieName)
	if err == nil && cookie.Value != "" {
		refreshToken = cookie.Value
	} else {
		var req RefreshTokenRequest
		if err := c.Bind(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "refresh token is required",
		})
	}

	accessToken, newRefreshToken, err := s.userSvc.RefreshAccessToken(c.Request().Context(), refreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	if cookie != nil {
		newCookie := &http.Cookie{
			Name:     RefreshTokenCookieName,
			Value:    newRefreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   RefreshTokenMaxAge,
		}
		c.SetCookie(newCookie)

		return c.JSON(http.StatusOK, RefreshTokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   900,
		})
	}

	return c.JSON(http.StatusOK, RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes
	})
}

// logout handles POST /api/v1/auth/logout
func (s *Server) logout(c echo.Context) error {
	var refreshToken string

	cookie, err := c.Cookie(RefreshTokenCookieName)
	if err == nil && cookie.Value != "" {
		refreshToken = cookie.Value
	} else {
		var req RefreshTokenRequest
		if err := c.Bind(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken != "" {
		if err := s.userSvc.LogoutUser(c.Request().Context(), refreshToken); err != nil {
			c.Logger().Errorf("Failed to revoke refresh token: %v", err)
		}
	}

	cookie = &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// getCurrentUser handles GET /api/v1/auth/me (requires authentication)
func (s *Server) getCurrentUser(c echo.Context) error {
	userID, ok := c.Get(UserContextKey).(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	uuid, err := parseUserID(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID",
		})
	}

	user, err := s.userSvc.GetUserByID(c.Request().Context(), uuid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get user",
		})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
	}

	var lastLogin *string
	if user.LastLogin != nil {
		formatted := user.LastLogin.Format(time.RFC3339)
		lastLogin = &formatted
	}

	return c.JSON(http.StatusOK, UserInfo{
		ID:            user.ID.String(),
		Name:          user.Name,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		IsActive:      user.IsActive,
		LastLogin:     lastLogin,
	})
}
