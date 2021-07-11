package implementation

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	"strings"
)

const (
	regionParameterName       = "awsRegion"
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
