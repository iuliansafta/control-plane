package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/iuliansafta/control-plane/api/proto"
)

type DeployConfig struct {
	Name        string
	Image       string
	Replicas    int
	CPU         float64
	Memory      int64
	Region      string
	NetworkMode string
	TraefikHost string
	TraefikSSL  bool
}

func (c *DeployConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if c.Image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if c.Replicas < 1 {
		return fmt.Errorf("replicas must be at least 1")
	}
	if c.CPU <= 0 {
		return fmt.Errorf("cpu must be greater than 0")
	}
	if c.Memory <= 0 {
		return fmt.Errorf("memory must be greater than 0")
	}
	if c.NetworkMode != "host" && c.NetworkMode != "bridge" {
		return fmt.Errorf("network mode must be 'host' or 'bridge'")
	}
	return nil
}

func main() {
	var (
		server      = flag.String("server", "localhost:50051", "gRPC server address")
		action      = flag.String("action", "", "Action: deploy, delete, status")
		name        = flag.String("name", "", "Application name")
		image       = flag.String("image", "", "Container image")
		replicas    = flag.Int("replicas", 1, "Number of replicas")
		cpu         = flag.Float64("cpu", 0.1, "CPU cores")
		memory      = flag.Int64("memory", 128, "Memory in MB")
		region      = flag.String("region", "global", "Target region")
		networkMode = flag.String("network", "host", "Network mode: host, bridge")
		traefikHost = flag.String("host", "", "Enable Traefik with hostname")
		traefikSSL  = flag.Bool("ssl", false, "Enable SSL for Traefik")
		deleteId    = flag.String("delete-id", "", "Deployment ID to delete (for delete action)")
	)
	flag.Parse()

	// Connect to gRPC server
	conn, err := grpc.NewClient(*server, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewControlPlaneClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch *action {
	case "deploy":
		config := &DeployConfig{
			Name:        *name,
			Image:       *image,
			Replicas:    *replicas,
			CPU:         *cpu,
			Memory:      *memory,
			Region:      *region,
			NetworkMode: *networkMode,
			TraefikHost: *traefikHost,
			TraefikSSL:  *traefikSSL,
		}
		deployApp(ctx, client, config)
	case "delete":
		deleteApp(ctx, client, *deleteId, *name)
	case "status":
		getStatus(ctx, client, *name)
	default:
		fmt.Printf("Unknown action: %s\n", *action)
		printUsage()
	}
}

func deployApp(ctx context.Context, client pb.ControlPlaneClient, config *DeployConfig) {
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	var networkMode pb.NetworkMode
	switch config.NetworkMode {
	case "host":
		networkMode = pb.NetworkMode_NETWORK_MODE_HOST
	case "bridge":
		networkMode = pb.NetworkMode_NETWORK_MODE_BRIDGE
	default:
		log.Fatalf("Invalid network mode: %s (must be 'host' or 'bridge')", config.NetworkMode)
	}

	// Configure Traefik if host is provided
	var traefikConfig *pb.TraefikConfig
	if config.TraefikHost != "" {
		traefikConfig = &pb.TraefikConfig{
			Enable:              true,
			Host:                config.TraefikHost,
			Entrypoint:          "websecure",
			EnableSsl:           config.TraefikSSL,
			HealthCheckPath:     "/",
			HealthCheckInterval: "30s",
		}
	}

	req := &pb.DeployRequest{
		Name:        config.Name,
		Image:       config.Image,
		Replicas:    int32(config.Replicas),
		Cpu:         config.CPU,
		Memory:      config.Memory,
		Region:      config.Region,
		NetworkMode: networkMode,
		Traefik:     traefikConfig,
	}

	fmt.Printf("Deploying application '%s' with image '%s'...\n", config.Name, config.Image)
	resp, err := client.DeployApplication(ctx, req)
	if err != nil {
		log.Fatalf("Deployment failed: %v", err)
	}

	fmt.Printf("Deployment successful!\n")
	fmt.Printf("ID: %s\n", resp.DeploymentId)
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Message: %s\n", resp.Message)
}

func deleteApp(ctx context.Context, client pb.ControlPlaneClient, deleteId, name string) {
	targetId := deleteId
	if targetId == "" {
		targetId = name
	}

	if targetId == "" {
		log.Fatalf("-delete-id or -name must be provided for delete action")
	}

	req := &pb.DeleteRequest{
		DeploymentId: targetId,
	}

	fmt.Printf("Deleting application with ID '%s'...\n", targetId)
	resp, err := client.DeleteApplication(ctx, req)
	if err != nil {
		log.Fatalf("Failed to delete application: %v", err)
	}

	fmt.Printf("%s\n", resp.Message)
}

func getStatus(ctx context.Context, client pb.ControlPlaneClient, name string) {
	if name == "" {
		log.Fatalf("-name must be provided for get deployment status")
	}

	req := &pb.StatusRequest{
		DeploymentId: name,
	}

	resp, err := client.GetApplicationStatus(ctx, req)
	if err != nil {
		log.Fatalf("Failed to get application status: %v", err)
	}

	fmt.Printf("\nApplication: %s\n", resp.DeploymentId)
	fmt.Printf("Status: %s\n", resp.JobStatus)
	fmt.Printf("Type: %s\n", resp.JobType)
	fmt.Printf("Instances: %d/%d running\n", resp.RunningInstances, resp.DesiredInstances)

	if len(resp.Allocations) > 0 {
		fmt.Printf("\nAllocations:\n")
		for _, alloc := range resp.Allocations {
			allocID := alloc.AllocationId
			if len(allocID) > 8 {
				allocID = allocID[:8]
			}
			fmt.Printf("  - %s on %s: %s\n", allocID, alloc.NodeName, alloc.Status)
		}
	}
	fmt.Printf("\nMessage: %s\n\n", resp.Message)
}

func printUsage() {
	fmt.Println("Control Plane CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cli [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -server string         gRPC server address (default: localhost:50051)")
	fmt.Println("  -action string         Action: deploy, delete, status")
	fmt.Println("  -name string           Application name")
	fmt.Println("  -image string          Container image")
	fmt.Println("  -replicas int          Number of replicas (default: 1)")
	fmt.Println("  -cpu float             CPU cores (default: 0.1)")
	fmt.Println("  -memory int            Memory in MB (default: 128)")
	fmt.Println("  -region string         Target region (default: global)")
	fmt.Println("  -network string        Network mode: host, bridge (default: host)")
	fmt.Println("  -host string   		  Enable Traefik with hostname")
	fmt.Println("  -ssl           		  Enable SSL for Traefik")
	fmt.Println("  -delete-id string      Deployment ID to delete (for delete action)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println()
	fmt.Println("  # Deploy application")
	fmt.Println("  cli -action=deploy -name=webapp -image=nginx:latest -replicas=2")
	fmt.Println()
	fmt.Println("  # Get application status")
	fmt.Println("  cli -action=status -name=webapp")
	fmt.Println()
	fmt.Println("  # Delete application")
	fmt.Println("  cli -action=delete -name=webapp")
}
