package implementation

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	"strings"
)

func translateToEnvVars(prefix string, entries map[string]string) []string {
	var envVars []string
	for key, value := range entries {
		envVarValue := fmt.Sprintf("%s_%s=%s", strings.ToUpper(prefix), strings.ToUpper(key), value)
		envVars = append(envVars, envVarValue)
	}

	return envVars
}

func getEnvVarsFromContext(actionContext *plugin.ActionContext) []string {
	contextEntries := map[string]string{}
	if actionContext != nil {
		for contextKey, contextValue := range actionContext.GetAllContextEntries() {
			contextEntries[contextKey] = fmt.Sprintf("%v", contextValue)
		}
	}

	contextEnvVars := translateToEnvVars("CONTEXT", contextEntries)

	var finalEnvVars []string
	finalEnvVars = append(finalEnvVars, contextEnvVars...)

	return finalEnvVars
}

func executeCoreBashAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	code, ok := request.Parameters[codeKey]
	if !ok {
		return nil, errors.New("no code provided for execution")
	}

	environmentVariables := getEnvVarsFromContext(ctx)

	output, err := executeCommand(environmentVariables, "/bin/bash", "-c", fmt.Sprintf("%s", code))
	if err != nil {
		output, err = getCommandFailureResponse(output, err)
		if err != nil {
			return nil, err
		}
	}

	return output, nil
}
