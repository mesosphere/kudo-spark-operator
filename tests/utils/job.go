package utils

import (
	"errors"
	log "github.com/sirupsen/logrus"
)

type SparkJob struct {
	Name           string
	Namespace      string
	Image          string
	SparkVersion   string
	Template       string
	ServiceAccount string
	Params         map[string]interface{}
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
		job.ServiceAccount = spark.InstanceName + "-spark-service-account"
	}

	yamlFile := createSparkJob(*job)
	log.Infof("Submitting the job")
	err := KubectlApply(job.Namespace, yamlFile)

	return err
}

func (spark *SparkOperatorInstallation) DriverLog(job SparkJob) (string, error) {
	driverPodName := driverPodName(job.Name)
	return getPodLog(spark.K8sClients, job.Namespace, driverPodName, 0)
}

func (spark *SparkOperatorInstallation) DriverLogContains(job SparkJob, text string) (bool, error) {
	driverPodName := driverPodName(job.Name)
	return podLogContains(spark.K8sClients, job.Namespace, driverPodName, text)
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
		log.Errorf("The text '%s' haven't appeared in the log in %s", text, defaultRetryTimeout.String())
		logPodLogTail(spark.K8sClients, job.Namespace, driverPodName(job.Name), 10)
	}
	return err
}

func (spark *SparkOperatorInstallation) WaitUntilSucceeded(job SparkJob) error {
	driverPodName := driverPodName(job.Name)
	return waitForPodStatusPhase(spark.K8sClients, driverPodName, job.Namespace, "Succeeded")
}

func driverPodName(jobName string) string {
	return jobName + "-driver"
}
