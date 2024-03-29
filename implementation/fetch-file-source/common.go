package fetch_file_source

import (
	"errors"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

const (
	fetchFileUrl          = "url"
	fetchFileDestination  = "destination"
	pathDelimiter         = string(os.PathSeparator)
	defaultParamDelimiter = "?"
)

func GetFileDestination(e *execution.PrivateExecutionEnvironment, fileUrl string, request *plugin.ExecuteActionRequest, paramDelimiter string) (string, error) {
	destination, ok := request.Parameters[fetchFileDestination]

	if !ok {
		destination = e.GetHomeDirectory()
	} else {
		if _, err := common.ExecuteCommand(e, request, nil, "/bin/mkdir", "-p", destination); err != nil {
			log.Debugf("Failed to create requested destination dir: %s", destination)
			destination = e.GetHomeDirectory()
		}
	}

	fileName := extractFilenameFromUrl(fileUrl, paramDelimiter)

	if !strings.HasSuffix(destination, pathDelimiter) {
		destination += pathDelimiter
	}

	return destination + fileName, nil
}

func GetFileUrl(request *plugin.ExecuteActionRequest) (string, error) {
	fileUrl, ok := request.Parameters[fetchFileUrl]

	if !ok {
		return "", errors.New("no file URL provided to fetch")
	}

	return fileUrl, nil
}

func extractFilenameFromUrl(fileUrl string, paramDelimiter string) string {
	var fileName string

	if paramDelimiter == "" {
		paramDelimiter = defaultParamDelimiter
	}

	splitUrl := strings.Split(fileUrl, paramDelimiter)
	_, fileName = filepath.Split(splitUrl[0])

	return fileName
}
