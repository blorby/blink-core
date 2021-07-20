package fetch_file_source

import (
	"errors"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"strings"
)

const (
	fetchFileUrl         = "url"
	fetchFileDestination = "destination"
	pathDelimiter        = "/"
)

func GetFileDestination(fileUrl string, request *plugin.ExecuteActionRequest, paramDelimiter string) (string, error) {
	destination, ok := request.Parameters[fetchFileDestination]

	if !ok {
		destination = getCurrentDirectoryPath()
	} else {
		_, err := common.ExecuteCommand(nil, "/bin/mkdir", "-p", destination)

		if err != nil {
			log.Debugf("Failed to create requested destination dir: %s", destination)
			destination = getCurrentDirectoryPath()
		}
	}

	splitUrl := strings.Split(fileUrl, pathDelimiter)
	fileNameParams := splitUrl[len(splitUrl)-1]
	fileName := fileNameParams

	if paramDelimiter != "" {
		splitFileName := strings.Split(fileNameParams, paramDelimiter)
		fileName = splitFileName[0]
	}

	if !strings.HasSuffix(destination, pathDelimiter) {
		destination += pathDelimiter
	}

	return destination + fileName, nil
}

func getCurrentDirectoryPath() string {
	output, err := common.ExecuteCommand(nil, "pwd")

	if err != nil {
		return "./"
	}

	return string(output)
}

func GetFileUrl(request *plugin.ExecuteActionRequest) (string, error) {
	fileUrl, ok := request.Parameters[fetchFileUrl]

	if !ok {
		return "", errors.New("no file URL provided to fetch")
	}

	return fileUrl, nil
}
