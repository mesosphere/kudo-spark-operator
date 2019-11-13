package tests

import (
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	v12 "k8s.io/api/core/v1"
	"testing"
)

/*
	Test that `hostNetwork` in SparkApplication propagates to driver and executor pods
*/
func TestHostNetworkPropagation(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err)
	}

	var testCases = []struct {
		driverHN   bool
		executorHN bool
	}{
		{false, false},
		{true, false},
		{false, true},
		{true, true},
	}

	for i, tc := range testCases {
		log.Infof("Running test case:\n- driver host network:\t\t%v\n- executor host network:\t%v", tc.driverHN, tc.executorHN)
		jobName := fmt.Sprintf("host-network-test-job-%d", i)
		job := utils.SparkJob{
			Name:     jobName,
			Template: "spark-mock-task-runner-job-host-network.yaml",
			Params: map[string]interface{}{
				"args":                []string{"1", "600"},
				"driverHostNetwork":   tc.driverHN,
				"executorHostNetwork": tc.executorHN,
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

		// Verify driver pod hostNetwork and dnsPolicy values
		driver, err := spark.DriverPod(job)
		if err != nil {
			t.Fatal(err)
		}
		log.Infof("Driver spec.hostNetwork: %v", driver.Spec.HostNetwork)
		log.Infof("Driver spec.dnspolicy: %v", driver.Spec.DNSPolicy)
		if driver.Spec.HostNetwork != tc.driverHN {
			t.Fatal(fmt.Sprintf("Unexpected hostNetwork value for driver %v: %s. Should be %v", driver.Spec.HostNetwork, driver.Name, tc.driverHN))
		} else if tc.driverHN && driver.Spec.DNSPolicy != v12.DNSClusterFirstWithHostNet {
			t.Fatal(fmt.Sprintf("Expected driver pod DNS policy to be \"dnsClusterFirstWithHostNet\", but it's %s", driver.Spec.DNSPolicy))
		}

		// Verify executor pods hostNetwork and dnsPolicy values
		executors, err := spark.ExecutorPods(job)
		if err != nil {
			t.Fatal(err)
		}
		for _, executor := range executors {
			log.Infof("Executor %s spec.hostNetwork: %v", executor.Name, executor.Spec.HostNetwork)
			log.Infof("Executor %s spec.dnsPolicy: %v", executor.Name, executor.Spec.DNSPolicy)
			if executor.Spec.HostNetwork != tc.executorHN {
				t.Fatal(fmt.Sprintf("Unexpected hostNetwork value for driver %v: %s. Should be %v", executor.Spec.HostNetwork, executor.Name, tc.executorHN))
			} else if tc.executorHN && executor.Spec.DNSPolicy != v12.DNSClusterFirstWithHostNet {
				t.Fatal(fmt.Sprintf("Expected executor pod DNS policy to be \"dnsClusterFirstWithHostNet\", but it's %s", executor.Spec.DNSPolicy))
			}
		}

		// Terminate the job while it's running
		spark.DeleteJob(job)
	}
}
