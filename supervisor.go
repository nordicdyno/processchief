package supervisor

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/nordicdyno/simple-hypervisor/pb"
)

type Supervisor struct {
	defaultPidFileDir string

	servicesMu sync.RWMutex
	services   map[string]*Service
}

type Service struct {
	name    string
	cmdline string
	// svc     pb.Service

	cmd *exec.Cmd
	fin chan struct{}

	sync.RWMutex
	status string
}

func (s *Service) setStatus(status string) {
	s.Lock()
	s.status = status
	s.Unlock()
}

func (s *Service) getStatus() string {
	s.RLock()
	defer s.RUnlock()
	return s.status
}

func (s *Service) wait() error {
	return s.cmd.Wait()
}

// type ServiceStatus struct {
// }

func NewSupervisor() *Supervisor {
	return &Supervisor{
		// defaultPidFileDir: "./.pids",
		services: map[string]*Service{},
	}
}

func (s *Supervisor) StopAll() {
	fmt.Println("StopAll() start")
	names := s.AllServiceNames()
	fmt.Println(names)
	for _, name := range names {
		fmt.Printf("wait %v...\n", name)
		err := s.StopService(name)
		if err != nil {
			fmt.Printf("%v service not found while stopped\n", name)
		} else {
			fmt.Printf("%v service stopped\n", name)
		}
	}
	fmt.Println("StopAll() end")
}

func (s *Supervisor) StopService(name string) error {
	s.servicesMu.Lock()
	defer s.servicesMu.Unlock()
	svc, ok := s.services[name]
	if !ok {
		return fmt.Errorf("service with name '%v' not found", name)
	}
	err := svc.cmd.Process.Signal(os.Kill)
	if err != nil {
		if err.Error() == "os: process already finished" {
			return nil
		}
		log.Printf("sending signal error: %T (%v)", err, err)
		return err
	}
	delete(s.services, name)
	<-svc.fin
	return nil
}

func (s *Supervisor) UpdateService(name string, cmdline string) error {
	err := s.StopService(name)
	if err != nil {
		return err
	}
	s.servicesMu.Lock()
	err = s.setService(name, cmdline)
	s.servicesMu.Unlock()
	return err
}

func (s *Supervisor) setService(name string, cmdline string) error {
	// XXX: it's oversimplification, because args could be with spaces like 'a b c'
	args := strings.Split(cmdline, " ")

	cmd := exec.Command(args[0], args[1:]...)
	log.Printf("Exec command %v with args %v", args[0], args[1:])
	var (
		stdout io.Writer = os.Stdout
		stderr io.Writer = os.Stderr
	)
	stdout = prefixer("<STDOUT> ["+name+"]: ", stdout)
	stderr = prefixer("<STDERR> ["+name+"]: ", stderr)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Start()
	if err != nil {
		return err
	}

	svc := Service{
		name:    name,
		cmdline: cmdline,
		cmd:     cmd,
		status:  "started",
		fin:     make(chan struct{}),
	}
	s.services[name] = &svc

	go func() {
		err := cmd.Wait()
		status := "finished"
		if err != nil && err.Error() != "exec: Wait was already called" {
			status += fmt.Sprintf(" ERROR: %v", err.Error())
		}
		svc.setStatus(status)
		close(svc.fin)
	}()
	return nil
}

func (s *Supervisor) AddService(name string, cmdline string) error {
	s.servicesMu.Lock()
	defer s.servicesMu.Unlock()
	if _, ok := s.services[name]; ok {
		return fmt.Errorf("service with name '%v' already registered", name)
	}

	return s.setService(name, cmdline)
}

func (s *Supervisor) AllServiceNames() []string {
	s.servicesMu.RLock()
	defer s.servicesMu.RUnlock()
	names := make([]string, 0, len(s.services))
	for name := range s.services {
		names = append(names, name)
	}
	return names
}

func (s *Supervisor) Get(name string) (*Service, error) {
	s.servicesMu.RLock()
	defer s.servicesMu.RUnlock()

	svc, ok := s.services[name]
	if !ok {
		return nil, fmt.Errorf("service with name '%v' not found", name)
	}
	return svc, nil
}

func (s *Supervisor) GetService(name string) (*pb.Service, error) {
	svc, err := s.Get(name)
	if err != nil {
		return nil, err
	}

	pbSVC := &pb.Service{
		Name:        name,
		Commandline: svc.cmdline,
		Pid:         int32(svc.cmd.Process.Pid),
		Status:      svc.getStatus(),
	}
	return pbSVC, nil
}
