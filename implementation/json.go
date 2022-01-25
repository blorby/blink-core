package implementation

import (
	"encoding/json"
	"fmt"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strconv"
)

func executeCoreJQAction(e *execution.PrivateExecutionEnvironment, _ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	providedJson, ok := request.Parameters[jsonKey]
	if !ok {
		return nil, errors.New("no json provided for execution")
	}

	query, ok := request.Parameters[queryKey]
	if !ok {
		return nil, errors.New("no query provided for execution")
	}

	file, err := e.WriteToTempFile([]byte(providedJson), "jq")
	if err != nil {
		log.Error("Failed to to write json to file: ", err)
		return nil, err
	}
	
	defer func() {
		rmCmd := fmt.Sprintf("/bin/rm -f %s", file)
		_, _ = common.ExecuteBash(e, request, nil, rmCmd)
	}()

	cmd := fmt.Sprintf("/bin/jq %s %s", query, file)
	outputBytes, execErr := common.ExecuteBash(e, request, nil, cmd)

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

func executeCoreJPAction(e *execution.PrivateExecutionEnvironment, _ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
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
	output, err := common.ExecuteCommand(e, request, nil, "/bin/bash", "-c", cmd)

	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}
