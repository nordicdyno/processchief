package supervisor

import (
	"context"
	"net"
	"net/http"

	"github.com/nordicdyno/simple-hypervisor/pb"
)

type serverWithContext struct {
	ctx      context.Context
	listener net.Listener
	http     *http.Server
}

type ControlServer struct {
	super *Supervisor

	server *serverWithContext
	fin    chan struct{}
}

var _ pb.ServicesAPIServer = &ControlServer{}

var nope = &pb.Nope{}

func NewControlServer(super *Supervisor) *ControlServer {
	return &ControlServer{
		super: super,
	}
}

func (s *ControlServer) AddService(ctx context.Context, svc *pb.NewService) (*pb.Nope, error) {
	return nope, s.super.AddService(svc.Name, svc.Commandline)
}

func (s *ControlServer) UpdateService(ctx context.Context, svc *pb.NewService) (*pb.Nope, error) {
	return nope, s.super.UpdateService(svc.Name, svc.Commandline)
}

// AllServices returns all registered services.
func (s *ControlServer) AllServices(context.Context, *pb.Nope) (*pb.Services, error) {
	result := &pb.Services{}
	for _, name := range s.super.AllServiceNames() {
		svc, err := s.super.GetService(name)
		if err != nil {
			continue
		}
		result.Service = append(result.Service, svc)
	}
	return result, nil
}

// GetService returns service description for provided topic name.
func (s *ControlServer) GetService(ctx context.Context, name *pb.ServiceName) (*pb.Service, error) {
	return s.super.GetService(name.Name)
}
