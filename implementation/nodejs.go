package implementation

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/blinkops/blink-sdk/plugin/connections"
	log "github.com/sirupsen/logrus"
)

func executeCoreNodejsAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {

	code, ok := request.Parameters[codeKey]
	if !ok {
		return nil, errors.New("no code provided for execution")
	}

	base64EncodedCode := base64.StdEncoding.EncodeToString([]byte(code))

	structToBeMarshaled := struct {
		Code        string                                    `json:"code"`
		Context     map[string]interface{}                    `json:"context"`
		Connections map[string]connections.ConnectionInstance `json:"connections"`
	}{Code: base64EncodedCode, Context: ctx.GetAllContextEntries(), Connections: ctx.GetAllConnections()}

	rawJsonBytes, err := json.Marshal(structToBeMarshaled)
	if err != nil {
		log.Error("Failed to marshal the code execution request, err: ", err)
		return nil, err
	}

	base64EncodedBytes := base64.StdEncoding.EncodeToString(rawJsonBytes)
	output, err := common.ExecuteCommand(request, nil, "/usr/bin/node", nodejsRunnerPath, "--input", base64EncodedBytes)

	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}
