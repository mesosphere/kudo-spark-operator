package tests

import (
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestSparkOperatorInstallation(t *testing.T) {
	spark := utils.InstallSparkOperator()
	defer spark.CleanUp()

	k8sNamespace, err := spark.Clients.CoreV1().Namespaces().Get(spark.Namespace, v1.GetOptions{})
	if err != nil {
		t.Error(err.Error())
	}

	log.Infof("Spark operator is installed in namespace %s, waiting for Running status", k8sNamespace.Name)
	err = spark.WaitUntilRunning()
	if err != nil {
		t.Error(err.Error())
	}
}

func TestSparkOperatorInstallationWithCustomNamespace(t *testing.T) {
	customNamespace := "custom-test-namespace"

	spark := utils.InstallSparkOperatorWithNamespace(customNamespace)
	defer spark.CleanUp()

	k8sNamespace, err := spark.Clients.CoreV1().Namespaces().Get(spark.Namespace, v1.GetOptions{})
	if err != nil {
		t.Error(err.Error())
	}

	if k8sNamespace.Name != customNamespace {
		t.Errorf("Actual namespace is %s, while %s was expected", k8sNamespace.Name, customNamespace)
	}
}

func TestJobSubmission(t *testing.T) {
	spark := utils.InstallSparkOperator()
	defer spark.CleanUp()

	job := utils.SparkJob{
		Name:         "linear-regression",
		Namespace:    spark.Namespace,
		Image:        utils.SparkImage,
		SparkVersion: utils.SparkVersion,
		Template:     "spark-linear-regression-job.yaml",
	}

	err := spark.SubmitJob(job)
	if err != nil {
		t.Error(err.Error())
	}

	err = spark.WaitUntilSucceeded(job)
	if err != nil {
		t.Error(err.Error())
	}
}
