package common

import (
	"context"
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os/exec"
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
	ctx := context.Background()

	if request != nil && request.Timeout != 0 {
		// Create a new context and add a timeout to it
		tctx, cancel := context.WithTimeout(ctx, time.Duration(request.Timeout)*time.Second)
		ctx = tctx
		defer cancel()
	}

	// Create the command with our context
	command := exec.CommandContext(
		ctx,
		name,
		args...)

	if environment != nil {
		command.Env = environment
	}

	log.Infof("Executing %s", command.String())
	outputBytes, execErr := command.CombinedOutput()

	// We want to check the context error to see if the timeout was executed.
	// The error returned by cmd.Output() will be OS specific based on what
	// happens when a process is killed.
	if ctx.Err() == context.DeadlineExceeded {
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
