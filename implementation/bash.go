package implementation

import (
	"encoding/json"
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
			switch contextValue.(type) {
			case string:
				contextEntries[contextKey] = contextValue.(string)
			case map[string]interface{}, []interface{}:
				marshaledValue, err := json.Marshal(contextValue)
				if err == nil {
					contextEntries[contextKey] = string(marshaledValue)
				} else {
					contextEntries[contextKey] = fmt.Sprintf("%v", contextValue)
				}

				contextValueMap, ok := contextValue.(map[string]interface{})

				if ok {
					for key, value := range contextValueMap {
						formattedValue, err := json.Marshal(value)

						if err == nil {
							contextEntries[contextKey+"_"+key] = string(formattedValue)
						} else {
							contextEntries[contextKey+"_"+key] = fmt.Sprintf("%v", value)
						}
					}
				}
			default:
				contextEntries[contextKey] = fmt.Sprintf("%v", contextValue)
			}
		}
	}

	contextEnvVars := translateToEnvVars("CONTEXT", contextEntries)

	var finalEnvVars []string
	finalEnvVars = append(finalEnvVars, contextEnvVars...)

	return finalEnvVars
}

func getConnectionsAsEnvVariables(actionContext *plugin.ActionContext) []string {
	ctxConnections := actionContext.GetAllConnections()
	var connections []string
	for _, connection := range ctxConnections {
		resolvedCredentials, err := connection.ResolveCredentials()
		if err != nil {
			continue
		}
		for key, value := range resolvedCredentials {
			variable := fmt.Sprintf("%s=%v", strings.ToUpper(key), value)
			if len(ctxConnections) > 1 {
				variable = fmt.Sprintf("%s_%s", connection.Name, variable)
			}
			connections = append(connections, variable)
		}
	}

	return connections
}

func executeCoreBashAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	code, ok := request.Parameters[codeKey]
	if !ok {
		return nil, errors.New("no code provided for execution")
	}

	environmentVariables := getEnvVarsFromContext(ctx)
	environmentVariables = append(environmentVariables, getConnectionsAsEnvVariables(ctx)...)
	output, err := executeCommand(environmentVariables, "/bin/bash", "-c", fmt.Sprintf("%s", code))
	if err != nil {
		output, err = getCommandFailureResponse(output, err)
		if err != nil {
			return nil, err
		}
	}

	return output, nil
}
