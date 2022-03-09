package common

import (
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"syscall"
	"time"
)

type CommandOutput struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

type Environment interface {
	GetHomeDirectory() string
	GetExecutorUid() uint32
	GetExecutorGid() uint32
}

func ExecuteBash(execution Environment, request *plugin.ExecuteActionRequest, environment []string, cmd string) ([]byte, error) {
	return ExecuteCommand(execution, request, environment, "/bin/bash", "-c", cmd)
}

func ExecuteCommand(execution Environment, request *plugin.ExecuteActionRequest, environment []string, name string, args ...string) ([]byte, error) {

	commandFinished := make(chan struct{})
	command := exec.Command(
		name,
		args...)

	command.Dir = execution.GetHomeDirectory()
	environment = append(environment, fmt.Sprintf("HOME=%s", execution.GetHomeDirectory()))
	environment = append(environment, fmt.Sprintf("PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:%[1]s/.local/bin:%[1]s/bin", execution.GetHomeDirectory()))
	command.Env = environment

	currentUser, err := user.Current()
	if err != nil {
		return nil, errors.Wrap(err, "failed getting current user: ")
	}

	if currentUser.Uid != fmt.Sprintf("%d", execution.GetExecutorUid()) {
		command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		command.SysProcAttr.Credential = &syscall.Credential{
			Uid: execution.GetExecutorUid(),
			Gid: execution.GetExecutorGid(),
		}
	}

	log.Infof("Executing command %s as user (%d, %d, %s)", name, execution.GetExecutorUid(), execution.GetExecutorGid(), execution.GetHomeDirectory())

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

func GetCommandFailureResponse(output []byte, err error, cli bool) ([]byte, error) {
	if cli {
		return []byte(fmt.Sprintf("%s; error: %s", string(output), err.Error())), CLIError
	}

	strOut := ""
	outLength := len(output)
	if outLength > 0 {
		strOut = string(output)
		if outLength > 1000 {
			strOut = strOut[:1000] + "..."
		}
	}

	return nil, errors.New(fmt.Sprintf("output (%d bytes): %s; error: %s", outLength, strOut, err))
}

func WriteToTempFile(execution Environment, bytes []byte, prefix string) (string, error) {
	file, err := ioutil.TempFile(execution.GetHomeDirectory(), prefix)
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
