package tests

import (
	"errors"
	"fmt"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"testing"
)

type securityTestCase interface {
	common() *commonTestCaseDetails
	getJobServiceAccount() string
	prepare(*kubernetes.Clientset) error
	cleanup(*kubernetes.Clientset)
	verify(*utils.SparkOperatorInstallation) error
}

type commonTestCaseDetails struct {
	name      string
	namespace string
	params    map[string]string
}

func (c *commonTestCaseDetails) common() *commonTestCaseDetails {
	return c
}

func (c *commonTestCaseDetails) getJobServiceAccount() string {
	return utils.DefaultInstanceName + utils.DefaultServiceAccountSuffix
}

type serviceAccountTestCase struct {
	commonTestCaseDetails
	expectedOperatorSA string
	prepareOperatorSA  bool
	expectedDriverSA   string
	prepareDriverSA    bool
}

func (tc *serviceAccountTestCase) getJobServiceAccount() string {
	return tc.expectedDriverSA
}

// Prepare SAs before installation if needed
func (tc *serviceAccountTestCase) prepare(client *kubernetes.Clientset) error {
	if tc.prepareOperatorSA {
		err := utils.CreateServiceAccount(client, tc.expectedOperatorSA, tc.namespace)
		if err != nil {
			log.Errorf("Can't create operator service account '%s'", tc.expectedOperatorSA)
			return err
		}
	}
	if tc.prepareDriverSA {
		err := utils.CreateServiceAccount(client, tc.expectedDriverSA, tc.namespace)
		if err != nil {
			log.Errorf("Can't create spark driver service account '%s'", tc.expectedDriverSA)
			return err
		}
	}

	return nil
}

func (tc *serviceAccountTestCase) cleanup(*kubernetes.Clientset) {
	// Nothing to clean up
}

// Verify that SAs exists
func (tc *serviceAccountTestCase) verify(spark *utils.SparkOperatorInstallation) error {

	_, err := spark.K8sClients.CoreV1().ServiceAccounts(spark.Namespace).Get(tc.expectedOperatorSA, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Can't get operator service account '%s'", tc.expectedOperatorSA)
		return err
	}

	_, err = spark.K8sClients.CoreV1().ServiceAccounts(spark.Namespace).Get(tc.expectedDriverSA, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Can't get Spark driver service account '%s'", tc.expectedDriverSA)
		return err
	}

	return nil
}

type rbacTestCase struct {
	commonTestCaseDetails
	prepareRBAC bool
}

func (tc *rbacTestCase) prepare(client *kubernetes.Clientset) error {
	if tc.prepareRBAC {
		log.Infof("Preparing RBAC entities before installing the operator")
		const rbacTemplate = "security_test_rbac.yaml"
		const sparkSA = utils.DefaultInstanceName + utils.DefaultServiceAccountSuffix
		const operatorSA = "spark-operator-test-service-account"

		// Create and apply RBAC template
		err := utils.KubectlApplyTemplate(tc.namespace, rbacTemplate, map[string]interface{}{
			"service-account":           sparkSA,
			"operator-service-account":  operatorSA,
			"service-account-namespace": tc.namespace,
			"instance-name":             utils.DefaultInstanceName,
		})
		if err != nil {
			return err
		}

		// Add additional parameters to use provided service accounts
		tc.params["createOperatorServiceAccount"] = "false"
		tc.params["createSparkServiceAccount"] = "false"
		tc.params["operatorServiceAccountName"] = operatorSA
		tc.params["sparkServiceAccountName"] = sparkSA
	}
	return nil
}

// Clean up cluster-wide resources at the end of the test
func (tc *rbacTestCase) cleanup(*kubernetes.Clientset) {
	utils.DeleteResource("default", "clusterrole", "spark-operator-test-cluster-role")
	utils.DeleteResource("default", "clusterrolebinding", "spark-operator-test-cluster-role-binding")
}

func (tc *rbacTestCase) verify(spark *utils.SparkOperatorInstallation) error {
	// Verify spark and operator roles
	croles, err := spark.K8sClients.RbacV1().ClusterRoles().List(metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/instance = " + spark.InstanceName,
	})
	if err != nil {
		return err
	} else if len(croles.Items) != 1 {
		return errors.New(fmt.Sprintf("Was expecting to find only one ClusterRole for the instance, but %d were found instead", len(croles.Items)))
	}
	log.Infof("Found a ClusterRole for instance %s: %s", spark.InstanceName, croles.Items[0].Name)

	roles, err := spark.K8sClients.RbacV1().Roles(spark.Namespace).List(metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/instance = " + spark.InstanceName,
	})
	if err != nil {
		return err
	} else if len(roles.Items) != 1 {
		return errors.New(fmt.Sprintf("Was expecting to find only one Role for the instance, but %d were found instead", len(roles.Items)))
	}
	log.Infof("Found a Role for instance %s: %s", spark.InstanceName, roles.Items[0].Name)

	return nil
}

func TestServiceAccounts(t *testing.T) {
	testCases := []serviceAccountTestCase{
		{
			commonTestCaseDetails: commonTestCaseDetails{
				name:      "DefaultConfiguration",
				namespace: "sa-test-default",
				params: map[string]string{
					"operatorServiceAccountName": "spark-operator-service-account",
					"sparkServiceAccountName":    "spark-service-account",
				},
			},
			expectedOperatorSA: utils.DefaultInstanceName + "-spark-operator-service-account",
			expectedDriverSA:   utils.DefaultInstanceName + "-spark-service-account",
		},
		{
			commonTestCaseDetails: commonTestCaseDetails{
				name:      "ProvidedOperatorServiceAccount",
				namespace: "sa-test-operator",
				params: map[string]string{
					"operatorServiceAccountName":   "custom-operator-sa",
					"createOperatorServiceAccount": "false",
					"sparkServiceAccountName":      "spark-service-account",
				},
			},
			prepareOperatorSA:  true,
			expectedOperatorSA: "custom-operator-sa",
			expectedDriverSA:   utils.DefaultInstanceName + "-spark-service-account",
		},
		{
			commonTestCaseDetails: commonTestCaseDetails{
				name:      "ProvidedSparkServiceAccount",
				namespace: "sa-test-spark",
				params: map[string]string{
					"operatorServiceAccountName": "spark-operator-service-account",
					"createSparkServiceAccount":  "false",
					"sparkServiceAccountName":    "custom-spark-sa",
				},
			},
			prepareDriverSA:    true,
			expectedOperatorSA: utils.DefaultInstanceName + "-spark-operator-service-account",
			expectedDriverSA:   "custom-spark-sa",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runTestCase(&tc)
			if err != nil {
				t.Errorf("Test case: %v\nfailed with error: %s", tc, err)
			}
		})
	}
}

func TestRoleBasedAccessControl(t *testing.T) {
	testCases := []rbacTestCase{
		{
			commonTestCaseDetails: commonTestCaseDetails{
				name:      "CreateDefaultRBAC",
				namespace: "rbac-test-default",
				params: map[string]string{
					"createRBAC": "true",
				},
			},
		},
		{
			commonTestCaseDetails: commonTestCaseDetails{
				name:      "ProvidedRBAC",
				namespace: "rbac-test-provided",
				params: map[string]string{
					"createRBAC": "false",
				},
			},
			prepareRBAC: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runTestCase(&tc)
			if err != nil {
				t.Errorf("Test case: %v\nfailed with error: %s", tc, err)
			}
		})
	}
}

func runTestCase(tc securityTestCase) error {
	client, err := utils.GetK8sClientSet()
	if err != nil {
		return err
	}

	utils.DropNamespace(client, tc.common().namespace)
	_, err = utils.CreateNamespace(client, tc.common().namespace)
	if err != nil {
		return err
	}

	err = tc.prepare(client)
	defer tc.cleanup(client)
	if err != nil {
		return err
	}

	// Install spark operator
	spark := utils.SparkOperatorInstallation{
		Namespace:            tc.common().namespace,
		SkipNamespaceCleanUp: true,
		Params:               tc.common().params,
	}

	err = spark.InstallSparkOperator()
	defer spark.CleanUp()
	if err != nil {
		return err
	}

	err = tc.verify(&spark)
	if err != nil {
		return err
	}

	// Run a test job
	jobName := "mock-task-runner"
	job := utils.SparkJob{
		Name:           jobName,
		Template:       "spark-mock-task-runner-job.yaml",
		ServiceAccount: tc.getJobServiceAccount(),
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
