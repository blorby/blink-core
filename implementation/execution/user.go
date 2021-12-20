package execution

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"runtime"
	"strings"
)

func CreateRandom(n int) ([]byte, error) {
	b := make([]byte, n)

	_, err := rand.Read(b)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate random: ")
	}

	return b, nil
}

type User struct {
	Name      string
	Directory string
	Group     string
	Shell     string
}

func AddNewGroup(name string) error {
	log.Infof("Adding new group named %s", name)
	groupCmd := exec.Command("groupadd", name)
	groupCmdOutput, err := groupCmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to create group with output [%s]: ", groupCmdOutput)
	}

	return nil
}

func RemoveGroup(name string) error {
	log.Infof("Removing group named %s", name)
	groupCmd := exec.Command("groupdel", name)
	groupCmdOutput, err := groupCmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to remove group with output [%s]: ", groupCmdOutput)
	}

	return nil
}

func AddNewUser(u *User) (string, error) {

	passwordBase, err := CreateRandom(randomPasswordLength)
	if err != nil {
		return "", err
	}

	password := base64.StdEncoding.EncodeToString(passwordBase)

	argUser := []string{"-m", "-d", u.Directory, "-g", u.Group, "-s", u.Shell, u.Name}
	argPass := []string{"-c", fmt.Sprintf("echo %s:%s | chpasswd", u.Name, password)}

	log.Infof("Running: useradd %s", strings.Join(argUser, " "))

	userCmd := exec.Command("useradd", argUser...)
	passCmd := exec.Command("/bin/sh", argPass...)

	createUserOutput, err := userCmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to create user with output [%s]: ", createUserOutput)
	}

	createPasswordOutput, err := passCmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to set user password with output [%s]: ", createPasswordOutput)
	}

	return password, nil
}

func RemoveUser(username string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	argUser := []string{"-r", username}
	userCmd := exec.Command("userdel", argUser...)

	output, err := userCmd.CombinedOutput()
	if err != nil {
		log.Errorf("Failed to delete user with error %v output: %s", err, string((output)))
		return errors.Wrap(err, "Failed to delete user with error: ")
	}

	return nil
}
