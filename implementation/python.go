package implementation

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
)

func executeCorePythonAction(execution *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	code, ok := request.Parameters[codeKey]
	if !ok {
		return nil, errors.New("no code provided for execution")
	}

	structToBeMarshaled := RunnerCodeStructure{Code: code, Context: ctx.GetAllContextEntries(), Connections: ctx.GetAllConnections()}
	rawJsonBytes, err := json.Marshal(structToBeMarshaled)
	if err != nil {
		log.Error("Failed to marshal the code execution request, err: ", err)
		return nil, err
	}

	filePath, err := common.WriteToTempFile(execution, rawJsonBytes, ".temp-blink-py-")
	if err != nil {
		return nil, err
	}
	defer func(name string) { _ = os.Remove(name) }(filePath)

	output, err := common.ExecuteCommand(execution, request, nil, "/bin/python", pythonRunnerPath, "--input", filePath)
	if err != nil {
		return common.GetCommandFailureResponse(output, err, true)
	}

	resultJson := RunnerCodeResponse{}
	if err = json.Unmarshal(output, &resultJson); err != nil {
		log.Error("Failed to unmarshal result, err: ", err)
		return nil, err
	}

	if resultJson.Error != "" {
		return common.GetCommandFailureResponse([]byte(resultJson.Output), errors.New(resultJson.Error), true)
	}

	ctx.ReplaceContext(resultJson.Context)
	return []byte(resultJson.Output), nil
}
