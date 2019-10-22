package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
)

const KudoNamespace = "kudo-system"
const KudoService = "kudo-controller-manager-service"
const KudoCmd = "kubectl-kudo"

func InstallKudo() error {
	installed := isKudoInstalled()

	log.Infof("Kudo status: %v", installed)

	var args []string
	if installed {
		args = []string{"init", "--client-only"}
	} else {
		args = []string{"init"}
	}
	kudoInit := exec.Command(KudoCmd, args...)
	_, err := runAndLogCommandOutput(kudoInit)

	return err
}

func UninstallKudo() error {
	// To be implemented
	return nil
}

func installKudoPackage(namespace string, operatorDir string, instance string, params map[string]string) error {
	var args []string
	args = append(args, "--namespace", namespace)
	args = append(args, "install", operatorDir)
	args = append(args, "--instance", instance)
	for k, v := range params {
		args = append(args, "-p", fmt.Sprintf(`%s=%s`, k, v))
	}

	cmd := exec.Command(KudoCmd, args...)
	_, err := runAndLogCommandOutput(cmd)

	return err
}

func isKudoInstalled() bool {
	log.Info("Checking if KUDO is installed")
	clients, err := GetK8sClientSet()
	if err != nil {
		panic(err)
	}

	_, err = clients.CoreV1().Services(KudoNamespace).Get(KudoService, v1.GetOptions{})
	if err != nil {
		if se, ok := err.(*errors.StatusError); ok && se.ErrStatus.Reason == v1.StatusReasonNotFound {
			return false
		} else {
			panic(err)
		}
	}

	return true
}
