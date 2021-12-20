package implementation

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	"os"
)

func executeCoreBashAction(execution *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	code, ok := request.Parameters[codeKey]
	if !ok {
		return nil, errors.New("no code provided for execution")
	}

	environmentVariables := os.Environ()

	output, err := common.ExecuteCommand(execution, request, environmentVariables, "/bin/bash", "-c", fmt.Sprintf("%s", code))
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}
