package tests

import (
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestDefaultServiceAccounts(t *testing.T) {
	const driverSA = "spark-service-account"
	const operatorSA = "spark-operator-service-account"
	spark := utils.SparkOperatorInstallation{
		Namespace: "test-default-service-accounts",
		Params: map[string]string{
			"createOperatorServiceAccount": "true",
			"createSparkServiceAccount":    "true",
			"operatorServiceAccountName":   operatorSA,
			"sparkServiceAccountName":      driverSA,
		},
	}

	err := testServiceAccounts(spark,
		utils.DefaultInstanceName+"-"+operatorSA,
		utils.DefaultInstanceName+"-"+driverSA)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestSparkServiceAccount(t *testing.T) {
	const namespace = "test-provided-spark-sa"
	const driverSA = "spark-driver"
	const operatorSA = "spark-operator-service-account"

	// Prepare a namespace with a service account
	err := createNamespaceAndServiceAccount(namespace, driverSA)
	if err != nil {
		t.Fatal(err)
	}

	spark := utils.SparkOperatorInstallation{
		Namespace:            namespace,
		SkipNamespaceCleanUp: true,
		Params: map[string]string{
			"createOperatorServiceAccount": "true",
			"operatorServiceAccountName":   operatorSA,
			"createSparkServiceAccount":    "false",
			"sparkServiceAccountName":      driverSA,
		},
	}

	err = testServiceAccounts(spark, utils.DefaultInstanceName+"-"+operatorSA, driverSA)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestOperatorServiceAccount(t *testing.T) {
	const namespace = "test-provided-operator-sa"
	const driverSA = "spark-service-account"
	const operatorSA = "operator-test-service-account"

	// Prepare a namespace with a service account
	err := createNamespaceAndServiceAccount(namespace, operatorSA)
	if err != nil {
		t.Fatal(err)
	}

	spark := utils.SparkOperatorInstallation{
		Namespace:            namespace,
		SkipNamespaceCleanUp: true,
		Params: map[string]string{
			"createOperatorServiceAccount": "false",
			"operatorServiceAccountName":   operatorSA,
			"createSparkServiceAccount":    "true",
			"sparkServiceAccountName":      driverSA,
		},
	}

	err = testServiceAccounts(spark, operatorSA, utils.DefaultInstanceName+"-"+driverSA)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func createNamespaceAndServiceAccount(namespace string, saName string) error {
	client, err := utils.GetK8sClientSet()
	if err != nil {
		return err
	}
	_, err = utils.CreateNamespace(client, namespace)
	if err != nil {
		return err
	}

	return utils.CreateServiceAccount(client, saName, namespace)
}

func testServiceAccounts(spark utils.SparkOperatorInstallation, operatorSAName string, driverSAName string) error {
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		return err
	}

	// Verify that SAs exists
	_, err = spark.K8sClients.CoreV1().ServiceAccounts(spark.Namespace).Get(operatorSAName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Can't get operator service account '%s'", operatorSAName)
		return err
	}

	_, err = spark.K8sClients.CoreV1().ServiceAccounts(spark.Namespace).Get(driverSAName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Can't get Spark driver service account '%s'", driverSAName)
		return err
	}

	// Run a test job
	jobName := "mock-task-runner"
	job := utils.SparkJob{
		Name:           jobName,
		Template:       "spark-mock-task-runner-job.yaml",
		ServiceAccount: driverSAName,
		Params: map[string]interface{}{
			"args": []string{"1", "15"},
		},
	}

	err = spark.SubmitJob(&job)
	if err != nil {
		return err
	}

	err = spark.WaitUntilSucceeded(job)
	if err != nil {
		return err
	}

	return err
}
