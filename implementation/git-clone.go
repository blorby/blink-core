package implementation

import (
	"errors"
	"github.com/blinkops/blink-sdk/plugin"
	"net/url"
	"regexp"
)

func executeCoreGitCloneAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	token, err := getGitToken(ctx)

	if err != nil {
		return nil, err
	}

	repository, ok := request.Parameters[gitRepositoryKey]

	if !ok {
		return nil, errors.New("no repository provided for git-clone")
	}

	branch, ok := request.Parameters[gitBranchKey]

	if !ok {
		return nil, errors.New("no branch provided for git-clone")
	}

	repoUrl, err := url.Parse(repository)

	if err != nil {
		return nil, errors.New("failed parsing repository url")
	}

	repoUrl.User = url.User(token)
	output, err := executeCommand(nil, gitClonePath, "/bin/git", "clone", "--single-branch", "--branch", branch, repoUrl.String())

	if err != nil {
		output, err = getCommandFailureResponse(output, err)

		if err != nil {
			return nil, err
		}

		return output, nil
	}

	repoNameRE := regexp.MustCompile("^Cloning into '(.+)'\\.\\.\\.")
	repoNameMatch := repoNameRE.FindStringSubmatch(string(output))

	if len(repoNameMatch) >= 2 {
		return []byte(gitClonePath + "/" + repoNameMatch[1]), nil
	}

	output, err = getCommandFailureResponse(output, errors.New("failed cloning git repository"))

	if err != nil {
		return nil, err
	}

	return output, nil
}

func getGitToken(ctx *plugin.ActionContext) (string, error) {
	gitCredentials, err := ctx.GetCredentials("git")

	if err != nil {
		return "", err
	}

	gitToken, ok := gitCredentials["token"]

	if !ok {
		return "", errors.New("no git token provided")
	}

	tokenString, ok := gitToken.(string)

	if !ok {
		return "", errors.New("invalid git token provided")
	}

	return tokenString, nil
}
