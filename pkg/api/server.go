package api

import (
	pb "github.com/iuliansafta/iulian-cloud-controller/api/proto"
)

type Server struct {
	pb.UnimplementedControlPlaneServer
}

func NewServer() *Server {
	return &Server{}
}
