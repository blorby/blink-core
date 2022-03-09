package common

import "errors"

var CLIError = errors.New("CLI Error")

const (
	ErrorCodeOK       int64 = 0
	ErrorCodeCLIError int64 = 999
)
