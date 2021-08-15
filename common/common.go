package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
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

func ExecuteCommandPipeline(request *plugin.ExecuteActionRequest, environment []string, name string, cmd string) ([]byte, error) {
	if !strings.Contains(cmd, "|") {
		return ExecuteCommand(request, environment, name, strings.Split(cmd, " ")...)
	}

	var commands []*exec.Cmd

	pipes := strings.Split(cmd, " | ")
	for index, pipe := range pipes {
		pipe = strings.Trim(pipe, " ")

		ctx := context.Background()

		if request != nil && request.Timeout != 0 {
			// Create a new context and add a timeout to it
			timeoutContext, cancel := context.WithTimeout(ctx, time.Duration(request.Timeout)*time.Second)
			ctx = timeoutContext
			defer cancel()
		}

		// Create the command with our context
		args := strings.Split(pipe, " ")
		if index > 0 {
			name = args[0]
			if len(args) >= 2 {
				args = args[1:]
			} else {
				args = nil
			}
		}
		execCommand := exec.CommandContext(
			ctx,
			name,
			args...)

		if environment != nil {
			execCommand.Env = environment
		}

		commands = append(commands, execCommand)
	}

	log.Infof("Executing %v", commands)
	outputBytes, execErr, err := pipeline(commands...)
	if err != nil {
		// We want to check the context error to see if the timeout was executed.
		// The error returned by cmd.Output() will be OS specific based on what
		// happens when a process is killed.
		if err == context.DeadlineExceeded {
			timeoutError := errors.New(fmt.Sprintf("command timed out: %v", commands))
			log.Error(timeoutError)
			return nil, timeoutError
		}
	}

	if execErr != nil {
		log.Errorf("Detected failure, building result! Error: %v", execErr)
	}

	return outputBytes, err
}

// pipeline strings together the given exec.Cmd commands in a similar fashion
// to the Unix pipeline.  Each command's standard output is connected to the
// standard input of the next command, and the output of the final command in
// the pipeline is returned, along with the collected standard error of all
// commands and the first error found (if any).
//
// To provide input to the pipeline, assign an io.Reader to the first's Stdin.
func pipeline(commands ...*exec.Cmd) (pipeLineOutput, collectedStandardError []byte, pipeLineError error) {
	// Require at least one command
	if len(commands) < 1 {
		return nil, nil, nil
	}

	// Collect the output from the command(s)
	var output bytes.Buffer
	var stderr bytes.Buffer

	last := len(commands) - 1
	for i, cmd := range commands[:last] {
		var err error
		// Connect each command's stdin to the previous command's stdout
		if commands[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
			return nil, nil, err
		}
		// Connect each command's stderr to a buffer
		cmd.Stderr = &stderr
	}

	// Connect the output and error for the last command
	commands[last].Stdout, commands[last].Stderr = &output, &stderr

	// Start each command
	for _, cmd := range commands {
		if err := cmd.Start(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}
	}

	// Wait for each command to complete
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}
	}

	// Return the pipeline output and the collected standard error
	return output.Bytes(), stderr.Bytes(), nil
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
