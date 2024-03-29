package processchief

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Chief is the chief of all managed processes.
type Chief struct {
	mu    sync.RWMutex
	procs map[string]*procHolder
}

func NewChief() *Chief {
	return &Chief{
		procs: map[string]*procHolder{},
	}
}

func (c *Chief) StopAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Println("Stopping all processess")
	for name := range c.procs {
		fmt.Printf("wait %v...\n", name)
		err := c.stopProcess(name)
		if err != nil {
			fmt.Printf("%v stop error: %v\n", name, err)
		} else {
			fmt.Printf("%v stopped\n", name)
		}
	}
}

func (c *Chief) StopProcess(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopProcess(name)
}

func (c *Chief) stopProcess(name string) error {
	p, ok := c.procs[name]
	if !ok {
		return fmt.Errorf("'%v' not found", name)
	}
	err := p.cmd.Process.Signal(os.Kill)
	if err != nil {
		if err.Error() == "os: process already finished" {
			return nil
		}
		log.Printf("sending signal error: %T (%v)", err, err)
		return err
	}
	delete(c.procs, name)
	<-p.fin
	return nil
}

func (c *Chief) LoggerSignal(name string, signal int32) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.procs[name]
	if !ok {
		return fmt.Errorf("'%v' not found", name)
	}
	if p.logCmd == nil {
		return fmt.Errorf("logger for '%v' not found", name)
	}
	fmt.Printf(">>> send signal %v to logger of '%v' (pid=%v)\n", signal, name, p.logCmd.Process.Pid)
	return p.logCmd.Process.Signal(syscall.Signal(signal))
}

func (c *Chief) ProcessSignal(name string, signal int32) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	svc, ok := c.procs[name]
	if !ok {
		return fmt.Errorf("'%v' not found", name)
	}
	if svc.logCmd == nil {
		return fmt.Errorf("logger for '%v' not found", name)
	}
	fmt.Printf(">>> send signal %v to '%v' (pid=%v)\n", signal, name, svc.cmd.Process.Pid)
	return svc.cmd.Process.Signal(syscall.Signal(signal))
}

func (c *Chief) UpdateProcess(name string, sp SetProc) (*ProcStatus, error) {
	err := c.StopProcess(name)
	if err != nil {
		return nil, err
	}
	// possible logical race, but it'c not a problem, it just returns error
	c.mu.Lock()
	status, err := c.setProcess(name, sp)
	c.mu.Unlock()
	return status, err
}

func (c *Chief) setProcess(name string, sp SetProc) (*ProcStatus, error) {
	var err error
	p := sp.Process
	pEnv := sp.Env
	// XXX: it'c oversimplification, because args could be with spaces like 'a b c'
	cmdArgs := strings.Split(p.CommandLine, " ")
	cmdArg0 := cmdArgs[0]
	var (
		logArg0 string
		logArgs []string
	)
	if strings.IndexByte(cmdArg0, filepath.Separator) != -1 {
		cmdArg0, err = exec.LookPath(cmdArg0)
		if err != nil {
			return nil, err
		}
	}

	var logCmd *exec.Cmd
	cmd := exec.Command(cmdArg0, cmdArgs[1:]...)
	// var (
	// 	stdout io.Writer = os.Stdout
	// 	stderr io.Writer = os.Stderr
	// )
	if p.LoggerCommandLine == "" {
		stdout := prefixer("<STDOUT> ["+name+"]: ", os.Stdout)
		stderr := prefixer("<STDERR> ["+name+"]: ", os.Stderr)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	} else {
		logArgs = strings.Split(p.LoggerCommandLine, " ")
		logArg0 = logArgs[0]
		if strings.IndexByte(logArg0, filepath.Separator) != -1 {
			logArg0, err = exec.LookPath(logArg0)
			if err != nil {
				return nil, err
			}
		}
		logCmd = exec.Command(logArg0, logArgs[1:]...)
		outPipe, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		errPipe, err := cmd.StderrPipe()
		if err != nil {
			panic(err)
		}
		logCmd.Stdin = io.MultiReader(outPipe, errPipe)
		// logCmd.Stderr, _ = logCmd.StdoutPipe()
		stdout := prefixer("<LOGGER> ["+name+"]: ", os.Stdout)
		stderr := prefixer("<LOGGER> ["+name+"]: ", os.Stderr)
		logCmd.Stdout = stdout
		logCmd.Stderr = stderr
	}
	cmd.Dir = pEnv.WorkingDir
	cmd.Env = pEnv.EnvVars

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	log.Printf(">>> Start service %v: %v args %v", name, cmdArg0, cmdArgs[1:])

	if logCmd != nil {
		err = logCmd.Start()
		if err != nil {
			return nil, err
		}
		log.Printf(">>> Start logger for '%v': %v args: %v", name, logArg0, logArgs[1:])
	}

	procHolder := &procHolder{
		status: ProcStatus{Process: p},

		cmd:    cmd,
		logCmd: logCmd,
		fin:    make(chan struct{}),
	}
	c.procs[name] = procHolder

	go func() {
		err := cmd.Wait()
		status := "finished"
		if err != nil && err.Error() != "exec: Wait was already called" {
			status += fmt.Sprintf(" ERROR: %v", err.Error())
		}
		procHolder.setState(status)
		close(procHolder.fin)
	}()

	startWaitDuration := time.Second * 5
	startT := time.Now()
	endWaitT := startT.Add(startWaitDuration)

	var status *ProcStatus
	startErr := fmt.Errorf("failed to wait process start during %v", startWaitDuration)
	for time.Now().Before(endWaitT) {
		time.Sleep(time.Millisecond * 100)
		proc := cmd.Process
		if proc == nil {
			// fmt.Println("process is nil")
			continue
		}

		startErr = nil
		status = &ProcStatus{
			Pid:     int32(proc.Pid),
			Process: p,
			State:   "started",
		}

		pc := cmd.ProcessState
		if pc != nil {
			// fmt.Println("Pid:", pc.Pid())
			// fmt.Println("Success:", pc.Success())
			// fmt.Println("ExitCode:", pc.ExitCode())
			status.State = "exited"
			if pc.ExitCode() == 0 {
				startErr = nil
			} else {
				startErr = fmt.Errorf("process exited with code %v", pc.ExitCode())
			}
			status.Exited = true
		}
		break
	}
	fmt.Println("start loop ended")

	return status, startErr
}

func (c *Chief) AddProcess(name string, sp SetProc) (*ProcStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.procs[name]; ok {
		return nil, fmt.Errorf("'%v' already registered", name)
	}

	return c.setProcess(name, sp)
}

func (c *Chief) AllProcesses() []ProcStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statuses := make([]ProcStatus, 0, len(c.procs))
	for _, p := range c.procs {
		statuses = append(statuses, p.Status())
	}
	return statuses
}

func (c *Chief) Get(name string) (*ProcStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	p, ok := c.procs[name]
	if !ok {
		return nil, fmt.Errorf("process with name '%v' not found", name)
	}
	status := p.Status()
	return &status, nil
}

func (c *Chief) GetProcess(name string) (*ProcStatus, error) {
	p, err := c.Get(name)
	if err != nil {
		return nil, err
	}

	proc := *p
	return &proc, nil
}

// TODO: refactor Stop and Delete commands
func (c *Chief) DeleteProcess(name string) error {
	return c.StopProcess(name)
}
