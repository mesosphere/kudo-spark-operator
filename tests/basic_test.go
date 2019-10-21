package tests

import (
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	utils.InstallKudo()
	defer utils.UninstallKudo()

	m.Run()
}

func TestSparkOperatorInstallation(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err.Error())
	}

	k8sNamespace, err := spark.K8sClients.CoreV1().Namespaces().Get(spark.Namespace, v1.GetOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	log.Infof("Spark operator is installed in namespace %s", k8sNamespace.Name)
}

func TestSparkOperatorInstallationWithCustomNamespace(t *testing.T) {
	customNamespace := "custom-test-namespace"
	spark := utils.SparkOperatorInstallation{
		Namespace: customNamespace,
	}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err.Error())
	}

	k8sNamespace, err := spark.K8sClients.CoreV1().Namespaces().Get(spark.Namespace, v1.GetOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	if k8sNamespace.Name != customNamespace {
		t.Errorf("Actual namespace is %s, while %s was expected", k8sNamespace.Name, customNamespace)
	}
}

func TestJobSubmission(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err)
	}

	job := utils.SparkJob{
		Name:     "linear-regression",
		Template: "spark-linear-regression-job.yaml",
	}

	err = spark.SubmitJob(job)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = spark.WaitUntilSucceeded(job)
	if err != nil {
		t.Error(err.Error())
	}
}

/*
	TODO
   [x] start MockTaskRunner job
   [ ] verify that all the executors are up and running
   [ ] kill/delete the Driver using kubectl delete sparkapplicatrion or similar command
   [ ] verify no orphaned resources left
*/
func TestMockTaskRunner(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Error(err)
	}

	jobName := "mock-task-runner"

	job := utils.SparkJob{
		Name:     jobName,
		Template: "spark-mock-task-runner-job.yaml",
	}

	err = spark.SubmitJob(job)
	if err != nil {
		t.Error(err)
	}

	err = spark.WaitForJobState(job, v1beta2.RunningState, 5*time.Minute)
	if err != nil {
		t.Error(err)
	}

	// TODO the rest
}
