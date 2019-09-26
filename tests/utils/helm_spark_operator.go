package utils

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"os"
	"os/exec"
)

type SparkOperatorInstallation struct {
	Namespace string
	Clients   *kubernetes.Clientset
}

func InstallSparkOperator() *SparkOperatorInstallation {
	return InstallSparkOperatorWithNamespace(DefaultNamespace)
}

func InstallSparkOperatorWithNamespace(namespace string) *SparkOperatorInstallation {
	clientSet, err := GetK8sClientSet()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	err = installSparkOperatorWithHelm(namespace)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	spark := SparkOperatorInstallation{
		Namespace: namespace,
		Clients:   clientSet,
		// TODO: add client-go/dynamic API client as well
	}
	return &spark
}

func (spark *SparkOperatorInstallation) CleanUp() {
	uninstallSparkOperatorWithHelm(spark.Namespace)
}

func installSparkOperatorWithHelm(namespace string) error {
	log.Info("Installing Spark Operator with helm")
	log.Info("Configuring RBAC")
	rbac := createSparkOperatorNamespace(namespace)
	defer os.Remove(rbac)

	_, err := KubectlApply(namespace, rbac)
	if err != nil {
		return err
	}

	log.Info("Adding the repository")
	addRepoCmd := exec.Command("helm", "repo", "add", "incubator", "http://storage.googleapis.com/kubernetes-charts-incubator")
	_, err = addRepoCmd.Output()
	if err != nil {
		return err
	}

	log.Info("Installing the chart")
	installOperatorCmd := exec.Command("helm", "install", "incubator/sparkoperator", "--namespace", namespace,
		"--name", OperatorName, "--set", "enableWebhook=true,sparkJobNamespace="+namespace+",enableMetrics=true")
	_, err = installOperatorCmd.Output()
	return err
}

func uninstallSparkOperatorWithHelm(namespace string) error {
	log.Info("Uninstalling Spark Operator")
	log.Info("Purging Spark operator")
	installOperatorCmd := exec.Command("helm", "del", "--purge", OperatorName)
	_, err := installOperatorCmd.Output()
	if err != nil {
		return err
	}

	log.Info("Removing the repository")
	addRepoCmd := exec.Command("helm", "repo", "remove", "incubator")
	_, err = addRepoCmd.Output()
	if err != nil {
		return err
	}

	log.Info("Cleaning up RBAC")
	rbac := createSparkOperatorNamespace(namespace)
	defer os.Remove(rbac)
	_, err = KubectlDelete(namespace, rbac)
	return err
}
