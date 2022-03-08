package implementation

import (
	"encoding/json"
	"errors"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os"
)

func executeCoreNodejsAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {

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

	filePath, err := e.WriteToTempFile(rawJsonBytes, "blink-js-")
	if err != nil {
		return nil, err
	}

	defer func(name string) { _ = os.Remove(name) }(filePath)

	output, err := common.ExecuteCommand(e, request, nil, "/usr/bin/node", nodejsRunnerPath, "--input", filePath)
	if err != nil {
		return common.GetCommandFailureResponse(output, err, true)
	}

	return output, nil
}
