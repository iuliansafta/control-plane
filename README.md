# Control Plane

A Go-based control plane service for deploying and managing containerized applications through HashiCorp Nomad. The system provides a gRPC API for application deployment operations and includes a command-line interface for easy interaction.

## Architecture

```

```

**Components:**
- **gRPC Server**: Accepts deployment requests and translates them to Nomad job specifications
- **CLI Client**: Command-line interface for interacting with the control plane
- **Nomad Integration**: Orchestrates container deployments through HashiCorp Nomad
- **Traefik Support**: Automatic reverse proxy configuration for web applications


### Prerequisites

- **Go >1.24+** for building
- **HashiCorp Nomad** running locally or remotely
- **Docker** for container runtime
- **Devbox** (optional, for development environment)

### Installation

1. **Clone the repository:**
   ```bash
   git clone github.com/iuliansafta/control-plane
   cd control-plane
   ```

2. **Build the binaries:**
   ```bash
   make build
   ```

3. **Start Nomad** (if not already running):
   ```bash
   # Download and start Nomad in dev mode
   nomad agent -dev
   ```

4. **Start the Control Plane server:**
   ```bash
   ./bin/controller -port=50051 -nomad=http://localhost:4646
   ```

5. **Deploy your first application:**
   ```bash
   ./bin/cli -action=deploy -name=whoami -image=traefik/whoami:latest
   ```

## gRPC Service

The Control Plane exposes a gRPC service for programmatic access to deployment operations.

### Service Definition

```protobuf
service ControlPlane {
    rpc DeployApplication(DeployRequest) returns (DeployResponse);
    rpc DeleteApplication(DeleteRequest) returns (DeleteResponse);
}
```

### How to Use the gRPC Service

#### 1. Generate Client Code

The protobuf definitions are in `api/proto/controlplane.proto`. Generate client code for your language:

```bash
make proto
```

#### 2. Connect to the Service

**Go Example:**
```go
conn, err := grpc.NewClient("localhost:50051", 
    grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewControlPlaneClient(conn)
```

#### 3. Deploy Applications

**Go Example:**
```go
resp, err := client.DeployApplication(ctx, &pb.DeployRequest{
    Name:        "my-app",
    Image:       "nginx:latest",
    Replicas:    3,
    Cpu:         0.5,
    Memory:      512,
    Region:      "global",
    NetworkMode: pb.NetworkMode_NETWORK_MODE_HOST,
    Labels:      map[string]string{"env": "production"},
    Traefik: &pb.TraefikConfig{
        Enable: true,
        Host:   "myapp.local",
    },
})
```

### API Reference

#### DeployRequest

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique application name |
| `image` | string | Docker image (e.g., `nginx:latest`) |
| `replicas` | int32 | Number of instances |
| `cpu` | double | CPU cores allocation |
| `memory` | int64 | Memory in MB |
| `region` | string | Target region |
| `network_mode` | NetworkMode | Host or bridge networking |
| `labels` | map<string,string> | Environment variables |
| `traefik` | TraefikConfig | Reverse proxy configuration |

#### NetworkMode Enum

- `NETWORK_MODE_UNSPECIFIED` (0) - Defaults to host
- `NETWORK_MODE_HOST` (1) - Host networking
- `NETWORK_MODE_BRIDGE` (2) - Bridge networking (requires CNI)

#### TraefikConfig

| Field | Type | Description |
|-------|------|-------------|
| `enable` | bool | Enable Traefik integration |
| `host` | string | Hostname for routing |
| `enable_ssl` | bool | Enable HTTPS |
| `health_check_path` | string | Health check endpoint |

### How to Develop the gRPC Service

#### 1. Development Environment

Using Devbox (recommended):
```bash
devbox shell
```

Manual setup:
```bash
# Install protobuf compiler
brew install protobuf

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

#### 2. Making Changes

**Modify the API:**
1. Edit `api/proto/controlplane.proto`
2. Regenerate code: `make proto`
3. Update service implementation in `pkg/api/server.go`

**Add New Features:**
1. Update protobuf schema
2. Implement in `pkg/api/server.go`
3. Add corresponding Nomad job logic in `pkg/nomad/job.go`
4. Test with CLI or custom client

#### 3. Development Commands

```bash
# Build everything
make build

# Generate protobuf code
make proto

# Check required tools
make check-tools

# Install missing tools
make install-tools

# Run the server in development
./bin/controller -port=50051 -nomad=http://localhost:4646

# Test with CLI
./bin/cli -action=deploy -name=test -image=nginx:latest
```

## CLI

The Command-Line Interface provides an easy way to interact with the Control Plane service.

### How to Use the CLI

#### Basic Usage

```bash
./bin/cli [flags]
```

#### Global Flags

- `-server string` - gRPC server address (default: `localhost:50051`)
- `-action string` - Action to perform: `deploy`, `delete` (default: `deploy`)

#### Deploy Applications

**Basic deployment:**
```bash
./bin/cli -action=deploy -name=whoami -image=traefik/whoami:latest
```

**Production deployment with resources:**
```bash
./bin/cli -action=deploy \
  -name=webapp \
  -image=nginx:latest \
  -replicas=3 \
  -cpu=0.5 \
  -memory=512 \
  -region=euparis01
```

**Web application with Traefik:**
```bash
./bin/cli -action=deploy \
  -name=frontend \
  -image=nginx:latest \
  -traefik-host=myapp.local \
  -traefik-ssl
```

**Application with environment variables:**
```bash
./bin/cli -action=deploy \
  -name=api \
  -image=myapi:latest \
```

**Bridge networking (with CNI):**
```bash
./bin/cli -action=deploy \
  -name=service \
  -image=myservice:latest \
  -network=bridge
```

#### Delete Applications

**Delete by name:**
```bash
./bin/cli -action=delete -name=whoami
```

**Delete by deployment ID:**
```bash
./bin/cli -action=delete -delete-id=abc123-def456
```

#### Deployment Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-name` | string | `test-app` | Application name |
| `-image` | string | `traefik/whoami:latest` | Container image |
| `-replicas` | int | `1` | Number of replicas |
| `-cpu` | float | `0.1` | CPU cores |
| `-memory` | int | `128` | Memory in MB |
| `-region` | string | `global` | Target region |
| `-network` | string | `host` | Network mode (host/bridge) |
| `-env` | string | `""` | Environment variables (KEY1=VALUE1,KEY2=VALUE2) |
| `-traefik-host` | string | `""` | Enable Traefik with hostname |
| `-traefik-ssl` | bool | `false` | Enable SSL for Traefik |

### How to Develop the CLI

#### 1. Architecture

The CLI follows a clean, flag-based pattern:

```go
type DeployConfig struct {
    Name        string
    Image       string
    Replicas    int
    CPU         float64
    Memory      int64
    NetworkMode string
}

func deployApp(ctx context.Context, client pb.ControlPlaneClient, config *DeployConfig)
```

#### 2. Making Changes

**Add New Flags:**
1. Add field to appropriate config struct
2. Add flag variable in `main()`
3. Set field value when creating config
4. Update validation if needed

**Add New Commands:**
1. Add new action to switch statement
2. Create handler function
3. Update usage documentation


## Development

### Build System

The project uses a Makefile for common development tasks:

```bash
make build
make proto
make check-tools
make install-tools
```

### Project Dependencies

**Core:**
- `google.golang.org/grpc` - gRPC framework
- `github.com/hashicorp/nomad/api` - Nomad client
- `google.golang.org/protobuf` - Protocol buffers

**Development:**
- `protoc` - Protocol buffer compiler
- `protoc-gen-go` - Go protobuf plugin
- `protoc-gen-go-grpc` - Go gRPC plugin

### Environment Setup

**Using Devbox (recommended):**
```bash
devbox shell
# Provides: Go 1.25, golangci-lint, and development tools
```

### Testing Strategy

**Manual Testing:**
- Deploy applications through CLI
- Verify with Nomad Web UI: `http://localhost:4646`
- Check container status with Docker

**Integration Testing:**
- Test different network modes
- Verify Traefik integration
- Test resource allocation limits

**Load Testing:**
- Deploy multiple applications simultaneously
- Test resource exhaustion scenarios
- Verify cleanup operations

## Additional Resources

- **Nomad Documentation**: https://developer.hashicorp.com/nomad/docs
- **gRPC Go Tutorial**: https://grpc.io/docs/languages/go/
- **Protocol Buffers**: https://protobuf.dev/
- **Traefik Documentation**: https://doc.traefik.io/traefik/
