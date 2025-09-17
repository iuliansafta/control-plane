package nomad

import (
	"log"

	nmd "github.com/hashicorp/nomad/api"
)

type NomadClient struct {
	client *nmd.Client
}

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

func (nc *NomadClient) DeployJob(jobTemplate *JobTemplate) (*nmd.JobRegisterResponse, error) {
	job := jobTemplate.ToNomadJob()

	jobs := nc.client.Jobs()
	resp, _, err := jobs.Register(job, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (nc *NomadClient) DeleteJob(jobID string) error {
	jobs := nc.client.Jobs()
	_, _, err := jobs.Deregister(jobID, true, nil)
	return err
}
