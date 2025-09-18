package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/iuliansafta/control-plane/api/proto"
	"github.com/iuliansafta/control-plane/pkg/api"
	"github.com/iuliansafta/control-plane/pkg/nomad"
	"google.golang.org/grpc"
)

var (
	grpcPort     = flag.String("port", "50051", "gRPC service port")
	nomadAddress = flag.String("nomad", "", "Nomad server address")
)

func main() {
	flag.Parse()

	nomadAddr := *nomadAddress
	if nomadAddr == "" {
		if envAddr := os.Getenv("NOMAD_ADDR"); envAddr != "" {
			nomadAddr = envAddr
		} else {
			nomadAddr = "http://localhost:4646"
		}
	}

	port := *grpcPort
	if port == "50051" {
		if envPort := os.Getenv("GRPC_PORT"); envPort != "" {
			port = envPort
			log.Printf("Using gRPC port from environment: %s", port)
		}
	}

	// Initialize Nomad client
	nomadClient, err := nomad.NewNomadClient(nomadAddr)
	if err != nil {
		log.Fatalf("Failed to create Nomad client: %v", err)
	}

	// Init gRPC service with Nomad client
	apiServer := api.NewApplicationService(nomadClient)

	// Create listener
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create the gRPC service
	grpcServer := grpc.NewServer()
	pb.RegisterControlPlaneServer(grpcServer, apiServer)

	// go func() {
	// 	log.Printf("Starting metrics server on :%s", *metricsPort)
	// 	if err := http.ListenAndServe(":"+*metricsPort, nil); err != nil {
	// 		log.Printf("Metrics server error: %v", err)
	// 	}
	// }()

	// Start gRPC server
	go func() {
		log.Printf("Starting gRPC server on :%s", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	grpcServer.GracefulStop()
}
