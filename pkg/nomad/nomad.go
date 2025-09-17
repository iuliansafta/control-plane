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
