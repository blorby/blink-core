package github

import (
	"errors"
	"fmt"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-core/implementation/fetch-file-source"
	"github.com/blinkops/blink-sdk/plugin"
)

const (
	fileSourceGitHub    = "github"
	gitHubTokenKey      = "token"
	headerAuthorization = "'Authorization: token %s'"
	paramDelimiter      = "?"
)

func CheckForConnection(ctx *plugin.ActionContext) bool {
	_, err := ctx.GetCredentials(fileSourceGitHub)

	if err != nil {
		return false
	}

	return true
}

func FetchFile(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	fileUrl, err := fetch_file_source.GetFileUrl(request)

	if err != nil {
		return nil, err
	}

	destination, err := fetch_file_source.GetFileDestination(e, fileUrl, request, paramDelimiter)

	token, err := getConnnection(ctx)

	if err != nil {
		return nil, err
	}

	tokenHeader := fmt.Sprintf(headerAuthorization, token)
	output, err := common.ExecuteCommand(e, request, nil, "/usr/bin/curl", "-H", tokenHeader, "-o", destination, fileUrl)

	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return []byte(destination), nil
}

func getConnnection(ctx *plugin.ActionContext) (string, error) {
	gitCredentials, err := ctx.GetCredentials(fileSourceGitHub)

	if err != nil {
		return "", err
	}

	gitToken, ok := gitCredentials[gitHubTokenKey]

	if !ok {
		return "", errors.New("no GitHub token provided")
	}

	return gitToken, nil
}
