package api

import (
	"context"
	"fmt"
	"maps"

	pb "github.com/iuliansafta/iulian-cloud-controller/api/proto"
	"github.com/iuliansafta/iulian-cloud-controller/pkg/nomad"
	"github.com/iuliansafta/iulian-cloud-controller/pkg/utils"
)

type ApplicationService struct {
	pb.UnimplementedControlPlaneServer
	orhClient *nomad.NomadClient //INFO: this could be extended to handle multiple orchestrators
}

func NewApplicationService(orchClient *nomad.NomadClient) *ApplicationService {
	return &ApplicationService{
		orhClient: orchClient,
	}
}

func (s *ApplicationService) DeployApplication(ctx context.Context, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	jobTemplate := &nomad.JobTemplate{
		Name:      req.Name,
		Image:     req.Image,
		Instances: int(req.Replicas),
		ResourcesSpec: nomad.Resources{
			CPU:      utils.IntPtr(int(req.Cpu * 100)),
			MemoryMB: utils.IntPtr(int(req.Memory)),
		},
		Environment: make(map[string]string),
	}

	if req.Traefik != nil {
		jobTemplate.Traefik = nomad.TraefikSpec{
			Enable:              req.Traefik.Enable,
			Host:                req.Traefik.Host,
			Entrypoint:          req.Traefik.Entrypoint,
			EnableSSL:           req.Traefik.EnableSsl,
			SSLHost:             req.Traefik.SslHost,
			CertResolver:        req.Traefik.CertResolver,
			HealthCheckPath:     req.Traefik.HealthCheckPath,
			HealthCheckInterval: req.Traefik.HealthCheckInterval,
			PathPrefix:          req.Traefik.PathPrefix,
			Middlewares:         req.Traefik.Middlewares,
			CustomLabels:        req.Traefik.CustomLabels,
		}
	}

	maps.Copy(jobTemplate.Environment, req.Labels)

	if jobTemplate.Ports.Label == "" {
		jobTemplate.Ports = nomad.Ports{
			Label: "http",
			Value: 0, // dynamic port from nomad
			To:    80,
		}
	}

	resp, err := s.orhClient.DeployJob(jobTemplate)
	if err != nil {
		return &pb.DeployResponse{
			Status:  "FAILED",
			Message: fmt.Sprintf("Failed to deploy application: %v", err),
		}, nil
	}

	return &pb.DeployResponse{
		DeploymentId: resp.EvalID,
		Status:       "SUBMITTED",
		Message:      "Application deployment submitted successfully",
	}, nil
}

func (s *ApplicationService) DeleteApplication(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := s.orhClient.DeleteJob(req.DeploymentId)
	if err != nil {
		return &pb.DeleteResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete application: %v", err),
		}, nil
	}

	return &pb.DeleteResponse{
		Success: true,
		Message: "Application deleted successfully",
	}, nil
}
