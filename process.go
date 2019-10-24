package processchief

import (
	"os/exec"
	"sync"

	"github.com/nordicdyno/processchief/pb"
)

// TODO: add states enum

type ProcStatus = pb.ProcStatus

type Process = pb.Process

type SetProc = pb.SetProc

type procHolder struct {
	sync.RWMutex
	status ProcStatus

	cmd    *exec.Cmd
	logCmd *exec.Cmd
	fin    chan struct{}
}

func (ph *procHolder) Status() ProcStatus {
	ph.RLock()
	s := ph.status
	s.Pid = int32(ph.cmd.Process.Pid)
	if ph.cmd.ProcessState != nil {
		s.Exited = true
	}
	ph.RUnlock()
	return s
}

func (ph *procHolder) setState(state string) {
	ph.Lock()
	ph.status.State = state
	ph.Unlock()
}

func (ph *procHolder) getState() string {
	ph.RLock()
	defer ph.RUnlock()
	return ph.status.State
}

func (ph *procHolder) wait() error {
	return ph.cmd.Wait()
}

type processCfg struct {
	workDir string
	env []string
}

type ProcOption func(*processCfg)

func Env(s string) ProcOption {
	return func(cfg *processCfg) {
		cfg.env = append(cfg.env, s)
	}
}

func WorkDir(s string) ProcOption {
	return func(cfg *processCfg) {
		cfg.workDir = s
	}
}


func NewProcess(name string, commandLine string, opts ...ProcOption) SetProc {
	cfg := &processCfg{}
	for _, opt := range  opts {
		opt(cfg)
	}
	p := &Process{
		Name:        name,
		CommandLine: commandLine,
	}
	return SetProc{
		Process: p,
		Env: &pb.ProcEnv{
			EnvVars:              cfg.env,
			WorkingDir:           cfg.workDir,
		},
	}
}
