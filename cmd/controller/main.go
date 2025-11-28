package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/iuliansafta/control-plane/api/proto"
	"github.com/iuliansafta/control-plane/pkg/api"
	"github.com/iuliansafta/control-plane/pkg/auth"
	"github.com/iuliansafta/control-plane/pkg/config"
	"github.com/iuliansafta/control-plane/pkg/database"
	"github.com/iuliansafta/control-plane/pkg/nomad"
	"github.com/iuliansafta/control-plane/pkg/rest"
	"google.golang.org/grpc"
)

// @title Control Plane API
// @version 1.0
// @description API for the Control Plane service
// @host localhost:8080
// @BasePath /api/v1
func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize Nomad client
	nomadClient, err := nomad.NewNomadClient(cfg.NomadAddr)
	if err != nil {
		log.Fatalf("Failed to create Nomad client: %v", err)
	}

	if cfg.DBConnection == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := database.New(cfg.DBConnection)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	apiKeyRepo := database.NewAPIKeyRepository(db)
	userRepo := database.NewUserRepository(db)
	refreshTokenRepo := database.NewRefreshTokenRepository(db)

	// Initialize services
	apiKeySvc := auth.NewAPIKeyService(apiKeyRepo)

	// JWT secret validation
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required for user authentication")
	}
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long")
	}

	// Initialize JWT service with token durations
	jwtService := auth.NewJWTService(
		cfg.JWTSecret,
		15*time.Minute, // Access token: 15 minutes
		7*24*time.Hour, // Refresh token: 7 days
	)

	// Initialize user service
	userSvc := auth.NewUserService(userRepo, refreshTokenRepo, jwtService)

	// Bootstrap initial API key if requested
	if cfg.BootstrapKey != "" {
		count, err := apiKeySvc.KeyCount()
		if err != nil {
			log.Fatalf("Failed to check API key count: %v", err)
		}

		if count == 0 {
			plainKey, apiKey, err := apiKeySvc.CreateKey(cfg.BootstrapKey)
			if err != nil {
				log.Fatalf("Failed to create bootstrap API key: %v", err)
			}
			log.Printf("Bootstrap API key created:")
			log.Printf("  Name: %s", apiKey.Name)
			log.Printf("  ID: %s", apiKey.ID)
			log.Printf("  Key: %s", plainKey)
			log.Println("  (Save this key - it will not be shown again)")
		} else {
			log.Println("API keys already exist, skipping bootstrap")
		}
	}

	// Init gRPC service with Nomad client
	appService := api.NewApplicationService(nomadClient)

	// Create gRPC listener
	grpcListener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	// Create the gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterControlPlaneServer(grpcServer, appService)

	// Create REST server
	restServer := rest.NewServer(appService, apiKeySvc, userSvc, cfg.HttpPort)

	// Start gRPC server
	go func() {
		log.Printf("Starting gRPC server on :%s (nomad: %s)", cfg.GRPCPort, cfg.NomadAddr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Start REST server
	go func() {
		log.Printf("Starting REST server on :%s", cfg.HttpPort)
		if err := restServer.Start(); err != nil {
			log.Printf("REST server stopped: %v", err)
		}
	}()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown REST server
	if err := restServer.Shutdown(ctx); err != nil {
		log.Printf("REST server shutdown error: %v", err)
	}

	// Stop gRPC server
	grpcServer.GracefulStop()

	log.Println("Servers stopped")
}
