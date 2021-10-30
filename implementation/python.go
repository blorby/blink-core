package implementation

import (
	"encoding/json"
	"errors"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/blinkops/blink-sdk/plugin/connections"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

func executeCorePythonAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {

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

	filePath, err := writeToTempFile(rawJsonBytes)
	if err != nil {
		return nil, err
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Error("Failed to remove temp file ", err)
		}
	}(filePath)

	output, err := common.ExecuteCommand(request, nil, "/bin/python", pythonRunnerPath, "--input", filePath)
	
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

func writeToTempFile(bytes []byte) (string, error) {
	file, err := ioutil.TempFile("/tmp", "blink-py-")
	if err != nil {
		return "", err
	}

	defer func() {
		// Close the file
		if err := file.Close(); err != nil {
			log.Error("failed to close file", err)
		}
	}()

	_, err = file.Write(bytes)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}
