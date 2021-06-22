package implementation

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

func executeCommand(environment []string, name string, args ...string) ([]byte, error) {
	command := exec.Command(
		name,
		args...)

	if environment != nil {
		command.Env = environment
	}

	log.Infof("Executing %s", command.String())

	outputBytes, execErr := command.CombinedOutput()
	if execErr != nil {
		log.Errorf("Detected failure, building result! Error: %v", execErr)
	}

	return outputBytes, execErr
}

func getCommandFailureResponse(output []byte, err error) ([]byte, error) {
	failureResult := CommandOutput{Output: string(output), Error: err.Error()}

	resultBytes, err := json.Marshal(failureResult)
	if err != nil {
		log.Error("Failed to properly marshal result, err: ", err)
		return nil, err
	}

	return resultBytes, nil
}
