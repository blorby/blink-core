package implementation

import (
	"errors"
	"github.com/blinkops/plugin-sdk/plugin"
	"github.com/blinkops/plugin-sdk/plugin/actions"
	"github.com/blinkops/plugin-sdk/plugin/config"
	"github.com/blinkops/plugin-sdk/plugin/connections"
	description2 "github.com/blinkops/plugin-sdk/plugin/description"
	log "github.com/sirupsen/logrus"
	"path"
)

type ActionHandler func(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error)

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

func (p *CorePlugin) ExecuteAction(ctx *plugin.ActionContext, request *plugin.ExecuteActionRequest) (*plugin.ExecuteActionResponse, error) {
	log.Debugf("Executing action: %v\n Context: %v", *request, ctx.GetAllContextEntries())

	actionHandler, ok := p.supportedActions[request.Name]
	if !ok {
		return nil, errors.New("action is not supported: " + request.Name)
	}

	resultBytes, err := actionHandler(ctx, request)
	if err != nil {
		log.Error("Failed executing action, err: ", err)
		return nil, err
	}

	return &plugin.ExecuteActionResponse{
		ErrorCode: 0,
		Result:    resultBytes,
	}, nil
}

func (p *CorePlugin) TestCredentials(_ map[string]connections.ConnectionInstance) (*plugin.CredentialsValidationResponse, error) {
	return nil, errors.New("credentials validation is not supported on this plugin :(")
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
		"python": executeCorePythonAction,
	}

	return &CorePlugin{
		description:      *description,
		actions:          actionsFromDisk,
		supportedActions: supportedActions,
	}, nil
}
