package implementation

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/blinkops/blink-sdk/plugin/connections"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

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

func executeCoreBashAction(execution *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	code, ok := request.Parameters[codeKey]
	if !ok {
		return nil, errors.New("no code provided for execution")
	}

	environmentVariables := os.Environ()
	environmentVariables = append(environmentVariables, getConnectionsAsEnvVariables(ctx.GetAllConnections())...)

	output, err := common.ExecuteCommand(execution, request, environmentVariables, "/bin/bash", "-c", fmt.Sprintf("%s", code))
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}
