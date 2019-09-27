package utils

import (
	log "github.com/sirupsen/logrus"
)

type SparkJob struct {
	Name         string
	Namespace    string
	Image        string
	SparkVersion string
	Template     string
}

func (spark *SparkOperatorInstallation) SubmitJob(job SparkJob) error {
	log.Info("Making sure spark operator is running")
	err := spark.WaitUntilRunning()
	if err != nil {
		return err
	}

	yamlFile := createSparkJob(job)
	log.Infof("Submitting the job")
	_, err = KubectlApply(job.Namespace, yamlFile)

	return err
}

func (spark *SparkOperatorInstallation) WaitUntilSucceeded(job SparkJob) error {
	driverPodName := job.Name + "-driver"
	return waitForPodStatus(spark.Clients, driverPodName, job.Namespace, "Succeeded")
}
