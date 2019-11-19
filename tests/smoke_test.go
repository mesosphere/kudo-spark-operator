package tests

import (
	"errors"
	"fmt"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
	"testing"
)

func TestShuffleAppDriverOutput(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err)
	}

	const expectedExecutorCount = 2
	const expectedGroupCount = 12000

	jobName := "shuffle-app"
	job := utils.SparkJob{
		Name:           jobName,
		Template:       "spark-shuffle-job.yaml",
		ExecutorsCount: expectedExecutorCount,
		Params: map[string]interface{}{
			"args": []string{"4", strconv.Itoa(expectedGroupCount), "100", "4", "1500"},
		},
	}

	// Submit the job and wait for it to start
	err = spark.SubmitAndWaitForExecutors(&job)
	if err != nil {
		t.Fatal(err)
	}

	err = spark.WaitForOutput(job, fmt.Sprintf("Groups count: %d", expectedGroupCount))
	if err != nil {
		t.Error(err)
	}
}

func TestRunningAppDeletion(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err)
	}

	jobName := "mock-task-runner"
	job := utils.SparkJob{
		Name:     jobName,
		Template: "spark-mock-task-runner-job.yaml",
		Params: map[string]interface{}{
			"args": []string{"1", "600"},
		},
	}

	// Submit the job and wait for it to start
	err = spark.SubmitAndWaitForExecutors(&job)
	if err != nil {
		t.Fatal(err)
	}

	// Terminate the job while it's running
	spark.DeleteJob(job)

	// Make sure no executors or drivers left
	log.Info("Verifying that all executors and drivers are terminated")
	err = utils.Retry(func() error {
		// Get all pods named mock-task-runner*
		var jobPods []v12.Pod
		pods, _ := spark.K8sClients.CoreV1().Pods(spark.Namespace).List(v1.ListOptions{})
		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, jobName) {
				jobPods = append(jobPods, pod)
			}
		}

		if len(jobPods) != 0 {
			for _, pod := range jobPods {
				log.Infof("found %s - %s", pod.Name, pod.Status.Phase)
			}

			return errors.New("there are still pods left after the job termination")
		}
		return nil
	})
	if err != nil {
		t.Error(err.Error())
	}
}
