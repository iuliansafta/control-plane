package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/iuliansafta/iulian-cloud-controller/api/proto"
	"github.com/iuliansafta/iulian-cloud-controller/pkg/api"
	"google.golang.org/grpc"
)

var (
	grpcPort     = flag.String("port", "50051", "gRPC service port")
	nomadAddress = flag.String("nomad", "", "Nomad server address")
)

func main() {
	flag.Parse()

	// Init gRPC service
	apiServer := api.NewServer()

	// Create listener
	listener, err := net.Listen("tcp", ":"+*grpcPort)
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
		log.Printf("Starting gRPC server on :%s", *grpcPort)
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
