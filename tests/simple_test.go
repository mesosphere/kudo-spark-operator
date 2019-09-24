package tests

import (
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

func TestSparkOperatorInstallation(t *testing.T) {
	spark := utils.InstallSparkOperator()
	defer spark.CleanUp()

	k8sNamespace, err := spark.Clients.CoreV1().Namespaces().Get(spark.Namespace, v1.GetOptions{})
	if err != nil {
		t.Error(err.Error())
	}

	log.Infof("Spark operator is installed in namespace %s", k8sNamespace.Name)

	var pods *coreV1.PodList
	pods, err = spark.Clients.CoreV1().Pods(k8sNamespace.Name).List(v1.ListOptions{})

	if len(pods.Items) != 1 {
		t.Error("More than one pod is found in spark operator namespace")
	} else if !strings.HasPrefix(pods.Items[0].Name, utils.OperatorName) {
		t.Errorf("Found unexpected spark operator pod name: %s", pods.Items[0].Name)
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
