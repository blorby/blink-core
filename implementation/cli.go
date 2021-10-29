package implementation

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"io/ioutil"
	"os"
	"strings"

	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	errors2 "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type environmentVariables []string

const (
	regionParameterName       = "Region"
	commandParameterName      = "Command"
	fileParameterName         = "file"
	regionEnvironmentVariable = "AWS_DEFAULT_REGION"
	vaultAddress              = "VAULT_ADDR"
	vaultToken                = "VAULT_TOKEN"
	awsAccessKeyId            = "aws_access_key_id"
	awsSecretAccessKey        = "aws_secret_access_key"
	awsSessionToken           = "aws_session_token"
	roleArn                   = "role_arn"
	externalID                = "external_id"
)

func executeCoreAWSAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	credentials, err := ctx.GetCredentials("aws")
	if err != nil {
		return nil, err
	}

	region, ok := request.Parameters[regionParameterName]
	if !ok {
		region = "us-east-1"
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to AWS CLI wasn't provided")
	}

	var environment environmentVariables

	m := convertInterfaceMapToStringMap(credentials)
	sessionType, k, v := detectConnectionType(m)
	switch sessionType {
	case "roleBased":
		sess, _ := session.NewSession(&aws.Config{
			Region: aws.String(region),
		})

		svc := sts.New(sess)
		m[awsAccessKeyId], m[awsSecretAccessKey], m[awsSessionToken], err = assumeRole(svc, k, v)
		if err != nil {
			return nil, fmt.Errorf("unable to assume role with error: %w", err)
		}
	case "userBased":
		m[awsSessionToken] = ""
	default:
		return nil, fmt.Errorf("invalid credentials: make sure access+secret key are supplied OR role_arn+external_id")
	}

	for key, value := range m {
		environment = append(environment, fmt.Sprintf("%s=%v", strings.ToUpper(key), value))
	}

	environment = append(environment, fmt.Sprintf("%s=%v", regionEnvironmentVariable, region))
	output, err := common.ExecuteCommand(request, environment, "/bin/bash", "-c", command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreGITAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	credentials, err := ctx.GetCredentials("github")
	if err != nil {
		return nil, err
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to GIT CLI wasn't provided")
	}

	var environment environmentVariables
	for key, value := range credentials {
		environment = append(environment, fmt.Sprintf("%s=%v", strings.ToUpper(key), value))
	}

	output, err := common.ExecuteCommand(request, environment, "/bin/bash", "-c", command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreKubernetesAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	credentials, err := ctx.GetCredentials("kubernetes")
	if err != nil {
		return nil, err
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

	if output, err := common.ExecuteCommand(nil, nil, "/bin/mkdir", "-p", pathToKubeConfigDirectory); err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	defer func() {
		// Delete kube config directory
		if _, err := common.ExecuteCommand(nil, nil, "/bin/rm", "-r", temporaryPath); err != nil {
			log.Errorf("failed to delete kube config credentials from temporary filesystem, error: %v", err)
		}
	}()

	verify, ok := verifyCertificate.(bool)
	if !ok {
		verify = false
	}

	environment := environmentVariables{
		fmt.Sprintf("KUBECONFIG=%s", pathToKubeConfig),
	}

	if output, err := initKubernetesEnvironment(environment, fmt.Sprintf("%s", bearerToken), fmt.Sprintf("%s", apiServerURL), verify); err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	output, err := common.ExecuteBash(request, environment, command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreVaultAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	credentials, err := ctx.GetCredentials("vault")
	if err != nil {
		return nil, err
	}

	token, ok := credentials[vaultToken]
	if !ok {
		return nil, errors.New("connection to vault is invalid")
	}

	apiServerURL, ok := credentials[vaultAddress]
	if !ok {
		return nil, errors.New("connection to vault is invalid")
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to vault wasn't provided")
	}

	environment := environmentVariables{
		fmt.Sprintf("%s=%s", vaultAddress, apiServerURL),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
	}

	// RUN vault login to connect to the vault at the address provided by the user in the connection.
	if output, err := common.ExecuteCommand(nil, environment, "/usr/bin/vault", "login", token.(string)); err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	// execute the user command
	output, err := common.ExecuteBash(request, environment, command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreKubernetesApplyAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	_, err := ctx.GetCredentials("kubernetes")
	if err != nil {
		return nil, err
	}

	applyFileContents, ok := request.Parameters[fileParameterName]
	if !ok {
		return nil, errors.New("the file is missing from action parameters")
	}

	if len(applyFileContents) == 0 {
		return nil, errors.New("can't run apply action with empty file")
	}

	temporaryUUID := uuid.NewV4().String()
	temporaryPath := fmt.Sprintf("/tmp/kubectl-apply-%s", temporaryUUID)

	err = ioutil.WriteFile(temporaryPath, []byte(applyFileContents), 0664)
	if err != nil {
		return nil, errors2.Wrap(err, "failed creating the apply file")
	}

	defer func() {
		err = os.Remove(temporaryPath)
		if err != nil {
			log.Errorf("Failed to remvoe kubectl apply file with error %v", err)
		}
	}()

	request.Parameters[commandParameterName] = fmt.Sprintf("kubectl apply -f %s", temporaryPath)
	return executeCoreKubernetesAction(ctx, request)
}

func executeCoreGoogleCloudAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	credentials, err := ctx.GetCredentials("gcp")
	if err != nil {
		return nil, err
	}

	gcpCredentials, ok := credentials["credentials"]
	if !ok {
		return nil, errors.New("connection to GCP is invalid")
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to Google Cloud CLI wasn't provided")
	}

	temporaryUUID := uuid.NewV4().String()
	temporaryPath := fmt.Sprintf("/tmp/%s", temporaryUUID)
	pathToConfigDirectory := fmt.Sprintf("%s/.gcp", temporaryPath)
	pathToConfig := fmt.Sprintf("%s/config", pathToConfigDirectory)

	if output, err := common.ExecuteCommand(nil, nil, "/bin/mkdir", "-p", pathToConfigDirectory); err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	defer func() {
		// Delete kube config directory
		if _, err := common.ExecuteCommand(nil, nil, "/bin/rm", "-r", temporaryPath); err != nil {
			log.Errorf("failed to delete kube config credentials from temporary filesystem, error: %v", err)
		}
	}()

	environment := environmentVariables{
		fmt.Sprintf("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=%s", pathToConfig),
	}

	if err := initGoogleCloudEnvironment(temporaryPath, fmt.Sprintf("%s", gcpCredentials)); err != nil {
		return common.GetCommandFailureResponse(nil, err)
	}

	output, err := common.ExecuteCommand(request, environment, "/bin/bash", "-c", command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreAzureAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	credentials, err := ctx.GetCredentials("azure")
	if err != nil {
		return nil, err
	}

	appId, ok := credentials["app_id"]
	if !ok {
		return nil, errors.New("connection to Azure is invalid")
	}

	clientSecret, ok := credentials["client_secret"]
	if !ok {
		return nil, errors.New("connection to Azure is invalid")
	}

	tenantId, ok := credentials["tenant_id"]
	if !ok {
		return nil, errors.New("connection to Azure is invalid")
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to Google Cloud CLI wasn't provided")
	}

	loginCmd := fmt.Sprintf("login --service-principal -u %s -p %s --tenant %s", appId, clientSecret, tenantId)
	if output, err := common.ExecuteCommand(request, environmentVariables{}, "/bin/az", strings.Split(loginCmd, " ")...); err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	output, err := common.ExecuteCommand(request, environmentVariables{}, "/bin/bash", "-c", command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func initKubernetesEnvironment(environment environmentVariables, bearerToken string, apiServerURL string, verifyCertificate bool) ([]byte, error) {

	cmd := fmt.Sprintf("kubectl config set-cluster cluster --server=%s", apiServerURL)
	if !verifyCertificate {
		cmd = fmt.Sprintf("%s --insecure-skip-tls-verify=true", cmd)
	}
	if output, err := common.ExecuteBash(nil, environment, cmd); err != nil {
		return output, err
	}

	cmd = fmt.Sprintf("kubectl config set-credentials user --token=%s", bearerToken)
	output, err := common.ExecuteBash(nil, environment, cmd)
	if err != nil {
		return output, err
	}

	cmd = fmt.Sprintf("kubectl config set-context ctx --cluster=cluster --user=user")
	output, err = common.ExecuteBash(nil, environment, cmd)
	if err != nil {
		return output, err
	}

	cmd = fmt.Sprintf("kubectl config use-context ctx")
	output, err = common.ExecuteBash(nil, environment, cmd)
	if err != nil {
		return output, err
	}

	return nil, nil
}

func initGoogleCloudEnvironment(temporaryPath string, credentials string) error {
	pathToGCPConfigDirectory := fmt.Sprintf("%s/.gcp", temporaryPath)
	pathToGCPConfig := fmt.Sprintf("%s/config", pathToGCPConfigDirectory)

	return os.WriteFile(pathToGCPConfig, []byte(credentials), os.ModePerm)
}
