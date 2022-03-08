package execution

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"runtime"
	"strings"
)

type User struct {
	Name      string
	Directory string
	Group     string
	Shell     string
}

func AddNewGroup(name string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	log.Infof("Adding new group named %s", name)
	groupCmd := exec.Command("groupadd", name)
	groupCmdOutput, err := groupCmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to create group with output [%s]: ", groupCmdOutput)
	}

	return nil
}

func RemoveGroup(name string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	log.Infof("Removing group named %s", name)
	groupCmd := exec.Command("groupdel", name)
	groupCmdOutput, err := groupCmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to remove group with output [%s]: ", groupCmdOutput)
	}

	return nil
}

func AddNewUser(u *User) error {
	argUser := []string{"-m", "-d", u.Directory, "-g", u.Group, "-G", "core", "-s", u.Shell, u.Name}

	log.Infof("Running: useradd %s", strings.Join(argUser, " "))
	userCmd := exec.Command("useradd", argUser...)
	createUserOutput, err := userCmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to create user with output [%s]: ", createUserOutput)
	}

	return nil
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
