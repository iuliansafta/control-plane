package rest

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"os"

	"github.com/go-logr/stdr"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/iuliansafta/control-plane/pkg/api"
	"github.com/iuliansafta/control-plane/pkg/auth"
)

// Server the REST API server
type Server struct {
	echo           *echo.Echo
	appService     *api.ApplicationService
	authSvc        *auth.APIKeyService
	userSvc        *auth.UserService
	port           string
	tracerProvider *sdktrace.TracerProvider
}

// NewServer creates a new REST API server
func NewServer(appService *api.ApplicationService, authSvc *auth.APIKeyService, userSvc *auth.UserService, port string) *Server {
	tp := initTracer()

	e := echo.New()
	e.HideBanner = true

	e.Use(otelecho.Middleware("vorhash-control-plane"))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	server := &Server{
		echo:           e,
		appService:     appService,
		authSvc:        authSvc,
		userSvc:        userSvc,
		port:           port,
		tracerProvider: tp,
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
	if err := s.tracerProvider.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down tracer provider: %v", err)
	}
	return s.echo.Shutdown(ctx)
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// initTracer initializes the OpenTelemetry tracer
func initTracer() *sdktrace.TracerProvider {
	otel.SetLogger(stdr.New(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)))

	ctx := context.Background()
	otlpExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithTLSClientConfig(&tls.Config{InsecureSkipVerify: true}),
	)
	if err != nil {
		log.Fatal(err)
	}

	stdoutExporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(otlpExporter),
		sdktrace.WithBatcher(stdoutExporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}
