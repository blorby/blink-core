package execution

import (
	"fmt"
	"github.com/blinkops/blink-core/common"
	"github.com/blinkops/blink-sdk/plugin"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	StopExecutionSessionAction  = "stop_execution"
	randomPasswordLength        = 9
	acquireSessionMaxRetryCount = 10
	acquireSessionDelay         = 500 * time.Millisecond
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

func (p *PrivateExecutionEnvironment) GetGroupName() string {
	// need to make sure the group name is not all numbers to prevent the following scenario with numeric group names:
	// # groupadd 123
	// # useradd -m -d /home/sh_778800 -g 123 -s /bin/sh sh_778800
	//   useradd: group '123' does not exist
	return fmt.Sprintf("g_%s", p.NameRoot)
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
	output, err := common.ExecuteCommand(p, nil, nil, "/bin/mkdir", "-p", path)
	if err != nil {
		log.Debugf("failed to create directory at: %v with output: %v and error: %v", path, string(output), err)
	}
	return err
}

func (p *PrivateExecutionEnvironment) CreateTempDirectory() (string, error) {
	temporaryUUID := uuid.NewV4().String()
	tempDirectoryPath := path.Join(p.GetHomeDirectory(), temporaryUUID)
	err := p.CreateDirectory(tempDirectoryPath)
	return tempDirectoryPath, err
}

func (p *PrivateExecutionEnvironment) WriteToFile(name string, bytes []byte, perm os.FileMode) error {
	err := os.WriteFile(name, bytes, perm)
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

func (p *PrivateExecutionEnvironment) CreateCliUser(cli string) (*user.User, error) {
	usr, err := p.createUser(cli)
	if err != nil {
		return nil, err
	}

	err = p.cliUserSetup(cli, usr)
	if err != nil {
		p.CleanupCliUser(usr.Username)
	}
	return usr, err
}

func sudoersFile(username string) string {
	return path.Join("/etc/sudoers.d", username)
}

func (p *PrivateExecutionEnvironment) addSudoersEntry(cli string, username string) error {
	sudoerLine := fmt.Sprintf("%s  ALL=(%s) NOPASSWD:SETENV: %s/%s *\n", p.User.Username, username, common.ClisDir, cli)
	return os.WriteFile(sudoersFile(username), []byte(sudoerLine), 0440)
}

func (p *PrivateExecutionEnvironment) removeSudoersEntry(username string) error {
	return os.Remove(sudoersFile(username))
}

func (p *PrivateExecutionEnvironment) cliUserSetup(cli string, usr *user.User) error {
	if err := p.addSudoersEntry(cli, usr.Username); err != nil {
		return errors.Wrap(err, "failed adding sudoers entry")
	}
	return nil
}

func (p *PrivateExecutionEnvironment) CleanupCliUser(username string) {
	if username == "" {
		return
	}

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

	ctrl.executionSessionsMutex.Lock()
	defer ctrl.executionSessionsMutex.Unlock()
	delete(ctrl.executionSessions, session.GetSessionId())

	// Will delete the directory we created too.
	err := RemoveUser(session.GetUserName())
	err2 := RemoveGroup(session.GetGroupName())
	if err != nil && err2 != nil {
		return errors.Errorf("Destroy sessions failed, user error: %v, group error: %v", err, err2)
	}
	return nil
}

func (ctrl *Controller) EnsureNameRootSet(pee *PrivateExecutionEnvironment) {
	ctrl.executionSessionsMutex.Lock()
	defer ctrl.executionSessionsMutex.Unlock()
	if pee.NameRoot != "" {
		return
	}

	pee.NameRoot = ctrl.generateName(pee.SessionId)

	ctrl.executionSessions[pee.SessionId] = pee
}

func (ctrl *Controller) generateName(sessionId string) string {
	if len(sessionId) < 6 {
		return sessionId
	}

	root := sessionId[:6]
	if !ctrl.nameInUse(root) {
		return root
	}

	// Handle possible name collision
	counter := 1
	for {
		// If there's a collision we'll append running number until there's no collision, e.g. 010101_1, 010101_2, ...
		candidate := fmt.Sprintf("%s_%d", root, counter)
		if !ctrl.nameInUse(candidate) {
			return candidate
		}
		counter++
	}
}

func (ctrl *Controller) nameInUse(candidate string) bool {
	for _, ses := range ctrl.executionSessions {
		if ses.NameRoot == candidate {
			return true
		}
	}
	return false
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
	retry := 0
	executionExists := false
	for retry <= acquireSessionMaxRetryCount {
		session := GetExecutionController().GetExecutionSession(executionId)
		if session == nil {
			executionExists = false
			break
		}

		executionExists = true
		log.Infof("Found execution session with id: %s, Trying to acquire private execution session (%v/%v)", executionId, retry+1, acquireSessionMaxRetryCount)
		if session.User != nil && session.NameRoot != "" {
			log.Infof("Successfully acquired execution session with id: %s", executionId)
			return session, nil
		}
		retry++
		time.Sleep(acquireSessionDelay)
	}

	if retry >= acquireSessionMaxRetryCount && executionExists {
		return nil, errors.Errorf("Failed to acquire existing execution session with id: %s", executionId)
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

func (p *PrivateExecutionEnvironment) createExecutionSession() (shellUser *user.User, err error) {
	GetExecutionController().EnsureNameRootSet(p)

	defer func(err *error) {
		if err != nil && *err != nil {
			GetExecutionController().executionSessionsMutex.Lock()
			defer GetExecutionController().executionSessionsMutex.Unlock()
			delete(GetExecutionController().executionSessions, p.GetSessionId())
		}
	}(&err)

	if err = p.createGroup(); err != nil {
		return nil, errors.Wrap(err, "failed to create a group")
	}

	defer func(err *error) {
		if err != nil && *err != nil {
			if removeError := RemoveGroup(p.GetGroupName()); removeError != nil {
				log.Errorf("failed to delete group: %v on recovery from %v", p.GetGroupName(), err)
			}
		}
	}(&err)

	if shellUser, err = p.createUser("sh"); err != nil {
		return nil, errors.Wrap(err, "failed to create shell user")
	}

	return
}

func (p *PrivateExecutionEnvironment) createGroup() error {
	return AddNewGroup(p.GetGroupName())
}

func (p *PrivateExecutionEnvironment) createUser(prefix string) (*user.User, error) {
	if runtime.GOOS != "linux" {
		return user.Current()
	}

	userName := fmt.Sprintf("%s_%s", prefix, p.NameRoot)
	userDirectory := fmt.Sprintf("/home/%s", userName)
	err := os.Mkdir(userDirectory, 0770)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create user directory: ")
	}

	userToCreate := &User{
		Name:      userName,
		Group:     p.GetGroupName(),
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

	if err = ChOwnMod(userDirectory, userInformation.Uid, userInformation.Gid); err != nil {
		return nil, err
	}

	//log.Infof("setting umask")
	//unix.Umask(0007) // umask uses octal representation

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

	if err := os.Chmod(path, 0770); err != nil {
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
