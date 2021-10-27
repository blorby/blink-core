package implementation

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"strconv"
	"math/rand"
)

func determineConnectionType(awsCredentials map[string]string) (credsType, key, value string) {
	if awsCredentials[awsAccessKeyId] == "" || awsCredentials[awsSecretAccessKey] == "" {
		if awsCredentials[roleArn] == "" || awsCredentials[externalID] == "" {
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

func assumeRole(role, externalID, region string) (string, string, string, error) {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	svc := sts.New(sess)

	sessionName := strconv.Itoa(rand.Int())
	result, err := svc.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         &role,
		RoleSessionName: &sessionName,
		ExternalId: &externalID,
	})

	if err != nil {
		return "", "", "", fmt.Errorf("unable to assume role with error: %w", err)
	}
	return *result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken, nil
}