package fetch_file_source

import (
	"errors"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

const (
	fetchFileUrl         = "url"
	fetchFileDestination = "destination"
	pathDelimiter        = string(os.PathSeparator)
	currentDir           = "./"
)

func GetFileDestination(fileUrl string, request *plugin.ExecuteActionRequest, paramDelimiter string) (string, error) {
	destination, ok := request.Parameters[fetchFileDestination]

	if !ok {
		destination = getCurrentDirectoryPath()
	} else {
		if _, err := common.ExecuteCommand(nil, "/bin/mkdir", "-p", destination); err != nil {
			log.Debugf("Failed to create requested destination dir: %s", destination)
			destination = getCurrentDirectoryPath()
		}
	}

	fileName := extractFilenameFromUrl(fileUrl, paramDelimiter)

	if !strings.HasSuffix(destination, pathDelimiter) {
		destination += pathDelimiter
	}

	return destination + fileName, nil
}

func getCurrentDirectoryPath() string {
	currentPath, err := os.Getwd()

	if err != nil {
		return currentDir
	}

	return currentPath
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

	if paramDelimiter != "" {
		var dir string

		log.Infof("url: %s, param delimiter: %s", fileUrl, paramDelimiter)
		splitUrl := strings.Split(fileUrl, paramDelimiter)
		log.Infof("split url: %v", splitUrl)
		dir, fileName = filepath.Split(splitUrl[0])
		log.Infof("dir: %s, fileName: %s", dir, fileName)
	} else {
		_, fileName = filepath.Split(fileUrl)
	}

	return fileName
}
