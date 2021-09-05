package common

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"syscall"
	"time"
)

type CommandOutput struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

func ExecuteBash(request *plugin.ExecuteActionRequest, environment []string, cmd string) ([]byte, error) {
	return ExecuteCommand(request, environment, "/bin/bash", "-c", cmd)
}

func ExecuteCommand(request *plugin.ExecuteActionRequest, environment []string, name string, args ...string) ([]byte, error) {

	commandFinished := make (chan struct{})
	command := exec.Command(
		name,
		args...)

	if environment != nil {
		command.Env = environment
	}

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// golang context with deadline kills the process but not the children, hence our own impl to kill all after timeout

	// Internally this will cause Go to call setpgid(2) between fork(2) and execve(2),
	// to assign the child process a new PGID identical to its PID.
	// This allows us to kill all processes in the process group by sending a KILL to -PID of the process,
	// which is the same as -PGID. Assuming that the child process did not use setpgid(2) when spawning its own child,
	// this should kill the child along with all of its children on any *Nix systems.
	var timedOut bool
	if request != nil && request.Timeout != 0 {
		// timeout goroutine
		go func() {
			select {
			case <- time.After(time.Duration(request.Timeout) * time.Second):
				syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
				timedOut = true
			case <- commandFinished:
			}
		}()
	}

	log.Infof("Executing %s", command.String())
	outputBytes, execErr := command.CombinedOutput()
	// signal timeout goroutine to exit
	close(commandFinished)

	// Check for timeout
	if timedOut {
		timeoutError := errors.New(fmt.Sprintf("command timed out: %s", command))
		log.Error(timeoutError)
		return nil, timeoutError
	}

	if execErr != nil {
		log.Errorf("Detected failure, building result! Error: %v", execErr)
	}

	return outputBytes, execErr
}

func GetCommandFailureResponse(output []byte, err error) ([]byte, error) {
	return nil, errors.New(string(output) + " - error: " + err.Error())
}
