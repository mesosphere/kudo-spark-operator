package tests

import (
	"errors"
	"fmt"
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

		// Submit the job and wait for it to start
		err = spark.SubmitAndWaitForExecutors(&job)
		if err != nil {
			t.Fatal(err)
		}

		// Verify driver pod hostNetwork and dnsPolicy values
		driver, err := spark.DriverPod(job)
		if err != nil {
			t.Fatal(err)
		}
		err = verifyPodHostNetwork(driver, tc.driverHN)
		log.Infof("Verifying driver %s spec values", driver.Name)
		if err != nil {
			t.Fatal(err)
		}

		// Verify executor pods hostNetwork and dnsPolicy values
		executors, err := spark.ExecutorPods(job)
		if err != nil {
			t.Fatal(err)
		}
		for _, executor := range executors {
			log.Infof("Verifying executor %s spec values", executor.Name)
			err = verifyPodHostNetwork(&executor, tc.executorHN)
			if err != nil {
				t.Fatal(err)
			}
		}

		// Terminate the job while it's running
		spark.DeleteJob(job)
	}
}

func verifyPodHostNetwork(pod *v12.Pod, expectedHostNetwork bool) error {
	log.Infof("Pod spec.hostNetwork: %v", pod.Spec.HostNetwork)
	log.Infof("Pod spec.dnspolicy: %v", pod.Spec.DNSPolicy)

	// Check spec values
	if pod.Spec.HostNetwork != expectedHostNetwork {
		return errors.New(fmt.Sprintf("Unexpected hostNetwork value for pod %v: %s. Should be %v", pod.Spec.HostNetwork, pod.Name, expectedHostNetwork))
	} else if expectedHostNetwork && pod.Spec.DNSPolicy != v12.DNSClusterFirstWithHostNet {
		return errors.New(fmt.Sprintf("Expected pod pod DNS policy to be \"dnsClusterFirstWithHostNet\", but it's %s", pod.Spec.DNSPolicy))
	}

	// Check pod IP
	log.Infof("Pod status.podIP: %v", pod.Status.PodIP)
	log.Infof("Pod status.hostIP: %v", pod.Status.HostIP)
	if expectedHostNetwork && pod.Status.PodIP != pod.Status.HostIP {
		return errors.New(fmt.Sprintf("Pod %s IP doesn't match the host IP", pod.Name))
	} else if !expectedHostNetwork && pod.Status.PodIP == pod.Status.HostIP {
		return errors.New(fmt.Sprintf("Pod %s IP matches the host IP", pod.Name))
	}

	return nil
}
