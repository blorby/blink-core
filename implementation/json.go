package implementation

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strconv"
)

func executeCoreJQAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	providedJson, ok := request.Parameters[jsonKey]
	if !ok {
		return nil, errors.New("no json provided for execution")
	}

	query, ok := request.Parameters[queryKey]
	if !ok {
		return nil, errors.New("no query provided for execution")
	}

	cmd := fmt.Sprintf("/bin/echo '%s' | /bin/jq %s", providedJson, query)
	command := exec.Command("/bin/bash", "-c", cmd)

	outputBytes, execErr := command.CombinedOutput()
	if execErr != nil {
		log.Error("Detected failure, building result! Error: ", execErr)

		failureResult := common.CommandOutput{Output: string(outputBytes), Error: execErr.Error()}

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
	providedJson, ok := request.Parameters[jsonKey]
	if !ok {
		return nil, errors.New("no json provided for execution")
	}

	query, ok := request.Parameters[queryKey]
	if !ok {
		return nil, errors.New("no query provided for execution")
	}

	unquoted := ""
	if unquotedKey, ok := request.Parameters[unquotedKey]; ok {
		unquotedBool, err := strconv.ParseBool(unquotedKey)
		if err == nil && unquotedBool {
			unquoted = "--unquoted "
		}
	}

	cmd := fmt.Sprintf("/bin/echo '%s' | /bin/jp %s%s", providedJson, unquoted, query)
	output, err := common.ExecuteCommand(nil, "/bin/bash", "-c", cmd)

	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}
