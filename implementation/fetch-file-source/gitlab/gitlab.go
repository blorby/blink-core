package gitlab

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/fetch-file-source"
	"github.com/blinkops/blink-sdk/plugin"
)

const (
	fileSourceGitLab    = "gitlab"
	gitLabTokenKey      = "token"
	headerAuthorization = "'Private-Token: %s'"
	paramDelimiter      = "\\?"
)

func CheckForConnection(ctx *plugin.ActionContext) bool {
	_, err := ctx.GetCredentials(fileSourceGitLab)

	if err != nil {
		return false
	}

	return true
}

func FetchFile(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	fileUrl, err := fetch_file_source.GetFileUrl(request)

	if err != nil {
		return nil, err
	}

	destination, err := fetch_file_source.GetFileDestination(fileUrl, request, paramDelimiter)

	token, err := getConnnection(ctx)

	if err != nil {
		return nil, err
	}

	tokenHeader := fmt.Sprintf(headerAuthorization, token)
	output, err := common.ExecuteCommand(nil, "/bin/curl", "-H", tokenHeader, "-o", destination, fileUrl)

	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return []byte(destination), nil
}

func getConnnection(ctx *plugin.ActionContext) (string, error) {
	gitCredentials, err := ctx.GetCredentials(fileSourceGitLab)

	if err != nil {
		return "", err
	}

	gitToken, ok := gitCredentials[gitLabTokenKey]

	if !ok {
		return "", errors.New("no GitLab token provided")
	}

	tokenString, ok := gitToken.(string)

	if !ok {
		return "", errors.New("invalid GitLab token provided")
	}

	return tokenString, nil
}
