package tests

import (
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	"testing"
)

func TestMountSparkConfigMap(t *testing.T) {
	const sparkConfFile = "resource/config/spark-defaults.conf"
	const cmName = "spark-conf"

	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err)
	}

	utils.CreateConfigMap(spark.K8sClients, spark.Namespace, cmName)
	defer utils.DropConfigMap(spark.K8sClients, spark.Namespace, cmName)
	utils.AddFileToConfigMap(spark.K8sClients, cmName, spark.Namespace, "spark-defaults.com", sparkConfFile)

	job := utils.SparkJob{
		Name:     "mount-spark-configmap-test",
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

}

func TestMountHadoopConfigMap(t *testing.T) {

}
