package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestEnvBasedSecret(t *testing.T) {
	secretName := "env-based-secret"
	secretKey := "secretKey"
	secretEnv := "SECRET_ENV"
	jobDescription, err := runSecretTest(secretName, "", secretKey)

	if err != nil {
		t.Error(err.Error())
	}

	if strings.Contains(jobDescription, fmt.Sprintf("%s from %s-volume", secretKey, secretName)) {
		log.Infof("Successfully exported environment variable '%s' from secret '%s'", secretEnv, secretName)
	} else {
		t.Errorf("Unnable to export environment variable '%s' from secret '%s'", secretEnv, secretName)
	}
}

func TestFileBasedSecrets(t *testing.T) {
	secretName := "file-based-secret"
	secretPath := "/mnt/secrets"
	jobDescription, err := runSecretTest(secretName, secretPath, "")

	if err != nil {
		t.Error(err.Error())
	}

	if strings.Contains(jobDescription, fmt.Sprintf("%s from %s-volume", secretPath, secretName)) {
		log.Infof("Successfully mounted secret path '%s' from '%s-volume'", secretPath, secretName)
	} else {
		t.Errorf("Unnable to mount secret path '%s' from '%s-volume'", secretPath, secretName)
	}
}

func runSecretTest(secretName string, secretPath string, secretKey string) (string, error) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		return "", err
	}

	client, err := utils.GetK8sClientSet()
	if err != nil {
		return "", err
	}

	secretData := map[string]string{
		secretKey: "secretValue",
	}

	err = utils.CreateSecret(client, secretName, spark.Namespace, secretData)
	if err != nil {
		return "", err
	}

	jobName := "mock-task-runner"
	job := utils.SparkJob{
		Name:     jobName,
		Template: "spark-mock-task-runner-job.yaml",
		Params: map[string]interface{}{
			"args":       []string{"1", "15"},
			"SecretName": secretName,
			"SecretPath": secretPath,
			"SecretKey":  secretKey,
		},
	}

	err = spark.SubmitJob(&job)
	if err != nil {
		return "", err
	}

	err = spark.WaitUntilSucceeded(job)
	if err != nil {
		return "", err
	}

	jobDescription, err := utils.Kubectl(
		"describe",
		"pod",
		"--namespace="+spark.Namespace,
		jobName+"-driver",
	)
	if err != nil {
		return "", err
	}

	return jobDescription, nil
}
