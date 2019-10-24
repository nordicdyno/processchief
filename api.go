package processchief

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/nordicdyno/processchief/pb"
)

type serverWithContext struct {
	ctx      context.Context
	listener net.Listener
	http     *http.Server
}

type ControlServer struct {
	chief *Chief

	server *serverWithContext
	fin    chan struct{}
}

var _ pb.ControlAPIServer = &ControlServer{}

var nope = &pb.Nope{}

func NewControlServer(chief *Chief) *ControlServer {
	return &ControlServer{
		chief: chief,
	}
}

func (cs *ControlServer) ProcessSignal(ctx context.Context, svcSig *pb.Signal) (*pb.Result, error) {
	err := cs.chief.ProcessSignal(svcSig.Name, svcSig.Signal)
	if err != nil {
		return nil, err
	}
	return &pb.Result{Description: "OK"}, nil
}

func (cs *ControlServer) LoggerSignal(ctx context.Context, svcSig *pb.Signal) (*pb.Result, error) {
	err := cs.chief.LoggerSignal(svcSig.Name, svcSig.Signal)
	if err != nil {
		return nil, err
	}
	return &pb.Result{Description: "OK"}, nil
}

func (cs *ControlServer) AddProcess(ctx context.Context, pSet *pb.SetProc) (*pb.ProcStatus, error) {
	p := pSet.Process
	return cs.chief.AddProcess(p.Name, *pSet)
}

func (cs *ControlServer) UpdateProcess(ctx context.Context, pSet *pb.SetProc) (*pb.ProcStatus, error) {
	p := pSet.Process
	return cs.chief.UpdateProcess(p.Name, *pSet)
}

// AllProcesses returns all registered processes.
func (cs *ControlServer) AllProcesses(context.Context, *pb.Nope) (*pb.ProcessesStatus, error) {
	statuses := &pb.ProcessesStatus{}
	all := cs.chief.AllProcesses()
	for _, p := range all {
		statuses.Statuses = append(statuses.Statuses, &p)
	}
	return statuses, nil
}

// Halt stops chief and stop process.
func (cs *ControlServer) Halt(context.Context, *pb.Nope) (*pb.Result, error) {
	cs.chief.StopAll()
	go func() {
		time.Sleep(time.Second * 2)
		os.Exit(0)
	}()
	return &pb.Result{Description: "OK."}, nil
}

// GetProcess returns process status description by name.
func (cs *ControlServer) GetProcess(ctx context.Context, pn *pb.ProcName) (*pb.ProcStatus, error) {
	return cs.chief.GetProcess(pn.Name)
}

// DeleteService stops and removes process by name.
func (cs *ControlServer) DeleteProcess(ctx context.Context, name *pb.ProcName) (*pb.Result, error) {
	err := cs.chief.DeleteProcess(name.Name)
	if err != nil {
		return nil, err
	}
	return &pb.Result{Description: "OK"}, nil
}
