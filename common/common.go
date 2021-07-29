package common

import (
	"context"
	"encoding/json"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"time"
)

type CommandOutput struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

func ExecuteCommand(request *plugin.ExecuteActionRequest, environment []string, name string, args ...string) ([]byte, error) {
	ctx := context.Background()

	if request != nil && request.Timeout != 0 {
		// Create a new context and add a timeout to it
		tctx, cancel := context.WithTimeout(ctx, time.Duration(request.Timeout) * time.Second)
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
		log.Error("Command timed out")
		return nil, ctx.Err()
	}

	if execErr != nil {
		log.Errorf("Detected failure, building result! Error: %v", execErr)
	}

	return outputBytes, execErr
}

func GetCommandFailureResponse(output []byte, err error) ([]byte, error) {
	failureResult := CommandOutput{Output: string(output), Error: err.Error()}

	resultBytes, err := json.Marshal(failureResult)
	if err != nil {
		log.Error("Failed to properly marshal result, err: ", err)
		return nil, err
	}

	return resultBytes, nil
}
