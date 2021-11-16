package implementation

import (
	"encoding/json"
	"errors"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/blinkops/blink-sdk/plugin/connections"
	log "github.com/sirupsen/logrus"
	"os"
)

func executeCoreNodejsAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {

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

	filePath, err := common.WriteToTempFile(rawJsonBytes, "blink-js-")
	if err != nil {
		return nil, err
	}
	defer func(name string) {_ = os.Remove(name) }(filePath)

	output, err := common.ExecuteCommand(request, nil, "/usr/bin/node", nodejsRunnerPath, "--input", filePath)

	if err != nil {
		return output, err
	}

	return output, nil
}
