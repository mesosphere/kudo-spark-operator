package tests

import (
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
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
	err := spark.InstallSparkOperator(true)
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err)
	}

	const expectedExecutorCount = 2
	const expectedGroupCount = 12000

	jobName := "shuffle-app"
	job := utils.SparkJob{
		Name:     jobName,
		Template: "spark-shuffle-job.yaml",
		Params: map[string]interface{}{
			"executor_count": expectedExecutorCount,
			"args":           []string{"4", strconv.Itoa(expectedGroupCount), "100", "4", "1500"},
		},
	}

	err = spark.SubmitJob(&job)
	if err != nil {
		t.Fatal(err)
	}
	err = spark.WaitForJobState(job, v1beta2.RunningState)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for correct number of executors to show up
	err = utils.Retry(func() error {
		executors, err := spark.GetExecutorState(job)
		if err != nil {
			return err
		} else if len(executors) != expectedExecutorCount {
			return errors.New(fmt.Sprintf("The number of executors is %d, but %d is expected", len(executors), expectedExecutorCount))
		}
		return nil
	})
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
	err := spark.InstallSparkOperator(true)
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
	expectedExecutorCount := 1

	// Submit the job and wait for it to start
	err = spark.SubmitJob(&job)
	if err != nil {
		t.Fatal(err)
	}
	err = spark.WaitForJobState(job, v1beta2.RunningState)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for correct number of executors to show up

	err = utils.Retry(func() error {
		executors, err := spark.GetExecutorState(job)
		if err != nil {
			return err
		} else if len(executors) != expectedExecutorCount {
			return errors.New(fmt.Sprintf("The number of executors is %d, but %d is expected", len(executors), expectedExecutorCount))
		}
		return nil
	})
	if err != nil {
		t.Error(err)
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
