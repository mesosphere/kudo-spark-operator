package tests

import (
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
	v12 "k8s.io/api/core/v1"
	"testing"
)

const testLabelName = "non_existing_label"

func TestPodAffinityAndToleration(t *testing.T) {

	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	assert.NilError(t, err)

	jobName := "mock-task-runner"
	job := utils.SparkJob{
		Name:     jobName,
		Template: "spark-mock-task-runner-job.yaml",
		Params: map[string]interface{}{
			"args":                []string{"1", "60"},
			"DriverAffinity":      true,
			"DriverTolerations":   true,
			"ExecutorAffinity":    true,
			"ExecutorTolerations": true,
			"Label":               testLabelName,
		},
	}

	err = spark.SubmitJob(&job)
	assert.NilError(t, err)

	err = spark.WaitForJobState(job, v1beta2.RunningState)
	assert.NilError(t, err)

	var executors []v12.Pod

	log.Infof("Checking executor pods...")
	err = utils.Retry(func() error {
		pods, e := spark.ExecutorPods(job)
		if e != nil {
			return err
		} else if len(pods) == 0 {
			return errors.New("No executors found")
		} else {
			log.Infof("Found %d executor(s).", len(pods))
			executors = pods
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	t.Run("TestDriverPod", func(t *testing.T) {
		driver, err := spark.DriverPod(job)
		assert.NilError(t, err)
		verifyPodSpec(t, *driver)
	})

	t.Run("TestExecutorPod", func(t *testing.T) {
		verifyPodSpec(t, executors[0])
	})

}

func verifyPodSpec(t *testing.T, pod v12.Pod) {
	testAffinityRules(t, pod, testLabelName)
	testTolerationWithKeyPresent(t, pod, testLabelName)
}

func testAffinityRules(t *testing.T, pod v12.Pod, label string) {
	assert.Assert(t, pod.Spec.Affinity != nil, "Pod affinity is nil")
	var nodeAffinityRulePresent bool
	var podAffinityRulePresent bool
	for _, rule := range pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.
		NodeSelectorTerms {
		for _, term := range rule.MatchExpressions {
			if term.Key == label {
				nodeAffinityRulePresent = true
				break
			}
		}
	}
	for _, rule := range pod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
		for _, expr := range rule.LabelSelector.MatchExpressions {
			if expr.Key == label {
				podAffinityRulePresent = true
				break
			}
		}
	}
	assert.Assert(t, nodeAffinityRulePresent, fmt.Sprintf("Node affinity rule is missing, pod spec: \n%s", pod.Spec.String()))
	assert.Assert(t, podAffinityRulePresent, fmt.Sprintf("Pod affinity rule is missing, pod spec: \n%s", pod.Spec.String()))
}

func testTolerationWithKeyPresent(t *testing.T, pod v12.Pod, label string) {
	var tolerationPresent bool
	for _, toleration := range pod.Spec.Tolerations {
		if toleration.Key == label {
			tolerationPresent = true
			break
		}
	}
	assert.Assert(t, tolerationPresent, fmt.Sprintf("Toleration with key \"%s\" not found, pod spec: \n%s",
		label, pod.Spec.String()))
}
