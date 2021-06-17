package implementation

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strconv"
)

func executeCoreJQAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	providedJson, ok := request.Parameters[userProviderJSONKey]
	if !ok {
		return nil, errors.New("no json provider for execution")
	}

	query, ok := request.Parameters[userProviderQueryKey]
	if !ok {
		return nil, errors.New("no query provider for execution")
	}

	cmd := fmt.Sprintf("/bin/echo '%s' | /bin/jq %s", providedJson, query)
	command := exec.Command("/bin/bash", "-c", cmd)

	outputBytes, execErr := command.CombinedOutput()
	if execErr != nil {
		log.Error("Detected failure, building result! Error: ", execErr)

		failureResult := FinalOutput{Output: string(outputBytes), Error: execErr.Error()}

		resultBytes, err := json.Marshal(failureResult)
		if err != nil {
			log.Error("Failed to properly marshal result, err: ", err)
			return nil, err
		}

		return resultBytes, nil
	}

	return outputBytes, nil

}

func executeCoreJPAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	providedJson, ok := request.Parameters[userProviderJSONKey]
	if !ok {
		return nil, errors.New("no json provider for execution")
	}

	query, ok := request.Parameters[userProviderQueryKey]
	if !ok {
		return nil, errors.New("no query provider for execution")
	}

	unquoted := ""
	unquotedKey, ok := request.Parameters[userProviderUnquotedKey]
	if ok {
		unquotedBool, err := strconv.ParseBool(unquotedKey)
		if err == nil && unquotedBool {
			unquoted = "--unquoted "
		}
	}
	cmd := fmt.Sprintf("/bin/echo '%s' | /bin/jp %s%s", providedJson, unquoted, query)
	command := exec.Command("/bin/bash", "-c", cmd)

	outputBytes, execErr := command.CombinedOutput()
	if execErr != nil {
		log.Error("Detected failure, building result! Error: ", execErr)

		failureResult := FinalOutput{Output: string(outputBytes), Error: execErr.Error()}

		resultBytes, err := json.Marshal(failureResult)
		if err != nil {
			log.Error("Failed to properly marshal result, err: ", err)
			return nil, err
		}

		return resultBytes, nil
	}

	return outputBytes, nil

}
