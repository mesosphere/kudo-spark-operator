package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

/* client-go util methods */

func GetK8sClientSet() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfig)
	if err != nil {
		log.Errorf("Can't build config [kubeconfig = %s]: %s", KubeConfig, err)
		return nil, err
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

	err := clientSet.CoreV1().Namespaces().Delete(name, &options)
	if err != nil {
		log.Errorf("Can't delete namespace '%s':%s", name, err)
		return err
	}

	return Retry(func() error {
		_, err := clientSet.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
		if err == nil {
			return errors.New(fmt.Sprintf("Namespace '%s' still exists", name))
		} else if statusErr, ok := err.(*apiErrors.StatusError); !ok || statusErr.Status().Reason != metav1.StatusReasonNotFound {
			return err
		} else {
			log.Info(fmt.Sprintf("Namespace '%s' successfully deleted", name))
			return nil
		}
	})
}

func CreateServiceAccount(clientSet *kubernetes.Clientset, name string, namespace string) error {
	log.Infof("Creating a service account %s/%s", namespace, name)
	sa := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := clientSet.CoreV1().ServiceAccounts(namespace).Create(&sa)
	return err
}

func CreateSecret(clientSet *kubernetes.Clientset, name string, namespace string, secretData map[string]string) error {
	log.Infof("Creating a secret %s/%s with Secret Data: %q", namespace, name, secretData)
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		StringData: secretData,
	}

	_, err := clientSet.CoreV1().Secrets(namespace).Create(&secret)
	return err
}

func getPodLog(clientSet *kubernetes.Clientset, namespace string, pod string, tailLines int64) (string, error) {
	opts := v1.PodLogOptions{}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}
	req := clientSet.CoreV1().Pods(namespace).GetLogs(pod, &opts)

	logSteam, err := req.Stream()
	if err != nil {
		return "", err
	}
	defer logSteam.Close()

	logBuffer := new(bytes.Buffer)
	_, err = io.Copy(logBuffer, logSteam)
	if err != nil {
		return "", err
	}

	return logBuffer.String(), nil
}

func podLogContains(clientSet *kubernetes.Clientset, namespace string, pod string, text string) (bool, error) {
	opts := v1.PodLogOptions{}
	req := clientSet.CoreV1().Pods(namespace).GetLogs(pod, &opts)

	logSteam, err := req.Stream()
	if err != nil {
		return false, err
	}
	defer logSteam.Close()

	scanner := bufio.NewScanner(logSteam)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), text) {
			return true, nil
		}
	}

	if err = scanner.Err(); err != nil {
		return false, err
	} else {
		return false, nil
	}
}

func logPodLogTail(clientSet *kubernetes.Clientset, namespace string, pod string, lines int64) error {
	logTail, err := getPodLog(clientSet, namespace, pod, lines)
	if err == nil {
		log.Infof("Last %d lines of %s log:\n%s", lines, pod, logTail)
	}
	return err
}

func waitForPodStatusPhase(clientSet *kubernetes.Clientset, podName string, namespace string, status string) error {
	log.Infof("Waiting for pod %s to enter phase %s", podName, status)

	return Retry(func() error {
		pod, err := clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err == nil && string(pod.Status.Phase) != status {
			err = errors.New("Expected pod status to be " + status + ", but it's " + string(pod.Status.Phase))
		} else if string(pod.Status.Phase) == status {
			log.Infof("\"%s\" completed successfully.", podName)
		}
		return err
	})
}

/* kubectl helpers */

func Kubectl(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)
	return runAndLogCommandOutput(cmd)

}

func DeleteResource(namespace string, resource string, name string) error {
	_, err := Kubectl("delete", resource, name, "--namespace", namespace, "--ignore-not-found=true")
	return err
}

func KubectlApply(namespace string, filename string) error {
	log.Infof("Applying file %s with kubectl", filename)
	return kubectlRunFile("apply", namespace, filename)
}

func KubectlDelete(namespace string, filename string) error {
	log.Infof("Deleting objects from file %s with kubectl", filename)
	return kubectlRunFile("delete", namespace, filename)
}

func kubectlRunFile(method string, namespace string, filename string) error {
	kubectl := exec.Command("kubectl", method, "--namespace", namespace, "-f", filename)
	_, err := runAndLogCommandOutput(kubectl)
	return err
}
