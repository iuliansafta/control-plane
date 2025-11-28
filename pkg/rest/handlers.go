package rest

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	pb "github.com/iuliansafta/control-plane/api/proto"
)

// DeployRequest JSON request for deployment
type DeployRequest struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Replicas    int32             `json:"replicas"`
	CPU         float64           `json:"cpu"`
	Memory      int64             `json:"memory"`
	Region      string            `json:"region,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	NetworkMode string            `json:"network_mode,omitempty"`
	Host        string            `json:"host,omitempty"`
	SSL         bool              `json:"ssl,omitempty"`
}

// TraefikConfig Traefik configuration
type TraefikConfig struct {
	Enable              bool              `json:"enable"`
	Host                string            `json:"host"`
	Entrypoint          string            `json:"entrypoint,omitempty"`
	EnableSSL           bool              `json:"enable_ssl,omitempty"`
	SSLHost             string            `json:"ssl_host,omitempty"`
	CertResolver        string            `json:"cert_resolver,omitempty"`
	HealthCheckPath     string            `json:"health_check_path,omitempty"`
	HealthCheckInterval string            `json:"health_check_interval,omitempty"`
	PathPrefix          string            `json:"path_prefix,omitempty"`
	Middlewares         []string          `json:"middlewares,omitempty"`
	CustomLabels        map[string]string `json:"custom_labels,omitempty"`
}

// DeployResponse response for deployment
type DeployResponse struct {
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// DeleteResponse response for deletion
type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// StatusResponse response for status
type StatusResponse struct {
	DeploymentID     string             `json:"deployment_id"`
	JobStatus        string             `json:"job_status"`
	JobType          string             `json:"job_type"`
	DesiredInstances int32              `json:"desired_instances"`
	RunningInstances int32              `json:"running_instances"`
	Allocations      []AllocationStatus `json:"allocations"`
	Message          string             `json:"message"`
}

// AllocationStatus represents allocation status in JSON
type AllocationStatus struct {
	AllocationID  string            `json:"allocation_id"`
	NodeID        string            `json:"node_id"`
	NodeName      string            `json:"node_name"`
	Status        string            `json:"status"`
	DesiredStatus string            `json:"desired_status"`
	CreateTime    int64             `json:"create_time"`
	ModifyTime    int64             `json:"modify_time"`
	TaskStates    map[string]string `json:"task_states"`
}

// deployApplication handles POST /api/v1/applications
// @Summary Deploy an application
// @Description Deploy a new application
// @Tags applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security ApiKeyAuth
// @Param request body DeployRequest true "Deploy Request"
// @Success 201 {object} DeployResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /applications [post]
func (s *Server) deployApplication(c echo.Context) error {
	var req DeployRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" || req.Image == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "name and image are required",
		})
	}

	pbReq := &pb.DeployRequest{
		Name:     req.Name,
		Image:    req.Image,
		Replicas: req.Replicas,
		Cpu:      req.CPU,
		Memory:   req.Memory,
		Region:   req.Region,
		Labels:   req.Labels,
	}

	switch req.NetworkMode {
	case "bridge":
		pbReq.NetworkMode = pb.NetworkMode_NETWORK_MODE_BRIDGE
	case "host":
		pbReq.NetworkMode = pb.NetworkMode_NETWORK_MODE_HOST
	default:
		pbReq.NetworkMode = pb.NetworkMode_NETWORK_MODE_HOST
	}

	if req.Host != "" {
		pbReq.Traefik = &pb.TraefikConfig{
			Enable:              true,
			Host:                req.Host,
			EnableSsl:           req.SSL,
			Entrypoint:          "websecure",
			HealthCheckPath:     "/",
			HealthCheckInterval: "30s",
		}
	}

	resp, err := s.appService.DeployApplication(context.Background(), pbReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, DeployResponse{
		DeploymentID: resp.DeploymentId,
		Status:       resp.Status,
		Message:      resp.Message,
	})
}

// deleteApplication handles DELETE /api/v1/applications/:id
// @Summary Delete an application
// @Description Delete an application by ID
// @Tags applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security ApiKeyAuth
// @Param id path string true "Deployment ID"
// @Success 200 {object} DeleteResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /applications/{id} [delete]
func (s *Server) deleteApplication(c echo.Context) error {
	deploymentID := c.Param("id")
	if deploymentID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "deployment_id is required",
		})
	}

	pbReq := &pb.DeleteRequest{
		DeploymentId: deploymentID,
	}

	resp, err := s.appService.DeleteApplication(context.Background(), pbReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	if !resp.Success {
		return c.JSON(http.StatusBadRequest, DeleteResponse{
			Success: resp.Success,
			Message: resp.Message,
		})
	}

	return c.JSON(http.StatusOK, DeleteResponse{
		Success: resp.Success,
		Message: resp.Message,
	})
}

// getApplicationStatus handles GET /api/v1/applications/:id/status
// @Summary Get application status
// @Description Get status of an application by ID
// @Tags applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security ApiKeyAuth
// @Param id path string true "Deployment ID"
// @Success 200 {object} StatusResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /applications/{id}/status [get]
func (s *Server) getApplicationStatus(c echo.Context) error {
	deploymentID := c.Param("id")
	if deploymentID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "deployment_id is required",
		})
	}

	pbReq := &pb.StatusRequest{
		DeploymentId: deploymentID,
	}

	resp, err := s.appService.GetApplicationStatus(context.Background(), pbReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	allocations := make([]AllocationStatus, len(resp.Allocations))
	for i, alloc := range resp.Allocations {
		allocations[i] = AllocationStatus{
			AllocationID:  alloc.AllocationId,
			NodeID:        alloc.NodeId,
			NodeName:      alloc.NodeName,
			Status:        alloc.Status,
			DesiredStatus: alloc.DesiredStatus,
			CreateTime:    alloc.CreateTime,
			ModifyTime:    alloc.ModifyTime,
			TaskStates:    alloc.TaskStates,
		}
	}

	return c.JSON(http.StatusOK, StatusResponse{
		DeploymentID:     resp.DeploymentId,
		JobStatus:        resp.JobStatus,
		JobType:          resp.JobType,
		DesiredInstances: resp.DesiredInstances,
		RunningInstances: resp.RunningInstances,
		Allocations:      allocations,
		Message:          resp.Message,
	})
}
