package utils

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os/exec"
)

// client-go util methods

func GetK8sClientSet() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfig)
	if err != nil {
		panic(err.Error())
	}

	return kubernetes.NewForConfig(config)
}

// kubectl helpers

func KubectlApply(namespace string, filename string) ([]byte, error) {
	log.Infof("Applying file %s with kubectl", filename)
	return kubectlRunFile("apply", namespace, filename)
}

func KubectlDelete(namespace string, filename string) ([]byte, error) {
	log.Infof("Deleting objects from file %s with kubectl", filename)
	return kubectlRunFile("delete", namespace, filename)
}

func kubectlRunFile(method string, namespace string, filename string) ([]byte, error) {
	kubectl := exec.Command("kubectl", method, "--namespace", namespace, "-f", filename)
	out, err := kubectl.Output()
	log.Info("kubectl output:")
	log.Info(string(out))
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.Stderr, err
		}
	}
	return out, err
}
