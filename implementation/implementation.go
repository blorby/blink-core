package implementation

import (
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/blinkops/blink-sdk/plugin/actions"
	"github.com/blinkops/blink-sdk/plugin/config"
	"github.com/blinkops/blink-sdk/plugin/connections"
	description2 "github.com/blinkops/blink-sdk/plugin/description"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"path"
)

var errActionNotFound = errors.New("Action not found")

type ActionHandler func(execution *execution.PrivateExecutionEnvironment, ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error)

type CorePlugin struct {
	description      plugin.Description
	actions          []plugin.Action
	supportedActions map[string]ActionHandler
}

func (p *CorePlugin) Describe() plugin.Description {
	log.Debug("Handling Describe request!")
	return p.description
}

func (p *CorePlugin) GetActions() []plugin.Action {
	log.Debug("Handling GetActions request!")
	return p.actions
}

func (p *CorePlugin) TryRouteExecutionRelatedAction(actionName string, request *plugin.ExecuteActionRequest) ([]byte, error) {

	switch actionName {
	case execution.StopExecutionSessionAction:
		return execution.StopPrivateExecution(request)
	default:
		break
	}

	return nil, errActionNotFound
}

func (p *CorePlugin) ExecuteAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) (*plugin.ExecuteActionResponse, error) {
	log.Debugf("Executing action: %v\n Context: %v", *request, ctx.GetAllContextEntries())

	resultBytes, err := p.TryRouteExecutionRelatedAction(request.Name, request)
	if err == errActionNotFound {

		executionId := ctx.GetAllContextEntries()["execution_id"]
		if executionId == nil {
			return nil, errors.New("Execution id is missing from context")
		}

		executionIdCasted, ok := executionId.(string)
		if !ok {
			return nil, errors.New("Execution id is not a string...")
		}

		session, err := execution.AcquirePrivateExecutionSession(executionIdCasted)
		if err != nil {
			return nil, err
		}

		actionHandler, ok := p.supportedActions[request.Name]
		if !ok {
			return nil, errors.New("action is not supported: " + request.Name)
		}

		resultBytes, err = actionHandler(session, ctx, request)
		if err != nil {
			log.Error("Failed executing action, err: ", err)
			return nil, err
		}
	}

	if err != nil && err != errActionNotFound {
		return nil, err
	}

	log.Debugf("Finished executing action: %v", request)

	if len(resultBytes) > 0 && resultBytes[len(resultBytes)-1] == '\n' {
		resultBytes = resultBytes[:len(resultBytes)-1]
	}

	errorCode := int64(0)
	if err != nil {
		errorCode = 1
	}

	return &plugin.ExecuteActionResponse{
		ErrorCode: errorCode,
		Result:    resultBytes,
	}, nil
}

func (p *CorePlugin) TestCredentials(_ map[string]connections.ConnectionInstance) (*plugin.CredentialsValidationResponse, error) {
	return &plugin.CredentialsValidationResponse{
		AreCredentialsValid:   true,
		RawValidationResponse: []byte("credentials validation is not supported on this plugin :("),
	}, nil
}

func NewCorePlugin(rootPluginDirectory string) (*CorePlugin, error) {

	pluginConfig := config.GetConfig()

	description, err := description2.LoadPluginDescriptionFromDisk(path.Join(rootPluginDirectory, pluginConfig.Plugin.PluginDescriptionFilePath))
	if err != nil {
		return nil, err
	}

	loadedConnections, err := connections.LoadConnectionsFromDisk(path.Join(rootPluginDirectory, pluginConfig.Plugin.PluginDescriptionFilePath))
	if err != nil {
		return nil, err
	}

	log.Infof("Loaded %d connections from disk", len(loadedConnections))
	description.Connections = loadedConnections

	actionsFromDisk, err := actions.LoadActionsFromDisk(path.Join(rootPluginDirectory, pluginConfig.Plugin.ActionsFolderPath))
	if err != nil {
		return nil, err
	}

	supportedActions := map[string]ActionHandler{
		"python":        executeCorePythonAction,
		"bash":          executeCoreBashAction,
		"jq":            executeCoreJQAction,
		"jp":            executeCoreJPAction,
		"email":         executeCoreMailAction,
		"aws":           executeCoreAWSAction,
		"git":           executeCoreGITAction,
		"eksctl":        executeCoreAWSAction,
		"kubectl":       executeCoreKubernetesAction,
		"vault":         executeCoreVaultAction,
		"terraform":     executeCoreTerraFormAction,
		"kubectl_apply": executeCoreKubernetesApplyAction,
		"gcloud":        executeCoreGoogleCloudAction,
		"az":            executeCoreAzureAction,
		"fetch_file":    executeCoreFetchFileAction,
		"nodejs":        executeCoreNodejsAction,
	}

	return &CorePlugin{
		description:      *description,
		actions:          actionsFromDisk,
		supportedActions: supportedActions,
	}, nil
}
