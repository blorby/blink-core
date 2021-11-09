package implementation

import (
	"encoding/json"
	"errors"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/blinkops/blink-sdk/plugin/connections"
	log "github.com/sirupsen/logrus"
	"os"
)

func executeCorePythonAction(execution *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {

	code, ok := request.Parameters[codeKey]
	if !ok {
		return nil, errors.New("no code provided for execution")
	}

	structToBeMarshaled := struct {
		Code        string                                    `json:"code"`
		Context     map[string]interface{}                    `json:"context"`
		Connections map[string]connections.ConnectionInstance `json:"connections"`
	}{Code: code, Context: ctx.GetAllContextEntries(), Connections: ctx.GetAllConnections()}

	rawJsonBytes, err := json.Marshal(structToBeMarshaled)
	if err != nil {
		log.Error("Failed to marshal the code execution request, err: ", err)
		return nil, err
	}

	filePath, err := common.WriteToTempFile(execution, rawJsonBytes, ".temp-blink-py-")
	if err != nil {
		return nil, err
	}
	defer func(name string) {_ = os.Remove(name) }(filePath)

	output, err := common.ExecuteCommand(execution, request, nil, "/bin/python", pythonRunnerPath, "--input", filePath)
	
	resultJson := struct {
		Context map[string]interface{} `json:"context"`
		Log     string                 `json:"log"`
		Output  string                 `json:"output"`
		Error   string                 `json:"error"`
	}{}

	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	err = json.Unmarshal(output, &resultJson)
	if err != nil {
		log.Error("Failed to unmarshal result, err: ", err)
		return nil, err
	}

	if resultJson.Error != "" {
		return common.GetCommandFailureResponse([]byte(resultJson.Output), errors.New(resultJson.Error))
	}

	ctx.ReplaceContext(resultJson.Context)
	return []byte(resultJson.Output), nil
}
