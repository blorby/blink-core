package implementation

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/blinkops/blink-sdk/plugin/connections"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

func executeCorePythonAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {

	code, ok := request.Parameters[userProviderCodeKey]
	if !ok {
		return nil, errors.New("no code provider for execution")
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
	command := exec.Command(
		"/bin/python",
		pythonRunnerPath,
		"--input",
		base64EncodedBytes)

	log.Infof("Executing %s", command.String())

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

	resultJson := struct {
		Context map[string]interface{} `json:"context"`
		Log     string                 `json:"log"`
		Output  string                 `json:"output"`
		Error   string                 `json:"error"`
	}{}

	err = json.Unmarshal(outputBytes, &resultJson)
	if err != nil {
		log.Error("Failed to unmarshal result, err: ", err)
		return nil, err
	}

	ctx.ReplaceContext(resultJson.Context)

	result := FinalOutput{Output: resultJson.Output, Error: resultJson.Error}
	finalJsonBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return finalJsonBytes, nil
}
