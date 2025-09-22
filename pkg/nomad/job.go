package nomad

import (
	"fmt"
	"maps"
	"time"

	nmd "github.com/hashicorp/nomad/api"
	"github.com/iuliansafta/control-plane/pkg/utils"
)

type Resources struct {
	CPU         *int
	Cores       *int
	MemoryMB    *int
	MemoryMaxMB *int
}

type ServiceCheck struct {
	Type     string
	Path     string
	Interval time.Duration
	Duration time.Duration
	Timeout  string
	Port     string
}

type TraefikSpec struct {
	Enable              bool
	Host                string
	Entrypoint          string
	EnableSSL           bool
	SSLHost             string
	CertResolver        string
	HealthCheckPath     string
	HealthCheckInterval string
	PathPrefix          string
	Middlewares         []string
	CustomLabels        map[string]string
}

type Ports struct {
	Label string
	Value int
	To    int
}

type JobTemplate struct {
	Name          string
	Image         string
	Instances     int
	Region        string
	Ports         Ports
	Environment   map[string]string
	ResourcesSpec Resources
	HealthCheck   ServiceCheck
	Traefik       TraefikSpec
	DisableConsul bool
	NetworkMode   string // "bridge" or "host", defaults to "host" if empty
}

func BuildJobTemplate(req *JobTemplate) *JobTemplate {
	return req
}

func (jt *JobTemplate) ToNomadJob() *nmd.Job {
	job := &nmd.Job{
		ID:          &jt.Name,
		Name:        &jt.Name,
		Type:        utils.StringPtr("service"),
		Datacenters: []string{"dc1"},
		TaskGroups:  jt.buildTaskGroup(),
	}

	if jt.Region != "" {
		job.Region = &jt.Region
	}

	return job
}

func (jt *JobTemplate) buildTaskGroup() []*nmd.TaskGroup {
	resources := &nmd.Resources{}
	if jt.ResourcesSpec.CPU != nil {
		resources.CPU = jt.ResourcesSpec.CPU
	}
	if jt.ResourcesSpec.Cores != nil {
		resources.Cores = jt.ResourcesSpec.Cores
	}
	if jt.ResourcesSpec.MemoryMB != nil {
		resources.MemoryMB = jt.ResourcesSpec.MemoryMB
	}
	if jt.ResourcesSpec.MemoryMaxMB != nil {
		resources.MemoryMaxMB = jt.ResourcesSpec.MemoryMaxMB
	}

	networks := []*nmd.NetworkResource{}
	if jt.Ports.Label != "" {
		networkMode := jt.NetworkMode
		if networkMode == "" {
			networkMode = "host"
		}

		network := &nmd.NetworkResource{
			Mode: networkMode,
		}

		var dynamicPorts []nmd.Port
		if networkMode == "bridge" {
			dynamicPorts = append(dynamicPorts, nmd.Port{
				Label: jt.Ports.Label,
				To:    jt.Ports.To, // Container port
			})
		} else {
			dynamicPorts = append(dynamicPorts, nmd.Port{
				Label: jt.Ports.Label,
				Value: jt.Ports.Value, // Host port (0 for dynamic allocation)
			})
		}

		network.DynamicPorts = dynamicPorts
		networks = append(networks, network)
	}

	driverConfig := map[string]any{
		"image": jt.Image,
	}

	task := &nmd.Task{
		Name:      jt.Name,
		Driver:    "containerd-driver", //TODO: I need to do this dynamically
		Config:    driverConfig,
		Resources: resources,
		Env:       jt.Environment,
	}

	var services []*nmd.Service
	if jt.Ports.Label != "" && !jt.DisableConsul {
		traefikTags := jt.Traefik.GenerateTraefikTags(jt.Name, jt.Ports.Label)

		service := &nmd.Service{
			Name:      jt.Name + "-" + jt.Ports.Label,
			PortLabel: jt.Ports.Label,
			Tags:      traefikTags,
		}

		if jt.HealthCheck.Type != "" {
			timeout, err := time.ParseDuration(jt.HealthCheck.Timeout)
			if err != nil {
				timeout = 10 * time.Second
			}

			check := &nmd.ServiceCheck{
				Type:     jt.HealthCheck.Type,
				Path:     jt.HealthCheck.Path,
				Interval: jt.HealthCheck.Interval,
				Timeout:  timeout,
			}
			if jt.HealthCheck.Port != "" {
				check.PortLabel = jt.HealthCheck.Port
			} else {
				check.PortLabel = jt.Ports.Label
			}
			service.Checks = []nmd.ServiceCheck{*check}
		}

		services = append(services, service)
	}

	taskGroup := &nmd.TaskGroup{
		Name:     utils.StringPtr(jt.Name + "-group"),
		Count:    &jt.Instances,
		Tasks:    []*nmd.Task{task},
		Networks: networks,
		Services: services,
	}

	return []*nmd.TaskGroup{taskGroup}
}

func (ts *TraefikSpec) GenerateTraefikTags(serviceName, portLabel string) []string {
	if !ts.Enable {
		return []string{"deployment"}
	}

	tags := []string{
		"deployment",
		"traefik.enable=true",
	}

	if ts.Host != "" {
		routerName := serviceName
		tags = append(tags, fmt.Sprintf("traefik.http.routers.%s.rule=Host(`%s`)", routerName, ts.Host))

		entrypoint := ts.Entrypoint
		if entrypoint == "" {
			entrypoint = "web"
		}
		tags = append(tags, fmt.Sprintf("traefik.http.routers.%s.entrypoints=%s", routerName, entrypoint))

		if ts.PathPrefix != "" {
			rule := fmt.Sprintf("Host(`%s`) && PathPrefix(`%s`)", ts.Host, ts.PathPrefix)
			tags = append(tags, fmt.Sprintf("traefik.http.routers.%s.rule=%s", routerName, rule))
		}

		if len(ts.Middlewares) > 0 {
			middlewares := ts.Middlewares[0]
			for i := 1; i < len(ts.Middlewares); i++ {
				middlewares += "," + ts.Middlewares[i]
			}
			tags = append(tags, fmt.Sprintf("traefik.http.routers.%s.middlewares=%s", routerName, middlewares))
		}
	}

	if ts.EnableSSL && ts.Host != "" {
		sslRouterName := serviceName + "-secure"
		sslHost := ts.SSLHost
		if sslHost == "" {
			sslHost = ts.Host
		}

		tags = append(tags,
			fmt.Sprintf("traefik.http.routers.%s.rule=Host(`%s`)", sslRouterName, sslHost),
			fmt.Sprintf("traefik.http.routers.%s.entrypoints=websecure", sslRouterName),
		)

		if ts.PathPrefix != "" {
			rule := fmt.Sprintf("Host(`%s`) && PathPrefix(`%s`)", sslHost, ts.PathPrefix)
			tags = append(tags, fmt.Sprintf("traefik.http.routers.%s.rule=%s", sslRouterName, rule))
		}

		if ts.CertResolver != "" {
			tags = append(tags, fmt.Sprintf("traefik.http.routers.%s.tls.certresolver=%s", sslRouterName, ts.CertResolver))
		} else {
			tags = append(tags, fmt.Sprintf("traefik.http.routers.%s.tls=true", sslRouterName))
		}

		if len(ts.Middlewares) > 0 {
			middlewares := ts.Middlewares[0]
			for i := 1; i < len(ts.Middlewares); i++ {
				middlewares += "," + ts.Middlewares[i]
			}
			tags = append(tags, fmt.Sprintf("traefik.http.routers.%s.middlewares=%s", sslRouterName, middlewares))
		}
	}

	tags = append(tags, fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=${NOMAD_PORT_%s}", serviceName, portLabel))

	if ts.HealthCheckPath != "" {
		tags = append(tags, fmt.Sprintf("traefik.http.services.%s.loadbalancer.healthcheck.path=%s", serviceName, ts.HealthCheckPath))

		interval := ts.HealthCheckInterval
		if interval == "" {
			interval = "30s"
		}
		tags = append(tags, fmt.Sprintf("traefik.http.services.%s.loadbalancer.healthcheck.interval=%s", serviceName, interval))
	}

	for key, value := range ts.CustomLabels {
		tags = append(tags, fmt.Sprintf("%s=%s", key, value))
	}

	return tags
}

func NewTraefikSpec(host string, options ...TraefikOption) TraefikSpec {
	spec := TraefikSpec{
		Enable:              true,
		Host:                host,
		Entrypoint:          "web",
		HealthCheckPath:     "/",
		HealthCheckInterval: "30s",
		CustomLabels:        make(map[string]string),
	}

	for _, opt := range options {
		opt(&spec)
	}

	return spec
}

type TraefikOption func(*TraefikSpec)

func WithSSL(certResolver string) TraefikOption {
	return func(spec *TraefikSpec) {
		spec.EnableSSL = true
		spec.CertResolver = certResolver
	}
}

func WithPathPrefix(prefix string) TraefikOption {
	return func(spec *TraefikSpec) {
		spec.PathPrefix = prefix
	}
}

func WithMiddlewares(middlewares ...string) TraefikOption {
	return func(spec *TraefikSpec) {
		spec.Middlewares = middlewares
	}
}

func WithHealthCheck(path, interval string) TraefikOption {
	return func(spec *TraefikSpec) {
		spec.HealthCheckPath = path
		spec.HealthCheckInterval = interval
	}
}

func WithCustomLabels(labels map[string]string) TraefikOption {
	return func(spec *TraefikSpec) {
		maps.Copy(spec.CustomLabels, labels)
	}
}
