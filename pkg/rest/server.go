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
	port       string
}

// NewServer creates a new REST API server
func NewServer(appService *api.ApplicationService, authSvc *auth.APIKeyService, port string) *Server {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	server := &Server{
		echo:       e,
		appService: appService,
		authSvc:    authSvc,
		port:       port,
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check (public)
	s.echo.GET("/api/v1/health", s.healthCheck)

	// API v1 routes (authenticated)
	v1 := s.echo.Group("/api/v1")
	v1.Use(s.authMiddleware)

	// Application operations
	v1.POST("/applications", s.deployApplication)
	v1.DELETE("/applications/:id", s.deleteApplication)
	v1.GET("/applications/:id/status", s.getApplicationStatus)

	// Admin operations
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
