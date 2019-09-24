package utils

import (
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"
)

func TestClientGo(t *testing.T) {
	clientSet, err := GetK8sClientSet()
	if err != nil {
		t.Error(err.Error())
	}

	pods, err := clientSet.CoreV1().Pods("").List(v1.ListOptions{})
	if err != nil {
		t.Error(err.Error())
	}

	log.Infof("There are %d pods in the cluster\n", len(pods.Items))
}

func TestTemplating(t *testing.T) {
	tmpFilePath := createSparkOperatorNamespace("testtesttest")
	defer os.Remove(tmpFilePath)

	log.Infof("Created a temp file at %s", tmpFilePath)
}

func TestHelmInstallation(t *testing.T) {
	err := installSparkOperatorWithHelm(DefaultNamespace)
	if err != nil {
		t.Error(err.Error())
	}

	err = uninstallSparkOperatorWithHelm(DefaultNamespace)
	if err != nil {
		t.Error(err.Error())
	}
}
