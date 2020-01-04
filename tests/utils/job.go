package utils

import (
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	log "github.com/sirupsen/logrus"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

type SparkJob struct {
	Name           string
	Namespace      string
	Image          string
	SparkVersion   string
	Template       string
	ServiceAccount string
	Params         map[string]interface{}
	Drivers        int
	ExecutorsCount int
}

func (spark *SparkOperatorInstallation) SubmitJob(job *SparkJob) error {

	// Set default values
	if job.Namespace == "" {
		job.Namespace = spark.Namespace
	}
	if job.Image == "" {
		job.Image = SparkImage
	}
	if job.SparkVersion == "" {
		job.SparkVersion = SparkVersion
	}
	if job.ServiceAccount == "" {
		job.ServiceAccount = spark.InstanceName + DefaultServiceAccountSuffix
	}
	if job.ExecutorsCount == 0 {
		job.ExecutorsCount = 1
	}

	yamlFile := createSparkJob(*job)
	defer os.Remove(yamlFile)
	log.Infof("Submitting the job")
	err := KubectlApply(job.Namespace, yamlFile)

	return err
}

func (spark *SparkOperatorInstallation) DriverPod(job SparkJob) (v12.Pod, error) {
	pod, err := spark.K8sClients.CoreV1().Pods(job.Namespace).Get(DriverPodName(job.Name), v1.GetOptions{})
	return *pod, err
}

func (spark *SparkOperatorInstallation) ExecutorPods(job SparkJob) ([]v12.Pod, error) {
	pods, err := spark.K8sClients.CoreV1().Pods(job.Namespace).List(v1.ListOptions{
		LabelSelector: fmt.Sprintf("spark-role=executor,sparkoperator.k8s.io/app-name=%s", job.Name),
	})

	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}

func (spark *SparkOperatorInstallation) DriverLog(job SparkJob) (string, error) {
	driverPodName := DriverPodName(job.Name)
	return getPodLog(spark.K8sClients, job.Namespace, driverPodName, 0)
}

func (spark *SparkOperatorInstallation) DriverLogContains(job SparkJob, text string) (bool, error) {
	driverPodName := DriverPodName(job.Name)
	return podLogContains(spark.K8sClients, job.Namespace, driverPodName, text)
}

func (spark *SparkOperatorInstallation) SubmitAndWaitForExecutors(job *SparkJob) error {
	// Submit the job and wait for it to start
	err := spark.SubmitJob(job)
	if err != nil {
		return err
	}

	err = spark.WaitForJobState(*job, v1beta2.RunningState)
	if err != nil {
		return err
	}

	// Wait for correct number of executors to show up
	err = Retry(func() error {
		executors, err := spark.GetExecutorState(*job)
		if err != nil {
			return err
		} else if len(executors) != job.ExecutorsCount {
			return errors.New(fmt.Sprintf("The number of executors is %d, but %d is expected", len(executors), job.ExecutorsCount))
		}
		return nil
	})
	return err
}

func (spark *SparkOperatorInstallation) WaitForOutput(job SparkJob, text string) error {
	log.Infof("Waiting for the following text to appear in the driver log: %s", text)
	err := Retry(func() error {
		if contains, err := spark.DriverLogContains(job, text); !contains {
			if err != nil {
				return err
			} else {
				return errors.New("The driver log doesn't contain the text")
			}
		} else {
			log.Info("The text was found!")
			return nil
		}
	})

	if err != nil {
		log.Errorf("The text '%s' haven't appeared in the log in %s", text, DefaultRetryTimeout.String())
		logPodLogTail(spark.K8sClients, job.Namespace, DriverPodName(job.Name), 0) // 0 - print logs since pod's creation
	}
	return err
}

func (spark *SparkOperatorInstallation) WaitUntilSucceeded(job SparkJob) error {
	driverPodName := DriverPodName(job.Name)
	return waitForPodStatusPhase(spark.K8sClients, driverPodName, job.Namespace, "Succeeded")
}

func DriverPodName(jobName string) string {
	return jobName + "-driver"
}
