package curl

import (
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/fetch-file-source"
	"github.com/blinkops/blink-sdk/plugin"
)

func FetchFile(request *plugin.ExecuteActionRequest) ([]byte, error) {
	fileUrl, err := fetch_file_source.GetFileUrl(request)

	if err != nil {
		return nil, err
	}

	destination, err := fetch_file_source.GetFileDestination(fileUrl, request, "")

	output, err := common.ExecuteCommand(nil, "/bin/curl", "-o", destination, fileUrl)

	if err != nil {
		return common.GetCommandFailureResponse(output, err)
	}

	return []byte(destination), nil
}
