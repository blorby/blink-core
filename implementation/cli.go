package implementation

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	uuid "github.com/satori/go.uuid"
	"strings"
)

const (
	regionParameterName       = "region"
	commandParameterName      = "command"
	regionEnvironmentVariable = "AWS_DEFAULT_REGION"
)

func executeCoreAWSAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	credentials, err := ctx.GetCredentials("aws")
	if err != nil {
		return nil, errors.New("connection to AWS wasn't provided")
	}

	region, ok := request.Parameters[regionParameterName]
	if !ok {
		return nil, errors.New("region to AWS CLI wasn't provided")
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to AWS CLI wasn't provided")
	}

	var environmentVariables []string
	for key, value := range credentials {
		environmentVariables = append(environmentVariables, fmt.Sprintf("%s=%v", strings.ToUpper(key), value))
	}

	environmentVariables = append(environmentVariables, fmt.Sprintf("%s=%v", regionEnvironmentVariable, region))

	output, err := executeCommand(environmentVariables, "/bin/aws", strings.Split(command, " ")...)
	if err != nil {
		output, err = getCommandFailureResponse(output, err)
		if err != nil {
			return nil, err
		}
	}

	return output, nil
}

func executeCoreKubernetesAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	credentials, err := ctx.GetCredentials("kubernetes")
	if err != nil {
		return nil, errors.New("connection to K8S wasn't provided")
	}

	bearerToken, ok := credentials["bearer_token"]
	if !ok {
		return nil, errors.New("connection to K8S is invalid")
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to K8S CLI wasn't provided")
	}

	contextEntries := ctx.GetAllContextEntries()
	executionId, ok := contextEntries["execution_id"]
	if !ok {
		executionId = uuid.NewV4().String()
	}

	pathToKubeConfig := fmt.Sprintf("/tmp/%s/.kube/config", executionId)

	environmentVariables := []string{
		fmt.Sprintf("KUBECONFIG=%s", pathToKubeConfig),
	}

	output, err := executeCommand(environmentVariables, "/bin/kubectl", "config", "--set-credentials", "user", fmt.Sprintf("--token=%s", bearerToken))
	if err != nil {
		output, err = getCommandFailureResponse(output, err)
		if err != nil {
			return nil, err
		}
	}

	command = fmt.Sprintf("--user user %s", command)

	output, err = executeCommand(environmentVariables, "/bin/kubectl", strings.Split(command, " ")...)
	if err != nil {
		output, err = getCommandFailureResponse(output, err)
		if err != nil {
			return nil, err
		}
	}

	return output, nil
}
