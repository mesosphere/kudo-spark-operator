package utils

import (
	"errors"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

type SparkJob struct {
	Name         string
	Namespace    string
	Image        string
	SparkVersion string
	Template     string
}

func SubmitJob(spark *SparkOperatorInstallation, job SparkJob) {
	yamlFile := createSparkJob(job)
	KubectlApply(job.Namespace, yamlFile)
	driverPodName := job.Name + "-driver"
	waitForPodStatus(spark.Clients, driverPodName, job.Namespace, "Succeeded")
}

func waitForPodStatus(clientSet *kubernetes.Clientset, podName string, namespace string, status string) {
	retry(10*60, 500*time.Millisecond, func() error {
		pod, err := clientSet.CoreV1().Pods(namespace).Get(podName, v1.GetOptions{})
		if err == nil && string(pod.Status.Phase) != status {
			err = errors.New("Expected pod status to be " + status + ", but it's " + string(pod.Status.Phase))
		}
		return err
	})
}

func retry(timeout int64, interval time.Duration, fn func() error) error {
	timeoutUnixTime := time.Now().Unix() + timeout
	var err error

	for err = fn(); err != nil || timeoutUnixTime > time.Now().Unix(); {
		log.Warn(err.Error())
		time.Sleep(interval)
		log.Warn("Retrying...")
		err = fn()
	}
	return err
}
