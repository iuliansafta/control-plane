package nomad

import (
	"log"

	nmd "github.com/hashicorp/nomad/api"
)

type NomadClient struct {
	client *nmd.Client
}

// NewNomadClient creates a new Nomad client
func NewNomadClient(address string) (*NomadClient, error) {
	config := nmd.DefaultConfig()
	config.Address = address

	client, err := nmd.NewClient(config)
	if err != nil {
		log.Fatal("connection with nomad failed")

		return nil, err
	}

	return &NomadClient{
		client: client,
	}, nil
}

// DeployJob deploys a job to the orchestrator
func (nc *NomadClient) DeployJob(jobTemplate *JobTemplate) (*nmd.JobRegisterResponse, error) {
	job := jobTemplate.ToNomadJob()

	jobs := nc.client.Jobs()
	resp, _, err := jobs.Register(job, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// DeleteJob deletes a job from the orchestrator
func (nc *NomadClient) DeleteJob(jobID string) error {
	jobs := nc.client.Jobs()
	_, _, err := jobs.Deregister(jobID, true, nil)
	return err
}

// GetJobStatus retrieves the status of a job and its allocations
func (nc *NomadClient) GetJobStatus(jobID string) (*nmd.Job, []*nmd.AllocationListStub, error) {
	jobs := nc.client.Jobs()

	job, _, err := jobs.Info(jobID, nil)
	if err != nil {
		return nil, nil, err
	}

	allocations, _, err := jobs.Allocations(jobID, false, nil)
	if err != nil {
		return job, nil, err
	}

	return job, allocations, nil
}

// HealthCheck checks the health of the Nomad connection
func (nc *NomadClient) HealthCheck() error {
	agent := nc.client.Agent()
	_, err := agent.Self()
	return err
}
