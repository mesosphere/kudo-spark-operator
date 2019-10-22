package utils

import (
	"errors"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os/exec"
	"strings"
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

func CreateNamespace(clientSet *kubernetes.Clientset, name string) (*v1.Namespace, error) {
	log.Infof("Creating namespace %s", name)
	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	return clientSet.CoreV1().Namespaces().Create(&namespace)
}

func DropNamespace(clientSet *kubernetes.Clientset, name string) error {
	log.Infof("Deleting namespace %s", name)
	gracePeriod := int64(0)
	propagationPolicy := metav1.DeletePropagationForeground
	options := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
		PropagationPolicy:  &propagationPolicy,
	}

	return clientSet.CoreV1().Namespaces().Delete(name, &options)
}

func waitForPodStatusPhase(clientSet *kubernetes.Clientset, podName string, namespace string, status string, timeout time.Duration) error {
	log.Infof("Waiting for pod %s to enter phase %s", podName, status)
	return retry(timeout, 1*time.Second, func() error {
		pod, err := clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err == nil && string(pod.Status.Phase) != status {
			err = errors.New("Expected pod status to be " + status + ", but it's " + string(pod.Status.Phase))
		}
		return err
	})
}

/* kubectl helpers */

func Kubectl(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	log.Infof(">%s %v\n%s", cmd.Path, cmd.Args, out)
	return strings.TrimSpace(string(out)), err

}

func DeleteResource(namespace string, resource string, name string) error {
	_, err := Kubectl("delete", resource, name, "--namespace", namespace)
	return err
}

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
	out, err := kubectl.CombinedOutput()
	log.Infof(">%s %v\n%s", kubectl.Path, kubectl.Args, out)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.Stderr, err
		}
	}
	return out, err
}
