package tests

import (
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type serviceAccountTestCase struct {
	name               string
	namespace          string
	params             map[string]string
	expectedOperatorSA string
	prepareOperatorSA  bool
	expectedDriverSA   string
	prepareDriverSA    bool
}

func TestServiceAccounts(t *testing.T) {
	testCases := []serviceAccountTestCase{
		{
			name:      "DefaultConfiguration",
			namespace: "sa-test-default",
			params: map[string]string{
				"operatorServiceAccountName": "spark-operator-service-account",
				"sparkServiceAccountName":    "spark-service-account",
			},
			expectedOperatorSA: utils.DefaultInstanceName + "-spark-operator-service-account",
			expectedDriverSA:   utils.DefaultInstanceName + "-spark-service-account",
		},
		{
			name:      "ProvidedOperatorServiceAccount",
			namespace: "sa-test-operator",
			params: map[string]string{
				"operatorServiceAccountName":   "custom-operator-sa",
				"createOperatorServiceAccount": "false",
				"sparkServiceAccountName":      "spark-service-account",
			},
			prepareOperatorSA:  true,
			expectedOperatorSA: "custom-operator-sa",
			expectedDriverSA:   utils.DefaultInstanceName + "-spark-service-account",
		},
		{
			name:      "ProvidedSparkServiceAccount",
			namespace: "sa-test-spark",
			params: map[string]string{
				"operatorServiceAccountName": "spark-operator-service-account",
				"createSparkServiceAccount":  "false",
				"sparkServiceAccountName":    "custom-spark-sa",
			},
			prepareDriverSA:    true,
			expectedOperatorSA: utils.DefaultInstanceName + "-spark-operator-service-account",
			expectedDriverSA:   "custom-spark-sa",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runServiceAccountTestCase(tc)
			if err != nil {
				t.Errorf("Test case: %v\nfailed with error: %s", tc, err)
			}
		})
	}
}

func runServiceAccountTestCase(tc serviceAccountTestCase) error {
	client, err := utils.GetK8sClientSet()
	if err != nil {
		return err
	}
	_, err = utils.CreateNamespace(client, tc.namespace)
	if err != nil {
		return err
	}

	// Prepare SAs before installation if needed
	if tc.prepareOperatorSA {
		err = utils.CreateServiceAccount(client, tc.expectedOperatorSA, tc.namespace)
		if err != nil {
			log.Errorf("Can't create operator service account '%s'", tc.expectedOperatorSA)
			return err
		}
	}
	if tc.prepareDriverSA {
		err = utils.CreateServiceAccount(client, tc.expectedDriverSA, tc.namespace)
		if err != nil {
			log.Errorf("Can't create spark driver service account '%s'", tc.expectedDriverSA)
			return err
		}
	}

	// Install spark operator
	spark := utils.SparkOperatorInstallation{
		Namespace:            tc.namespace,
		SkipNamespaceCleanUp: true,
		Params:               tc.params,
	}

	err = spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		return err
	}

	// Verify that SAs exists
	_, err = spark.K8sClients.CoreV1().ServiceAccounts(spark.Namespace).Get(tc.expectedOperatorSA, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Can't get operator service account '%s'", tc.expectedOperatorSA)
		return err
	}

	_, err = spark.K8sClients.CoreV1().ServiceAccounts(spark.Namespace).Get(tc.expectedDriverSA, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Can't get Spark driver service account '%s'", tc.expectedDriverSA)
		return err
	}

	// Run a test job
	jobName := "mock-task-runner"
	job := utils.SparkJob{
		Name:           jobName,
		Template:       "spark-mock-task-runner-job.yaml",
		ServiceAccount: tc.expectedDriverSA,
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
