package utils

import (
	log "github.com/sirupsen/logrus"
	"time"
)

type SparkJob struct {
	Name         string
	Namespace    string
	Image        string
	SparkVersion string
	Template     string
}

func (spark *SparkOperatorInstallation) SubmitJob(job SparkJob) error {
	yamlFile := createSparkJob(job)
	log.Infof("Submitting the job")
	_, err := KubectlApply(job.Namespace, yamlFile)

	return err
}

func (spark *SparkOperatorInstallation) WaitUntilSucceeded(timeout time.Duration, job SparkJob) error {
	driverPodName := job.Name + "-driver"
	return waitForPodStatusPhase(spark.Clients, driverPodName, job.Namespace, "Succeeded", timeout)
}
