package implementation

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/blinkops/blink-core/implementation/execution"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/pkg/errors"
)

const (
	regionParameterName       = "Region"
	commandParameterName      = "Command"
	fileParameterName         = "file"
	regionEnvironmentVariable = "AWS_DEFAULT_REGION"
	vaultAddress              = "VAULT_ADDR"
	vaultToken                = "VAULT_TOKEN"
	terraformAddress          = "TERRAFORM_ADDR"
	terraformToken            = "TERRAFORM_TOKEN"
	awsAccessKeyId            = "aws_access_key_id"
	awsSecretAccessKey        = "aws_secret_access_key"
	awsSessionToken           = "aws_session_token"
	roleArn                   = "role_arn"
	externalID                = "external_id"
)

func executeAwsCli(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	return executeCoreAWSAction(e, ctx, request, "aws")
}

func executeEksctlCli(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	return executeCoreAWSAction(e, ctx, request, "eksctl")
}

func executeCoreAWSAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest, cliCommand string) ([]byte, error) {
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

	var environment []string

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
	user, err := e.CreateCliUser(cliCommand, environment)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cli user")
	}
	defer e.CleanupCliUser(user.Username)

	awsUsernameEnv := fmt.Sprintf("AWS_USER=%s", user.Username)
	output, err := common.ExecuteCommand(e, request, []string{awsUsernameEnv}, "/bin/bash", "-c", command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreGITAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	credentials, err := ctx.GetCredentials("github")
	if err != nil {
		return nil, err
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to GIT CLI wasn't provided")
	}

	var environment []string
	for key, value := range credentials {
		environment = append(environment, fmt.Sprintf("%s=%v", strings.ToUpper(key), value))
	}

	user, err := e.CreateCliUser("git", environment)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cli user")
	}
	defer e.CleanupCliUser(user.Username)

	gitUsernameEnv := fmt.Sprintf("GIT_USER=%s", user.Username)
	output, err := common.ExecuteCommand(e, request, []string{gitUsernameEnv}, "/bin/bash", "-c", command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreKubernetesAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	return kubectl(e, ctx, request, nil)
}

type prepareFunc func(e *execution.PrivateExecutionEnvironment) (string, error)

func kubectl(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest, prepFn prepareFunc) ([]byte, error) {
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

	cliUser, err := e.CreateCliUser("kubectl", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cli user")
	}
	defer e.CleanupCliUser(cliUser.Username)
	ce := e.CreateCliUserPee(cliUser)

	pathToKubeConfigDirectory := path.Join(ce.GetHomeDirectory(), ".kube")
	if err = ce.CreateDirectory(pathToKubeConfigDirectory); err != nil {
		return nil, errors.Wrap(err, "Failed to create kube config directory")
	}

	verify, ok := verifyCertificate.(bool)
	if !ok {
		verify = false
	}

	if output, err := initKubernetesEnvironment(ce, nil, fmt.Sprintf("%s", bearerToken), fmt.Sprintf("%s", apiServerURL), verify); err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	if prepFn != nil {
		realCmd, err := prepFn(ce)
		if err != nil {
			return nil, err
		}
		request.Parameters[commandParameterName] = realCmd
	}

	k8sUserEnv := fmt.Sprintf("K8S_USER=%s", cliUser.Username)
	output, err := common.ExecuteBash(e, request, []string{k8sUserEnv}, command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreVaultAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
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

	environment := []string{
		fmt.Sprintf("%s=%s", vaultAddress, apiServerURL),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
	}
	cliUser, err := e.CreateCliUser("vault", environment)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cli user")
	}
	defer e.CleanupCliUser(cliUser.Username)
	ce := e.CreateCliUserPee(cliUser)
	vaultUsernameEnv := fmt.Sprintf("VAULT_USER=%s", cliUser.Username)

	// RUN vault login to connect to the vault at the address provided by the user in the connection.
	if output, err := common.ExecuteCommand(ce, nil, environment, "/opt/blink/vault", "login", token.(string)); err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	// execute the user command
	output, err := common.ExecuteBash(e, request, []string{vaultUsernameEnv}, command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreTerraFormAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	var environment []string

	cliUser, err := e.CreateCliUser("terraform", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cli user")
	}
	defer e.CleanupCliUser(cliUser.Username)
	ce := e.CreateCliUserPee(cliUser)

	terraformUsernameEnv := fmt.Sprintf("TERRAFORM_USER=%s", cliUser.Username)

	// Use either TerraForm or AWS credentials
	credentials, err := ctx.GetCredentials("terraform")
	if err != nil {
		awsCredentials, err := ctx.GetCredentials("aws")
		if err != nil {
			return nil, errors.New("connection with terraform or aws is missing from action context")
		}

		m := convertInterfaceMapToStringMap(awsCredentials)
		for key, value := range m {
			environment = append(environment, fmt.Sprintf("%s=%v", strings.ToUpper(key), value))
		}
		if err := ce.WriteProfileFile(cliUser, environment); err != nil {
			return nil, errors.Wrap(err, "failed creating profile file")
		}

	} else {
		tokenRaw, ok := credentials[terraformToken]
		if !ok {
			return nil, errors.New("connection to terraform is invalid")
		}

		apiServerURLRaw, ok := credentials[terraformAddress]
		if !ok {
			return nil, errors.New("connection to terraform is invalid")
		}

		token, ok := tokenRaw.(string)
		if !ok {
			return nil, errors.New("Terraform token is not a string")
		}

		apiServerURL, ok := apiServerURLRaw.(string)
		if !ok {
			return nil, errors.New("Api server url is not a string")
		}

		// Create credentials file if it doesn't exist and write the user's credentials to it
		_, err := createTerraFormCredentialsFile(ce, apiServerURL, token)
		if err != nil {
			return nil, err
		}
	}

	command, ok := request.Parameters[commandParameterName]
	if !ok {
		return nil, errors.New("command to terraform wasn't provided")
	}

	// Validate the command to check that it doesn't require input, since it can't be supplied through the cli
	output, err := validateTerraFormCommand(command)
	if err != nil {
		return output, err
	}

	//environment = append(environment, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))

	// Execute the user's command
	output, err = common.ExecuteBash(e, request, []string{terraformUsernameEnv}, command)
	if err != nil {
		if strings.Contains(string(output), "Couldn't find an alternative") {
			return nil, errors.New("terraform commands must start with \"terraform\" prefix")
		}

		_, commandFailureResponse := common.GetCommandFailureResponse(output, err)

		// Replace characters which make the output unreadable
		outputStr := fixTerraFormOutput(commandFailureResponse.Error())

		return nil, errors.New(outputStr)
	}

	return []byte(fixTerraFormOutput(string(output))), nil
}

func executeCoreKubernetesApplyAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	applyFileContents, ok := request.Parameters[fileParameterName]
	if !ok {
		return nil, errors.New("the file is missing from action parameters")
	}

	if len(applyFileContents) == 0 {
		return nil, errors.New("can't run apply action with empty file")
	}

	return kubectl(e, ctx, request, func(ce *execution.PrivateExecutionEnvironment) (string, error) {
		tempPath := path.Join(ce.GetHomeDirectory(), "kubectl-apply")
		err := ce.WriteFile([]byte(applyFileContents), tempPath)
		if err != nil {
			return "", errors.Wrap(err, "failed creating the apply file")
		}
		return fmt.Sprintf("kubectl apply -f %s", tempPath), nil
	})
}

func executeCoreGoogleCloudAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
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

	user, err := e.CreateCliUser("gcloud", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cli user")
	}
	defer e.CleanupCliUser(user.Username)
	ce := e.CreateCliUserPee(user)

	gcloudUsernameEnv := fmt.Sprintf("GCLOUD_USER=%s", user.Username)

	pathToConfig, err := initGoogleCloudEnvironment(ce, fmt.Sprintf("%s", gcpCredentials))
	if err != nil {
		return common.GetCommandFailureResponse(nil, err)
	}

	cliEnv := []string{
		fmt.Sprintf("CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=%s", pathToConfig),
	}
	if err := ce.WriteProfileFile(user, cliEnv); err != nil {
		return nil, errors.Wrap(err, "failed writing profile file")
	}

	output, err := common.ExecuteCommand(e, request, []string{gcloudUsernameEnv}, "/bin/bash", "-c", command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func executeCoreAzureAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
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

	cliUser, err := e.CreateCliUser("az", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cli user")
	}
	defer e.CleanupCliUser(cliUser.Username)
	ce := e.CreateCliUserPee(cliUser)

	azureUsernameEnv := fmt.Sprintf("AZURE_USER=%s", cliUser.Username)

	loginCmd := fmt.Sprintf("login --service-principal -u %s -p %s --tenant %s", appId, clientSecret, tenantId)
	if output, err := common.ExecuteCommand(ce, request, []string{}, "/opt/blink/az", strings.Split(loginCmd, " ")...); err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	output, err := common.ExecuteCommand(e, request, []string{azureUsernameEnv}, "/bin/bash", "-c", command)
	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return output, nil
}

func initKubernetesEnvironment(e *execution.PrivateExecutionEnvironment, environment []string, bearerToken string, apiServerURL string, verifyCertificate bool) ([]byte, error) {

	output, err := common.ExecuteBash(e, nil, environment, "whoami && pwd && env")
	if err != nil {
		return output, err
	}
	log.Infof("whoami && pwd && env output: %s", output)

	cmd := fmt.Sprintf("/opt/blink/kubectl config set-cluster cluster --server=%s", apiServerURL)
	if !verifyCertificate {
		cmd = fmt.Sprintf("%s --insecure-skip-tls-verify=true", cmd)
	}
	if output, err := common.ExecuteBash(e, nil, environment, cmd); err != nil {
		return output, err
	}

	cmd = fmt.Sprintf("/opt/blink/kubectl config set-credentials user --token=%s", bearerToken)
	output, err = common.ExecuteBash(e, nil, environment, cmd)
	if err != nil {
		return output, err
	}

	cmd = fmt.Sprintf("/opt/blink/kubectl config set-context ctx --cluster=cluster --user=user")
	output, err = common.ExecuteBash(e, nil, environment, cmd)
	if err != nil {
		return output, err
	}

	cmd = fmt.Sprintf("/opt/blink/kubectl config use-context ctx")
	output, err = common.ExecuteBash(e, nil, environment, cmd)
	if err != nil {
		return output, err
	}

	return nil, nil
}

func initGoogleCloudEnvironment(e *execution.PrivateExecutionEnvironment, credentials string) (string, error) {
	pathToGCPConfigDirectory := path.Join(e.GetHomeDirectory(), ".gcp")
	if err := e.CreateDirectory(pathToGCPConfigDirectory); err != nil {
		return "", errors.Wrap(err, "Failed to create .gcp sub-directory: ")
	}
	pathToGCPConfig := path.Join(pathToGCPConfigDirectory, "config")

	err := e.WriteToFile(pathToGCPConfig, []byte(credentials))
	return pathToGCPConfig, err
}

func createTerraFormCredentialsFile(e *execution.PrivateExecutionEnvironment, apiServerURL string, token string) (string, error) {

	// Create TerraForm credentials file
	tempDir := e.GetHomeDirectory()
	terraformDir := path.Join(tempDir, ".terraform.d")
	credentialsJsonPath := path.Join(terraformDir, "credentials.tfrc.json")

	err := e.CreateDirectory(terraformDir)
	if err != nil {
		return "", err
	}
	content := fmt.Sprintf(`{
  "credentials": {
    "%s": {
      "token": "%s"
    }
  }
}`, apiServerURL, token)

	err = e.WriteFile([]byte(content), credentialsJsonPath)
	if err != nil {
		return "", err
	}

	return terraformDir, nil
}

func validateTerraFormCommand(command string) ([]byte, error) {
	if strings.HasPrefix(command, "destroy") {
		return nil, errors.New("please use \"apply -destroy -auto-approve\" in order to auto approve your changes")
	}

	autoApplyExists := false
	destroyExists := false
	if strings.HasPrefix(command, "apply") {
		for _, field := range strings.Fields(command) {
			if field == "-auto-approve" {
				autoApplyExists = true
			} else if field == "-destroy" {
				destroyExists = true
			}
		}

		if !autoApplyExists {
			if !destroyExists {
				return nil, errors.New("please use \"apply -auto-approve\" in order to auto approve your changes")
			}
			return nil, errors.New("please use \"apply -destroy -auto-approve\" in order to auto approve your changes")
		}
	}

	return nil, nil
}

func fixTerraFormOutput(output string) string {
	exp1 := regexp.MustCompile("\\[[0-9]+m")
	exp2 := regexp.MustCompile("[╷│╵\u001B]+?")
	exp3 := regexp.MustCompile(" +")
	output = exp1.ReplaceAllString(output, "")
	output = exp2.ReplaceAllString(output, "")
	output = exp3.ReplaceAllString(output, " ")
	return output
}
