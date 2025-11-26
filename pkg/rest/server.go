package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/iuliansafta/control-plane/pkg/api"
	"github.com/iuliansafta/control-plane/pkg/auth"
)

// Server the REST API server
type Server struct {
	echo       *echo.Echo
	appService *api.ApplicationService
	authSvc    *auth.APIKeyService
	userSvc    *auth.UserService
	port       string
}

// NewServer creates a new REST API server
func NewServer(appService *api.ApplicationService, authSvc *auth.APIKeyService, userSvc *auth.UserService, port string) *Server {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	server := &Server{
		echo:       e,
		appService: appService,
		authSvc:    authSvc,
		userSvc:    userSvc,
		port:       port,
	}

	server.setupRoutes()

	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	s.echo.GET("/api/v1/health", s.healthCheck)

	// Authentication routes
	auth := s.echo.Group("/api/v1/auth")
	auth.POST("/register", s.register)
	auth.POST("/login", s.login)
	auth.POST("/refresh", s.refreshToken)
	auth.POST("/logout", s.logout)

	// Protected auth routes
	auth.GET("/me", s.getCurrentUser, s.jwtOnlyMiddleware)

	// API v1 routes (authenticated with either JWT or API key)
	v1 := s.echo.Group("/api/v1")
	v1.Use(s.authMiddleware)

	// Application operations (accept both JWT and API key)
	v1.POST("/applications", s.deployApplication)
	v1.DELETE("/applications/:id", s.deleteApplication)
	v1.GET("/applications/:id/status", s.getApplicationStatus)

	// User API key management (require JWT only)
	userKeys := v1.Group("/user/keys")
	userKeys.Use(s.jwtOnlyMiddleware)
	userKeys.POST("", s.createUserAPIKey)
	userKeys.GET("", s.listUserAPIKeys)
	userKeys.DELETE("/:id", s.deleteUserAPIKey)

	// Admin operations (accept both JWT and API key, but should check permissions)
	admin := v1.Group("/admin")
	admin.POST("/keys", s.createAPIKey)
	admin.GET("/keys", s.listAPIKeys)
	admin.DELETE("/keys/:id", s.deleteAPIKey)
}

// Start starts the REST API server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.port)
	return s.echo.Start(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
