package implementation

import "github.com/blinkops/blink-sdk/plugin/connections"

type RunnerCodeStructure struct {
	Code        string                                     `json:"code"`
	Context     map[string]interface{}                     `json:"context"`
	Connections map[string]*connections.ConnectionInstance `json:"connections"`
}

type RunnerCodeResponse struct {
	Context map[string]interface{} `json:"context"`
	Log     string                 `json:"log"`
	Output  string                 `json:"output"`
	Error   string                 `json:"error"`
}
