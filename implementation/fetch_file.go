package implementation

import (
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-core/implementation/fetch-file-source/curl"
	"github.com/blinkops/blink-core/implementation/fetch-file-source/github"
	"github.com/blinkops/blink-core/implementation/fetch-file-source/gitlab"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
)

func executeCoreFetchFileAction(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	filePath, err := fetchFile(e, ctx, request)
	if err != nil {
		return []byte{}, err
	}

	if showOutput, ok := request.Parameters["show_output"]; ok {
		if val, _ := strconv.ParseBool(showOutput); val {
			return os.ReadFile(string(filePath))
		}
	}

	return filePath, nil
}

func fetchFile(e *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	if github.CheckForConnection(ctx) {
		log.Infof("Fetching file from GitHub")
		return github.FetchFile(e, ctx, request)
	} else if gitlab.CheckForConnection(ctx) {
		log.Infof("Fetching file from GitLab")
		return gitlab.FetchFile(e, ctx, request)
	}

	log.Infof("Fetching file")
	return curl.FetchFile(e, request)
}
