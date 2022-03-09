package implementation

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
)

const (
	packageKey = "package"
)

func executeInstallAction(_ *execution.PrivateExecutionEnvironment, _ *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	pkg, ok := request.Parameters[packageKey]
	if !ok {
		return nil, errors.New("missing mandatory parameter: package")
	}

	updateCmd := exec.Command("/usr/bin/apt-get", "update")
	if out, err := updateCmd.CombinedOutput(); err != nil {
		return out, fmt.Errorf("apt-get update: %w", err)
	}

	installCmd := exec.Command("/usr/bin/apt-get", "install", "-y", pkg)
	out, err := installCmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("apt-get install: %w", err)
	}

	return out, nil
}
