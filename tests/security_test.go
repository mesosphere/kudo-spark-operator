package tests

import (
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type serviceAccountTestCase struct {
	name                   string
	namespace              string
	operatorServiceAccount string
	driverServiceAccount   string
}

func TestServiceAccounts(t *testing.T) {
	testCases := []serviceAccountTestCase{
		{
			name:      "DefaultConfiguration",
			namespace: "sa-test-default",
		},
		{
			name:                   "ProvidedOperatorServiceAccount",
			namespace:              "sa-test-operator",
			operatorServiceAccount: "custom-operator-sa",
		},
		{
			name:                 "ProvidedSparkServiceAccount",
			namespace:            "sa-test-spark",
			driverServiceAccount: "custom-spark-sa",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runServiceAccountTestCase(t, tc)
			if err != nil {
				t.Errorf("Test case: %v\nfailed with error: %s", tc, err)
			}
		})
	}
}

func runServiceAccountTestCase(t *testing.T, tc serviceAccountTestCase) error {
	client, err := utils.GetK8sClientSet()
	if err != nil {
		return err
	}
	_, err = utils.CreateNamespace(client, tc.namespace)
	if err != nil {
		return err
	}

	// Prepare parameters and expected SA names
	var expectedDriverSA, expectedOperatorSA string
	sparkParams := make(map[string]string)

	if tc.operatorServiceAccount != "" {
		utils.CreateServiceAccount(client, tc.operatorServiceAccount, tc.namespace)
		sparkParams["createOperatorServiceAccount"] = "false"
		sparkParams["operatorServiceAccountName"] = tc.operatorServiceAccount
		expectedOperatorSA = tc.operatorServiceAccount
	} else {
		sparkParams["createOperatorServiceAccount"] = "true"
		sparkParams["operatorServiceAccountName"] = "spark-operator-service-account"
		expectedOperatorSA = utils.DefaultInstanceName + "-" + sparkParams["operatorServiceAccountName"]
	}

	if tc.driverServiceAccount != "" {
		utils.CreateServiceAccount(client, tc.driverServiceAccount, tc.namespace)
		sparkParams["createSparkServiceAccount"] = "false"
		sparkParams["sparkServiceAccountName"] = tc.driverServiceAccount
		expectedDriverSA = tc.driverServiceAccount
	} else {
		sparkParams["createSparkServiceAccount"] = "true"
		sparkParams["sparkServiceAccountName"] = "spark-service-account"
		expectedDriverSA = utils.DefaultInstanceName + "-" + sparkParams["sparkServiceAccountName"]
	}

	// Install spark operator
	spark := utils.SparkOperatorInstallation{
		Namespace:            tc.namespace,
		SkipNamespaceCleanUp: true,
		Params:               sparkParams,
	}

	err = spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		return err
	}

	// Verify that SAs exists
	_, err = spark.K8sClients.CoreV1().ServiceAccounts(spark.Namespace).Get(expectedOperatorSA, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Can't get operator service account '%s'", expectedOperatorSA)
		return err
	}

	_, err = spark.K8sClients.CoreV1().ServiceAccounts(spark.Namespace).Get(expectedDriverSA, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Can't get Spark driver service account '%s'", expectedDriverSA)
		return err
	}

	// Run a test job
	jobName := "mock-task-runner"
	job := utils.SparkJob{
		Name:           jobName,
		Template:       "spark-mock-task-runner-job.yaml",
		ServiceAccount: expectedDriverSA,
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
