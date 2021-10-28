package implementation

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strconv"
	"math/rand"
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

func assumeRole(role, externalID, region string) (access, secret, sessionToken string, err error) {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})

	svc := sts.New(sess)
	sessionName := strconv.Itoa(rand.Int())

	// irsa does not work with externalID, only the "traditional" assume role does
	if externalID == "" {
		tokenFile, ok := os.LookupEnv("AWS_WEB_IDENTITY_TOKEN_FILE")
		if !ok {
			return access, secret, sessionToken, fmt.Errorf("token file for irsa not found. make sure pod is configured correctly and that your service account is created and properly annotated")
		}

		log.Debug("assuming role with web identity")
		data, err := ioutil.ReadFile(tokenFile)
		if err != nil {
			return access, secret, sessionToken, fmt.Errorf("unable to open web identity token file with error: %w", err)
		}

		fmt.Println("Contents of file:", string(data))
		result, err := svc.AssumeRoleWithWebIdentity(&sts.AssumeRoleWithWebIdentityInput{
			DurationSeconds:  aws.Int64(3600),
			RoleArn:          aws.String(role),
			RoleSessionName:  aws.String(sessionName),
			WebIdentityToken: aws.String(string(data)),
		})
		if err != nil {
			return access, secret, sessionToken, fmt.Errorf("unable to assume web identity role with error: %w", err)
		}
		access, secret, sessionToken = *result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken
	} else {
		log.Debug("assuming role with trusted entity")
		result, err := svc.AssumeRole(&sts.AssumeRoleInput{
			RoleArn:         &role,
			RoleSessionName: &sessionName,
			ExternalId:      &externalID,
		})
		if err != nil {
			return access, secret, sessionToken, fmt.Errorf("unable to assume trusted identity role with error: %w", err)
		}
		access, secret, sessionToken = *result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken
	}
	return access, secret, sessionToken, nil
}