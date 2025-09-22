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
	EnvVars     string
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
		action      = flag.String("action", "usage", "Action: deploy, delete")
		name        = flag.String("name", "", "Application name")
		image       = flag.String("image", "", "Container image")
		replicas    = flag.Int("replicas", 1, "Number of replicas")
		cpu         = flag.Float64("cpu", 0.1, "CPU cores")
		memory      = flag.Int64("memory", 128, "Memory in MB")
		region      = flag.String("region", "global", "Target region")
		networkMode = flag.String("network", "host", "Network mode: host, bridge")
		envVars     = flag.String("env", "", "Environment variables (KEY1=VALUE1,KEY2=VALUE2)")
		traefikHost = flag.String("traefik-host", "", "Enable Traefik with hostname")
		traefikSSL  = flag.Bool("traefik-ssl", false, "Enable SSL for Traefik")
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
			EnvVars:     *envVars,
			TraefikHost: *traefikHost,
			TraefikSSL:  *traefikSSL,
		}
		deployApp(ctx, client, config)
	case "delete":
		deleteApp(ctx, client, *deleteId, *name)
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

	// environment := make(map[string]string)
	// if config.EnvVars != "" {
	// 	pairs := strings.Split(config.EnvVars, ",")
	// 	for _, pair := range pairs {
	// 		parts := strings.SplitN(pair, "=", 2)
	// 		if len(parts) != 2 {
	// 			log.Fatalf("Invalid environment variable format: %s (should be KEY=VALUE)", pair)
	// 		}
	// 		environment[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	// 	}
	// }

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
		// Labels:      environment,
		Traefik: traefikConfig,
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

	if resp.Success {
		fmt.Printf("%s\n", resp.Message)
	} else {
		fmt.Printf("%s\n", resp.Message)
	}
}

func printUsage() {
	fmt.Println("Control Plane CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cli [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -server string         gRPC server address (default: localhost:50051)")
	fmt.Println("  -action string         Action: deploy, delete (default: deploy)")
	fmt.Println("  -name string           Application name (default: test-app)")
	fmt.Println("  -image string          Container image (default: traefik/whoami:latest)")
	fmt.Println("  -replicas int          Number of replicas (default: 1)")
	fmt.Println("  -cpu float             CPU cores (default: 0.1)")
	fmt.Println("  -memory int            Memory in MB (default: 128)")
	fmt.Println("  -region string         Target region (default: global)")
	fmt.Println("  -network string        Network mode: host, bridge (default: host)")
	fmt.Println("  -env string            Environment variables (KEY1=VALUE1,KEY2=VALUE2)")
	fmt.Println("  -traefik-host string   Enable Traefik with hostname")
	fmt.Println("  -traefik-ssl           Enable SSL for Traefik")
	fmt.Println("  -delete-id string      Deployment ID to delete (for delete action)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println()
	fmt.Println("  # Deploy with custom resources")
	fmt.Println("  cli -action=deploy -name=webapp -image=nginx:latest -replicas=3 -cpu=0.5 -memory=512")
	fmt.Println()
	fmt.Println("  # Deploy with Traefik")
	fmt.Println("  cli -action=deploy -name=webapp -image=nginx:latest -traefik-host=webapp.local -traefik-ssl")
	fmt.Println()
	fmt.Println("  # Deploy with environment variables")
	fmt.Println("  cli -action=deploy -name=app -image=myapp:latest -env=\"ENV=production,DEBUG=false\"")
	fmt.Println()
	fmt.Println("  # Delete application")
	fmt.Println("  cli -action=delete -name=whoami")
	fmt.Println()
	fmt.Println("  # Delete by deployment ID")
	fmt.Println("  cli -action=delete -delete-id=abc123-def456")
}
