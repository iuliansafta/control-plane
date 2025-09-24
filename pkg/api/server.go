package api

import (
	"context"
	"fmt"
	"maps"
	"time"

	pb "github.com/iuliansafta/control-plane/api/proto"
	"github.com/iuliansafta/control-plane/pkg/nomad"
	"github.com/iuliansafta/control-plane/pkg/utils"
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

// DeployApplication deploys an application to the orchestrator
func (s *ApplicationService) DeployApplication(ctx context.Context, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	networkMode := "host"
	switch req.NetworkMode {
	case pb.NetworkMode_NETWORK_MODE_BRIDGE:
		networkMode = "bridge"
	case pb.NetworkMode_NETWORK_MODE_HOST:
		networkMode = "host"
	default:
		networkMode = "host"
	}

	jobTemplate := &nomad.JobTemplate{
		Name:          req.Name,
		Image:         req.Image,
		Instances:     int(req.Replicas),
		Region:        req.Region,
		DisableConsul: false,
		NetworkMode:   networkMode,
		ResourcesSpec: nomad.Resources{
			CPU:      utils.IntPtr(int(req.Cpu * 10)),
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

// DeleteApplication deletes an application.
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

// GetApplicationStatus retrieves the status of an application.
func (s *ApplicationService) GetApplicationStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	job, allocations, err := s.orhClient.GetJobStatus(req.DeploymentId)

	if err != nil {
		return &pb.StatusResponse{
			DeploymentId: req.DeploymentId,
			Message:      fmt.Sprintf("Failed to get application status: %v", err),
		}, nil
	}

	var allocationStatuses []*pb.AllocationStatus
	runningInstances := int32(0)

	for _, alloc := range allocations {
		taskStates := make(map[string]string)
		if alloc.TaskStates != nil {
			for taskName, taskState := range alloc.TaskStates {
				taskStates[taskName] = taskState.State
			}
		}

		if alloc.ClientStatus == "running" {
			runningInstances++
		}

		allocationStatus := &pb.AllocationStatus{
			AllocationId:  alloc.ID,
			NodeId:        alloc.NodeID,
			NodeName:      alloc.NodeName,
			Status:        alloc.ClientStatus,
			DesiredStatus: alloc.DesiredStatus,
			CreateTime:    alloc.CreateTime,
			ModifyTime:    alloc.ModifyTime,
			TaskStates:    taskStates,
		}
		allocationStatuses = append(allocationStatuses, allocationStatus)
	}

	desiredInstances := int32(0)

	if len(job.TaskGroups) > 0 {
		desiredInstances = int32(*job.TaskGroups[0].Count)
	}

	return &pb.StatusResponse{
		DeploymentId:     req.DeploymentId,
		JobStatus:        *job.Status,
		JobType:          *job.Type,
		DesiredInstances: desiredInstances,
		RunningInstances: runningInstances,
		Allocations:      allocationStatuses,
		Message:          "Application status retrieved successfully",
	}, nil
}

// HealthCheck performs a health check on the service
func (s *ApplicationService) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	status := pb.HealthStatus_SERVING
	message := "Service is healthy"

	if s.orhClient != nil {
		err := s.orhClient.HealthCheck()
		if err != nil {
			status = pb.HealthStatus_NOT_SERVING
			message = fmt.Sprintf("Nomad client unhealthy: %v", err)
		}
	} else {
		status = pb.HealthStatus_NOT_SERVING
		message = "Nomad client not initialized"
	}

	return &pb.HealthCheckResponse{
		Status:    status,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}, nil
}
