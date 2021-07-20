package implementation

import (
	"github.com/blinkops/blink-core/implementation/fetch-file-source/curl"
	"github.com/blinkops/blink-core/implementation/fetch-file-source/github"
	"github.com/blinkops/blink-core/implementation/fetch-file-source/gitlab"
	"github.com/blinkops/blink-sdk/plugin"
)

func executeCoreFetchFileAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	if github.CheckForConnection(ctx) {
		return github.FetchFile(ctx, request)
	} else if gitlab.CheckForConnection(ctx) {
		return gitlab.FetchFile(ctx, request)
	}

	return curl.FetchFile(request)
}
