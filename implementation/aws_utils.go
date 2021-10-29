package implementation

import (
	"math/rand"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	log "github.com/sirupsen/logrus"
)

// access keys have to be both set
// role arn can be supplied alone if it's irsa
// role arn and external id have to be supplied together for traditional assume role
func detectConnectionType(awsCredentials map[string]string) (credsType, key, value string) {
	if awsCredentials[awsAccessKeyId] == "" || awsCredentials[awsSecretAccessKey] == "" {
		if awsCredentials[roleArn] == "" {
			return "", "", ""
		} else {
			return "roleBased", awsCredentials[roleArn], awsCredentials[externalID]
		}
	}
	return "userBased", awsCredentials[awsAccessKeyId], awsCredentials[awsSecretAccessKey]
}

func convertInterfaceMapToStringMap(m map[string]interface{}) map[string]string {
	mapString := make(map[string]string)
	for key, value := range m {
		var strValue string
		strKey := fmt.Sprintf("%v", key)
		if value == nil {
			strValue = ""
		} else {
			strValue = fmt.Sprintf("%v", value)
		}
		mapString[strKey] = strValue
	}
	return mapString
}

func readFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func assumeRoleWithWebIdentity(svc stsiface.STSAPI, role, sessionName string) (string, string, string, error) {
	log.Debug("assuming role with web identity")
	tokenFile, ok := os.LookupEnv("AWS_WEB_IDENTITY_TOKEN_FILE")
	if !ok {
		return "", "", "", fmt.Errorf("token file for irsa not found. make sure your pod is configured correctly and that your service account is created and annotated properly")
	}

	data, err := readFile(tokenFile)
	if err != nil {
		return "", "", "", fmt.Errorf("unable to open web identity token file with error: %w", err)
	}

	result, err := svc.AssumeRoleWithWebIdentity(&sts.AssumeRoleWithWebIdentityInput{
		DurationSeconds:  aws.Int64(3600),
		RoleArn:          aws.String(role),
		RoleSessionName:  aws.String(sessionName),
		WebIdentityToken: aws.String(string(data)),
	})
	if err != nil {
		return "", "", "", err
	}
	return *result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken, err
}

func assumeRoleWithTrustedIdentity(svc stsiface.STSAPI, role, externalID, sessionName string) (string, string, string, error) {
	log.Debug("assuming role with trusted entity")
	result, err := svc.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         &role,
		RoleSessionName: &sessionName,
		ExternalId:      &externalID,
	})
	if err != nil {
		return "", "", "", err
	}
	return *result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken, err
}

func assumeRole(svc stsiface.STSAPI, role, externalID string) (access, secret, sessionToken string, err error) {
	sessionName := strconv.Itoa(rand.Int())

	// irsa does not work with externalID, only the "traditional" assume role does
	if externalID == "" {
		return assumeRoleWithWebIdentity(svc, role, sessionName)
	}
	return assumeRoleWithTrustedIdentity(svc, role, externalID, sessionName)
}
