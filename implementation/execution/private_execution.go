package execution

import (
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strconv"
	"sync"
)

const (
	StopExecutionSessionAction = "stop_execution"

	randomPasswordLength = 9
)

var (
	createOnce sync.Once
	controller *ExecutionController
)

type PrivateExecutionEnvironment struct {
	SessionId string
	User      user.User
}

func (p *PrivateExecutionEnvironment) GetUserName() string {
	return p.User.Username
}

func (p *PrivateExecutionEnvironment) GetSessionId() string {
	return p.SessionId
}

func (p *PrivateExecutionEnvironment) GetTempDirectory() string {
	return p.User.HomeDir
}

func (p *PrivateExecutionEnvironment) GetExecutorUid() uint32 {
	result, _ := strconv.Atoi(p.User.Uid)
	return uint32(result)
}

func (p *PrivateExecutionEnvironment) GetExecutorGid() uint32 {
	result, _ := strconv.Atoi(p.User.Gid)
	return uint32(result)
}

func (p *PrivateExecutionEnvironment) CreateDirectory(path string) error {
	err := os.MkdirAll(path, 0700)
	if err != nil {
		return errors.Wrap(err, "Failed to create temporary directory")
	}

	err = os.Chown(path, int(p.GetExecutorUid()), int(p.GetExecutorGid()))
	if err != nil {
		return err
	}

	return nil
}

func (p *PrivateExecutionEnvironment) CreateTempDirectory(name string) (string, error) {
	tempDirectoryPath := path.Join(p.GetTempDirectory(), name)
	err := p.CreateDirectory(tempDirectoryPath)
	return tempDirectoryPath, err
}

func (p *PrivateExecutionEnvironment) WriteToFile(name string, bytes []byte) error {
	err := os.WriteFile(name, bytes, 0700)
	if err != nil {
		return errors.Wrap(err, "Failed writing to file: ")
	}

	err = os.Chown(name, int(p.GetExecutorUid()), int(p.GetExecutorGid()))
	if err != nil {
		return errors.Wrap(err, "Failed to Chown file: ")
	}

	return nil
}

func (p *PrivateExecutionEnvironment) WriteToTempFileWithRootPath(root string, bytes []byte, prefix string) (string, error) {

	file, err := ioutil.TempFile(root, prefix)
	if err != nil {
		return "", err
	}

	defer func() {
		// Close the file
		if err := file.Close(); err != nil {
			log.Error("failed to close file", err)
		}
	}()

	_, err = file.Write(bytes)
	if err != nil {
		return "", err
	}

	err = os.Chown(file.Name(), int(p.GetExecutorUid()), int(p.GetExecutorGid()))
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}

func (p *PrivateExecutionEnvironment) WriteToTempFile(bytes []byte, prefix string) (string, error) {
	return p.WriteToTempFileWithRootPath(p.GetTempDirectory(), bytes, prefix)
}

type ExecutionController struct {
	executionSessionsMutex sync.RWMutex
	executionSessions      map[string]*PrivateExecutionEnvironment
}

func (ctrl *ExecutionController) GetExecutionSession(executionId string) *PrivateExecutionEnvironment {
	ctrl.executionSessionsMutex.RLock()
	defer ctrl.executionSessionsMutex.RUnlock()

	session, ok := ctrl.executionSessions[executionId]
	if !ok {
		return nil
	}

	return session
}

func (ctrl *ExecutionController) SaveExecutionSession(session *PrivateExecutionEnvironment) {
	ctrl.executionSessionsMutex.Lock()
	defer ctrl.executionSessionsMutex.Unlock()

	ctrl.executionSessions[session.GetSessionId()] = session
}

func (ctrl *ExecutionController) DestroyExecutionSession(executionId string) error {
	session := ctrl.GetExecutionSession(executionId)
	if session == nil {
		return errors.Errorf("No execution session found for %s", executionId)
	}

	// Will delete the directory we created too.
	return RemoveUser(session.GetUserName())
}

func GetExecutionController() *ExecutionController {
	createOnce.Do(func() {
		controller = &ExecutionController{
			executionSessions: map[string]*PrivateExecutionEnvironment{},
		}
	})

	return controller
}

func AcquirePrivateExecutionSession(executionId string) (*PrivateExecutionEnvironment, error) {

	if session := GetExecutionController().GetExecutionSession(executionId); session != nil {
		log.Infof("Execution session already exists %s", executionId)
		return session, nil
	}

	log.Infof("Creating execution session for %s", executionId)

	userDirectory := fmt.Sprintf("/executions/%s", executionId)
	err := os.Mkdir(userDirectory, 0777)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create user directory: ")
	}

	userToCreate := &User{
		Name:      Sanitize(executionId),
		Group:     "core",
		Shell:     "/bin/sh",
		Directory: userDirectory,
	}

	_, err = AddNewUser(userToCreate)
	if err != nil {
		return nil, err
	}

	userInformation, err := user.Lookup(userToCreate.Name)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to lookup user for private execution: ")
	}

	uidAsInt, _ := strconv.Atoi(userInformation.Uid)
	gidAsInt, _ := strconv.Atoi(userInformation.Gid)

	err = os.Chown(userDirectory, uidAsInt, gidAsInt)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to change change ownership of  directory : ")
	}

	err = os.Chmod(userDirectory, 0777)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to change user directory permissions: ")
	}

	session := &PrivateExecutionEnvironment{
		SessionId: executionId,
		User:      *userInformation,
	}

	log.Infof("Created user for private execution %v", *userInformation)

	GetExecutionController().SaveExecutionSession(session)
	return session, nil
}

func StopPrivateExecution(request *plugin.ExecuteActionRequest) ([]byte, error) {
	executionId := request.Parameters["execution_id"]
	if executionId == "" {
		return nil, errors.New("Failed to stop core private execution, execution_id parameter is missing")
	}

	err := GetExecutionController().DestroyExecutionSession(executionId)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to destroy execution session with error: ")
	}

	return []byte("Success"), nil // Does not really matter what output we return here
}
