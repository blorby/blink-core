package common

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type CommandOutput struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

func ExecuteBash(execution *execution.PrivateExecutionEnvironment, request *plugin.ExecuteActionRequest, environment []string, cmd string) ([]byte, error) {
	return ExecuteCommand(execution, request, environment, "/bin/bash", "-c", cmd)
}

func ExecuteCommand(execution *execution.PrivateExecutionEnvironment, request *plugin.ExecuteActionRequest, environment []string, name string, args ...string) ([]byte, error) {

	commandFinished := make(chan struct{})
	command := exec.Command(
		name,
		args...)

	command.Dir = execution.GetTempDirectory()

	if environment != nil {
		command.Env = environment
	}

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.SysProcAttr.Credential = &syscall.Credential{
		Uid:         execution.GetExecutorUid(),
		Gid:         execution.GetExecutorGid(),
	}

	log.Infof("Executing command %s as user (%d, %d, %s)", name, execution.GetExecutorUid(), execution.GetExecutorGid(), execution.GetTempDirectory())

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
			case <-time.After(time.Duration(request.Timeout) * time.Second):
				syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
				timedOut = true
			case <-commandFinished:
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



func WriteToTempFile(execution *execution.PrivateExecutionEnvironment, bytes []byte, prefix string) (string, error) {
	file, err := ioutil.TempFile(execution.GetTempDirectory(), prefix)
	if err != nil {
		return "", err
	}

	defer func() {
		// Close the file
		if err := file.Close(); err != nil {
			log.Error("failed to close file", err)
		}
	}()

	_, err = file.Write(bytes)
	if err != nil {
		return "", err
	}

	err = os.Chown(file.Name(), int(execution.GetExecutorUid()), int(execution.GetExecutorGid()))
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}
