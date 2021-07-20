package implementation

import (
	"github.com/blinkops/blink-core/implementation/fetch-file-source/curl"
	"github.com/blinkops/blink-core/implementation/fetch-file-source/github"
	"github.com/blinkops/blink-core/implementation/fetch-file-source/gitlab"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
)

func executeCoreFetchFileAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	if github.CheckForConnection(ctx) {
		log.Infof("Fetching file from GitHub")
		return github.FetchFile(ctx, request)
	} else if gitlab.CheckForConnection(ctx) {
		log.Infof("Fetching file from GitLab")
		return gitlab.FetchFile(ctx, request)
	}

	log.Infof("Fetching file")
	return curl.FetchFile(request)
}
