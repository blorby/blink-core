package implementation

import (
	"encoding/json"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

func executeCorePsAction(_ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	flags, ok := request.Parameters[userProviderFlagsKey]
	if !ok {
		flags = ""
	}
	command := exec.Command("/bin/ps", flags)

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
