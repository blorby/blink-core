package execution

import (
	"fmt"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"sync"
)

const (
	StopExecutionSessionAction = "stop_execution"
	randomPasswordLength       = 9
)

var (
	createOnce sync.Once
	controller *Controller
)

type PrivateExecutionEnvironment struct {
	SessionId string
	User      *user.User
	NameRoot  string
}

func (p *PrivateExecutionEnvironment) GetUserName() string {
	return p.User.Username
}

func (p *PrivateExecutionEnvironment) GetSessionId() string {
	return p.SessionId
}

func (p *PrivateExecutionEnvironment) GetHomeDirectory() string {
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
	_, err := common.ExecuteCommand(p, nil, nil, "/bin/mkdir", "-p", path)
	return err
}

func (p *PrivateExecutionEnvironment) CreateTempDirectory() (string, error) {
	temporaryUUID := uuid.NewV4().String()
	tempDirectoryPath := path.Join(p.GetHomeDirectory(), temporaryUUID)
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

func (p *PrivateExecutionEnvironment) WriteFile(bytes []byte, fileName string) error {

	err := ioutil.WriteFile(fileName, bytes, 0700)
	if err != nil {
		return err
	}

	err = os.Chown(fileName, int(p.GetExecutorUid()), int(p.GetExecutorGid()))
	if err != nil {
		return err
	}

	return nil
}

func (p *PrivateExecutionEnvironment) WriteToTempFile(bytes []byte, prefix string) (string, error) {
	temporaryUUID := uuid.NewV4().String()
	fileName := fmt.Sprintf("%s%s", prefix, temporaryUUID)
	fullFileName := path.Join(p.GetHomeDirectory(), fileName)
	err := p.WriteFile(bytes, fullFileName)
	return fullFileName, err
}

func (p *PrivateExecutionEnvironment) CreateCliUser(cli string, environment []string) (*user.User, error) {
	usr, err := p.createUser(cli)
	if err != nil {
		return nil, err
	}

	err = p.cliUserSetup(cli, usr, environment)
	if err != nil {
		p.CleanupCliUser(usr.Username)
	}
	return usr, err
}

func (p *PrivateExecutionEnvironment) WriteProfileFile(usr *user.User, environment []string) error {
	if len(environment) < 1 {
		return nil
	}
	profileFile := path.Join(usr.HomeDir, ".profile")

	log.Infof("Writing environment variables into: %s", profileFile)
	file, err := os.OpenFile(profileFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
		_ = ChOwnMod(profileFile, usr.Uid, usr.Gid)
	}(file)

	for _, line := range environment {
		if _, err = file.WriteString(fmt.Sprintf("export %s\n", line)); err != nil {
			return err
		}
	}
	return err
}

func sudoersFile(username string) string {
	return path.Join("/etc/sudoers.d", username)
}

func (p *PrivateExecutionEnvironment) addSudoersEntry(cli string, username string) error {
	sudoerLine := fmt.Sprintf("%s  ALL=(%s) NOPASSWD: /usr/bin/sh -c /opt/blink/%s *\n", p.User.Username, username, cli)
	return os.WriteFile(sudoersFile(username), []byte(sudoerLine), 0440)
}

func (p *PrivateExecutionEnvironment) removeSudoersEntry(username string) error {
	return os.Remove(sudoersFile(username))
}

func (p *PrivateExecutionEnvironment) cliUserSetup(cli string, usr *user.User, environment []string) error {
	if err := p.WriteProfileFile(usr, environment); err != nil {
		return errors.Wrap(err, "failed creating .profile file")
	}
	if err := p.addSudoersEntry(cli, usr.Username); err != nil {
		return errors.Wrap(err, "failed adding sudoers entry")
	}
	return nil
}

func (p *PrivateExecutionEnvironment) CleanupCliUser(username string) {
	if err := RemoveUser(username); err != nil {
		log.Errorf("failed to remove a user with error: %v", err)
	}

	if err := p.removeSudoersEntry(username); err != nil {
		log.Errorf("failed to remove sudoers entry: %v", err)
	}
}

type Controller struct {
	executionSessionsMutex sync.RWMutex
	executionSessions      map[string]*PrivateExecutionEnvironment
}

func (ctrl *Controller) GetExecutionSession(executionId string) *PrivateExecutionEnvironment {
	ctrl.executionSessionsMutex.RLock()
	defer ctrl.executionSessionsMutex.RUnlock()

	session, ok := ctrl.executionSessions[executionId]
	if !ok {
		return nil
	}

	return session
}

func (ctrl *Controller) SaveExecutionSession(session *PrivateExecutionEnvironment) {
	ctrl.executionSessionsMutex.Lock()
	defer ctrl.executionSessionsMutex.Unlock()

	ctrl.executionSessions[session.GetSessionId()] = session
}

func (ctrl *Controller) DestroyExecutionSession(executionId string) error {
	session := ctrl.GetExecutionSession(executionId)
	if session == nil {
		return errors.Errorf("No execution session found for %s", executionId)
	}

	// Will delete the directory we created too.
	err := RemoveUser(session.GetUserName())

	ctrl.executionSessionsMutex.Lock()
	defer ctrl.executionSessionsMutex.Unlock()
	delete(ctrl.executionSessions, session.GetSessionId())
	return err
}

func (ctrl *Controller) GetRandomName(sessionID string) string {
	ctrl.executionSessionsMutex.Lock()
	defer ctrl.executionSessionsMutex.Unlock()
	candidate := sessionID[:6]

	// Handle possible name collision
OuterLoop:
	for {
		counter := 1
		randomName := candidate
		for _, ses := range ctrl.executionSessions {
			if ses.NameRoot == randomName {
				randomName = fmt.Sprintf("%s_%d", candidate, counter)
				counter++
				continue OuterLoop
			}
		}
		return randomName
	}
}

func GetExecutionController() *Controller {
	createOnce.Do(func() {
		controller = &Controller{
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

	session := &PrivateExecutionEnvironment{
		SessionId: executionId,
	}

	userInformation, err := session.createExecutionSession()
	if err != nil {
		return nil, err
	}

	session.User = userInformation
	log.Infof("Created user for private execution %v", userInformation)

	GetExecutionController().SaveExecutionSession(session)
	return session, nil
}

func (p *PrivateExecutionEnvironment) createExecutionSession() (*user.User, error) {
	return p.createUser("sh")
}

func (p *PrivateExecutionEnvironment) createUser(prefix string) (*user.User, error) {
	if runtime.GOOS != "linux" {
		currentUser, err := user.Current()
		return currentUser, err
	}
	if p.NameRoot == "" {
		p.NameRoot = GetExecutionController().GetRandomName(p.SessionId)
	}
	userName := fmt.Sprintf("%s_%s", prefix, p.NameRoot)
	userDirectory := fmt.Sprintf("/home/%s", userName)
	err := os.Mkdir(userDirectory, 0700)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create user directory: ")
	}

	userToCreate := &User{
		Name:      userName,
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

	err = ChOwnMod(userDirectory, userInformation.Uid, userInformation.Gid)
	if err != nil {
		return nil, err
	}

	log.Infof("setting umask")
	unix.Umask(077) // umask uses octal representation

	return userInformation, nil
}

func (p *PrivateExecutionEnvironment) CreateCliUserPee(cliUser *user.User) *PrivateExecutionEnvironment {
	return &PrivateExecutionEnvironment{
		SessionId: p.SessionId,
		User:      cliUser,
		NameRoot:  p.NameRoot,
	}
}

func ChOwnMod(path string, uid string, gid string) error {
	uidAsInt, _ := strconv.Atoi(uid)
	gidAsInt, _ := strconv.Atoi(gid)

	if err := os.Chown(path, uidAsInt, gidAsInt); err != nil {
		return errors.Wrap(err, "Failed to change change ownership of  directory : ")
	}

	if err := os.Chmod(path, 0700); err != nil {
		return errors.Wrap(err, "Failed to change user directory permissions: ")
	}
	return nil
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
