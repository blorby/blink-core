package implementation

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

func executeCoreBashAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	code, ok := request.Parameters[userProviderCodeKey]
	if !ok {
		return nil, errors.New("no code provider for execution")
	}
	command := exec.Command("/bin/bash",
		"-c",
		fmt.Sprintf("%s", code))

	outputBytes, execErr := command.CombinedOutput()
	if execErr != nil {
		log.Error("Detected failure, building result! Error: ", execErr)

		failureResult := CommandOutput{Output: string(outputBytes), Error: execErr.Error()}

		resultBytes, err := json.Marshal(failureResult)
		if err != nil {
			log.Error("Failed to properly marshal result, err: ", err)
			return nil, err
		}

		return resultBytes, nil
	}

	return outputBytes, nil
}
