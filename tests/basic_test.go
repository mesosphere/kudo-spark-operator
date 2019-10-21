package tests

import (
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
		t.Error(err.Error())
	}

	k8sNamespace, err := spark.Clients.CoreV1().Namespaces().Get(spark.Namespace, v1.GetOptions{})
	if err != nil {
		t.Error(err.Error())
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
		t.Error(err.Error())
	}

	k8sNamespace, err := spark.Clients.CoreV1().Namespaces().Get(spark.Namespace, v1.GetOptions{})
	if err != nil {
		t.Error(err.Error())
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
		t.Error(err)
	}

	job := utils.SparkJob{
		Name:         "linear-regression",
		Namespace:    spark.Namespace,
		Image:        utils.SparkImage,
		SparkVersion: utils.SparkVersion,
		Template:     "spark-linear-regression-job.yaml",
	}

	err = spark.SubmitJob(job)
	if err != nil {
		t.Error(err.Error())
	}

	err = spark.WaitUntilSucceeded(10*time.Minute, job)
	if err != nil {
		t.Error(err.Error())
	}
}
