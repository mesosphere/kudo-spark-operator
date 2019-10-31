package tests

import (
	"fmt"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

func TestTenancyTwoOperatorsDifferentNamespaces(t *testing.T) {
	operators := operatorBuilder(2, true, true)
	for _, operator := range operators {
		err := operator.InstallSparkOperator(true)
		assert.NilError(t, err)
	}

	t.Run("TestComponents", func(t *testing.T) {
		verifyComponents(t, operators)
	})
	t.Run("TestWorkloads", func(t *testing.T) {
		verifyWorkloads(t, operators)
	})

	t.Run("TestCRDsDeletion", func(t *testing.T) {
		// verify CRDs are present after one of the operators is deleted
		operators[0].CleanUp()
		assert.Assert(t, crdsInstalled(t), "CRDs are not present!")

		// check that CRDs are deleted after no operator instances left
		operators[1].CleanUp()
		assert.Assert(t, !crdsInstalled(t), "CRDs are not deleted!")
	})
}

func TestTenancyTwoOperatorsSingleNamespace(t *testing.T) {
	operators := operatorBuilder(2, false, true)
	for _, operator := range operators {
		err := operator.InstallSparkOperator(false)
		assert.NilError(t, err)
		defer operator.CleanUp()
	}

	t.Run("TestComponents", func(t *testing.T) {
		verifyComponents(t, operators)
	})
	t.Run("TestWorkloads", func(t *testing.T) {
		verifyWorkloads(t, operators)
	})
}

func TestTenancyTwoOperatorsSameNameDifferentNamespaces(t *testing.T) {
	operators := operatorBuilder(2, true, false)
	for _, operator := range operators {
		err := operator.InstallSparkOperator(true)
		assert.NilError(t, err)
		defer operator.CleanUp()
	}

	t.Run("TestComponents", func(t *testing.T) {
		verifyComponents(t, operators)
	})
	t.Run("TestWorkloads", func(t *testing.T) {
		verifyWorkloads(t, operators)
	})

}

func verifyComponents(t *testing.T, operators []*utils.SparkOperatorInstallation) {
	serviceAccounts := []string{"spark-operator-service-account", "spark-service-account"}
	services := []string{"spark-webhook", "spark-operator-metrics"}
	roles := []string{"spark-driver-role", "%s-spark-role"}

	for _, operator := range operators {
		t.Run("TestServices", func(t *testing.T) {
			for _, service := range services {
				serviceName := fmt.Sprint(operator.InstanceName, "-", service)
				log.Infof("Checking Service \"%s\" is created in namespace \"%s\" for \"%s\"", serviceName,
					operator.Namespace, operator.InstanceName)
				result, err := operator.K8sClients.CoreV1().Services(operator.Namespace).Get(fmt.Sprint(serviceName), v1.GetOptions{})
				assert.NilError(t, err)
				assert.Equal(t, result.Labels["kudo.dev/instance"], operator.InstanceName)
			}
		})

		t.Run("TestServiceAccounts", func(t *testing.T) {
			for _, sa := range serviceAccounts {
				serviceAccount := fmt.Sprint(operator.InstanceName, "-", sa)
				log.Infof("Checking ServiceAccount \"%s\" is created in namespace \"%s\" for \"%s\"", serviceAccount,
					operator.Namespace, operator.InstanceName)
				result, err := operator.K8sClients.CoreV1().ServiceAccounts(operator.Namespace).Get(serviceAccount, v1.GetOptions{})
				assert.NilError(t, err)
				assert.Equal(t, result.Labels["kudo.dev/instance"], operator.InstanceName)
			}
		})

		t.Run("TestRoles", func(t *testing.T) {
			for _, role := range roles {
				if strings.Contains(role, "%s") {
					role = fmt.Sprintf(role, operator.InstanceName)
				}
				log.Infof("Checking Role \"%s\" is created in namespace \"%s\" for \"%s\"",
					role, operator.Namespace, operator.InstanceName)
				result, err := operator.K8sClients.RbacV1().Roles(operator.Namespace).Get(role, v1.GetOptions{})
				assert.NilError(t, err)
				instanceLabel, present := result.Labels["kudo.dev/instance"]
				if present {
					assert.Equal(t, instanceLabel, operator.InstanceName)
				}
			}
		})

		t.Run("TestClusterRole", func(t *testing.T) {
			clusterRole := fmt.Sprintf("%s-%s-cr", operator.InstanceName, operator.Namespace)
			_, err := operator.K8sClients.RbacV1().ClusterRoles().Get(clusterRole, v1.GetOptions{})
			assert.NilError(t, err)
		})

		t.Run("TestClusterRoleBinding", func(t *testing.T) {
			clusterRoleBinding := fmt.Sprintf("%s-%s-crb", operator.InstanceName, operator.Namespace)
			_, err := operator.K8sClients.RbacV1().ClusterRoleBindings().Get(clusterRoleBinding, v1.GetOptions{})
			assert.NilError(t, err)
		})

	}
}

func verifyWorkloads(t *testing.T, operators []*utils.SparkOperatorInstallation) {
	for _, operator := range operators {
		job := utils.SparkJob{
			Name:      "spark-pi",
			Namespace: operator.Namespace,
			Template:  "spark-pi.yaml",
		}

		err := operator.SubmitJob(&job)
		assert.NilError(t, err)

		err = operator.WaitUntilSucceeded(job)
		assert.NilError(t, err)
	}
}

func crdsInstalled(t *testing.T) bool {
	output, err := utils.Kubectl("get", "crds", "-o=name")

	assert.NilError(t, err)

	return strings.Contains(output, "sparkapplications.sparkoperator.k8s.io") &&
		strings.Contains(output, "scheduledsparkapplications.sparkoperator.k8s.io")
}

func operatorBuilder(numberOfOperators int, separateNamespace bool, uniqueOperatorName bool) []*utils.SparkOperatorInstallation {
	const operatorInstanceName = "spark-operator"
	const operatorNamespace = "namespace"

	var operators []*utils.SparkOperatorInstallation
	for i := 1; i <= numberOfOperators; i++ {
		operator := utils.SparkOperatorInstallation{
			InstanceName: operatorInstanceName,
			Namespace:    operatorNamespace,
		}
		if separateNamespace {
			operator.Namespace = fmt.Sprintf("%s-%d", operatorNamespace, i)
		}
		if uniqueOperatorName {
			operator.InstanceName = fmt.Sprintf("%s-%d", operatorInstanceName, i)
		}
		operators = append(operators, &operator)
	}
	return operators
}
