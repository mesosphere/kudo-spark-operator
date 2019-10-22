package utils

import (
	log "github.com/sirupsen/logrus"
	"time"
)

const defaultJobCompletionTimeout = 10 * time.Minute

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

func (spark *SparkOperatorInstallation) WaitUntilSucceeded(job SparkJob) error {
	return spark.WaitUntilSucceededWithTimeout(defaultJobCompletionTimeout, job)
}

func (spark *SparkOperatorInstallation) WaitUntilSucceededWithTimeout(timeout time.Duration, job SparkJob) error {
	driverPodName := job.Name + "-driver"
	return waitForPodStatusPhase(spark.Clients, driverPodName, job.Namespace, "Succeeded", timeout)
}
