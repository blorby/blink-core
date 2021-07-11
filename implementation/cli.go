package implementation

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"strings"
)

const (
	regionParameterName       = "region"
	commandParameterName      = "command"
	regionEnvironmentVariable = "AWS_DEFAULT_REGION"

	kubernetesUsername = "user"
	kubernetesCluster  = "cluster"
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

	apiServerURL, ok := credentials["kubernetes_api_url"]
	if !ok {
		return nil, errors.New("connection to K8S is invalid")
	}

	verifyCertificate, ok := credentials["verify"]
	if !ok {
		return nil, errors.New("connection to K8S is invalid")
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to K8S CLI wasn't provided")
	}

	temporaryUUID := uuid.NewV4().String()
	temporaryPath := fmt.Sprintf("/tmp/%s", temporaryUUID)
	pathToKubeConfigDirectory := fmt.Sprintf("%s/.kube", temporaryPath)
	pathToKubeConfig := fmt.Sprintf("%s/config", pathToKubeConfigDirectory)

	if output, err := executeCommand(nil, "/bin/mkdir", "-p", pathToKubeConfigDirectory); err != nil {
		if _, err := getCommandFailureResponse(output, err); err != nil {
			return nil, err
		}
	}

	defer func() {
		// Delete kube config directory
		if _, err := executeCommand(nil, "/bin/rm", "-rf", temporaryPath); err != nil {
			log.Errorf("failed to delete kube config, error: %v", err)
		}
	}()

	verify, ok := verifyCertificate.(bool)
	if !ok {
		verify = false
	}

	if err := initKubernetesEnvironment(temporaryPath, fmt.Sprintf("%s", bearerToken), fmt.Sprintf("%s", apiServerURL), verify); err != nil {
		return nil, err
	}

	environmentVariables := []string{
		fmt.Sprintf("KUBECONFIG=%s", pathToKubeConfig),
	}

	command = fmt.Sprintf("--user %s --cluster %s %s", kubernetesUsername, kubernetesCluster, command)
	output, err := executeCommand(environmentVariables, "/bin/kubectl", strings.Split(command, " ")...)
	if err != nil {
		output, err = getCommandFailureResponse(output, err)
		if err != nil {
			return nil, err
		}
	}

	return output, nil
}

func initKubernetesEnvironment(temporaryPath string, bearerToken string, apiServerURL string, verifyCertificate bool) error {
	pathToKubeConfigDirectory := fmt.Sprintf("%s/.kube", temporaryPath)
	pathToKubeConfig := fmt.Sprintf("%s/config", pathToKubeConfigDirectory)

	output, err := executeCommand(nil, "/bin/kubectl", "config", "view", "--merge", "--flatten", ">", pathToKubeConfig)
	if err != nil {
		if _, err := getCommandFailureResponse(output, err); err != nil {
			return err
		}
	}

	environmentVariables := []string{
		fmt.Sprintf("KUBECONFIG=%s", pathToKubeConfig),
	}

	clusterBaseCmd := fmt.Sprintf("config set-cluster cluster")
	cmd := fmt.Sprintf("%s --server=%s", clusterBaseCmd, apiServerURL)
	output, err = executeCommand(environmentVariables, "/bin/kubectl", strings.Split(cmd, " ")...)
	if err != nil {
		if _, err := getCommandFailureResponse(output, err); err != nil {
			return err
		}
	}

	if !verifyCertificate {
		cmd := fmt.Sprintf("%s --insecure-skip-tls-verify=true", clusterBaseCmd)
		output, err = executeCommand(environmentVariables, "/bin/kubectl", strings.Split(cmd, " ")...)
		if err != nil {
			if _, err := getCommandFailureResponse(output, err); err != nil {
				return err
			}
		}
	}

	output, err = executeCommand(environmentVariables, "/bin/kubectl", "config", "set-credentials", "user", fmt.Sprintf("--token=%s", bearerToken))
	if err != nil {
		if _, err := getCommandFailureResponse(output, err); err != nil {
			return err
		}
	}

	return nil
}
