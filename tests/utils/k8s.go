package utils

import (
	"errors"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os/exec"
	"time"
)

/* client-go util methods */

func GetK8sClientSet() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfig)
	if err != nil {
		panic(err.Error())
	}

	return kubernetes.NewForConfig(config)
}

func waitForPodStatus(clientSet *kubernetes.Clientset, podName string, namespace string, status string) error {
	log.Infof("Waiting for pod %s to enter phase %s", podName, status)
	return retry(10*time.Minute, 1*time.Second, func() error {
		pod, err := clientSet.CoreV1().Pods(namespace).Get(podName, v1.GetOptions{})
		if err == nil && string(pod.Status.Phase) != status {
			err = errors.New("Expected pod status to be " + status + ", but it's " + string(pod.Status.Phase))
		}
		return err
	})
}

/* kubectl helpers */

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
