package implementation

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/blinkops/blink-sdk/plugin/connections"
	log "github.com/sirupsen/logrus"
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

func getConnectionsAsEnvVariables(ctxConnections map[string]connections.ConnectionInstance) []string {

	var resolvedConnections []string
	for _, connection := range ctxConnections {
		resolvedCredentials, err := connection.ResolveCredentials()
		if err != nil {
			log.Errorf("failed to resolve connection: \"%s\", credentials, error: %v", connection.Name, err)
			continue
		}

		for credentialEntry, credentialValue := range resolvedCredentials {
			variable := fmt.Sprintf("%s_%s=%v", strings.ToUpper(connection.Name), strings.ToUpper(credentialEntry), credentialValue)
			resolvedConnections = append(resolvedConnections, variable)
		}
	}

	return resolvedConnections
}

func executeCoreBashAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	code, ok := request.Parameters[codeKey]
	if !ok {
		return nil, errors.New("no code provided for execution")
	}

	environmentVariables := getEnvVarsFromContext(ctx)
	environmentVariables = append(environmentVariables, getConnectionsAsEnvVariables(ctx.GetAllConnections())...)
	output, err := common.ExecuteCommand(environmentVariables, "/bin/bash", "-c", fmt.Sprintf("%s", code))
	if err != nil {
		output, err = common.GetCommandFailureResponse(output, err)
		if err != nil {
			return nil, err
		}
	}

	return output, nil
}
