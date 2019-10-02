package utils

import (
	log "github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"os/exec"
	"strings"
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
	}
	return &spark
}

func (spark *SparkOperatorInstallation) CleanUp() {
	uninstallSparkOperatorWithHelm(spark.Namespace)
}

func (spark *SparkOperatorInstallation) OperatorPod() (coreV1.Pod, error) {
	pods, err := spark.Clients.CoreV1().Pods(spark.Namespace).List(v1.ListOptions{LabelSelector: "app.kubernetes.io/name=sparkoperator"})

	if err != nil {
		return pods.Items[0], nil
	} else if len(pods.Items) != 1 {
		return pods.Items[0], nil
	} else if !strings.HasPrefix(pods.Items[0].Name, OperatorName) {
		return pods.Items[0], nil
	}

	return pods.Items[0], nil
}

func (spark *SparkOperatorInstallation) WaitUntilRunning() error {
	pod, err := spark.OperatorPod()
	if err != nil {
		return err
	}

	return waitForPodStatusPhase(spark.Clients, pod.Name, spark.Namespace, "Running")
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

	log.Info("Initializing helm")
	initCmd := exec.Command("helm", "init")
	out, err := initCmd.CombinedOutput()
	log.Infof("Helm output: \n%s", out)
	if err != nil {
		return err
	}

	log.Info("Adding the repository")
	addRepoCmd := exec.Command("helm", "repo", "add", "incubator", "http://storage.googleapis.com/kubernetes-charts-incubator")
	out, err = addRepoCmd.CombinedOutput()
	log.Infof("Helm output: \n%s", out)
	if err != nil {
		return err
	}

	log.Info("Installing the chart")
	operatorImage := strings.Split(OperatorImage, ":")
	installOperatorCmd := exec.Command("helm", "install", "incubator/sparkoperator", "--namespace", namespace,
		"--name", OperatorName, "--set", "sparkJobNamespace="+namespace+
			",enableMetrics=true,operatorImageName="+operatorImage[0]+",operatorVersion="+operatorImage[1])
	out, err = installOperatorCmd.CombinedOutput()
	log.Infof("Helm output: \n%s", out)
	return err
}

func uninstallSparkOperatorWithHelm(namespace string) error {
	log.Info("Uninstalling Spark Operator")
	log.Info("Purging Spark operator")
	installOperatorCmd := exec.Command("helm", "del", "--purge", OperatorName)
	out, err := installOperatorCmd.CombinedOutput()
	log.Infof("Helm output: \n%s", out)
	if err != nil {
		return err
	}

	log.Info("Removing the repository")
	addRepoCmd := exec.Command("helm", "repo", "remove", "incubator")
	out, err = addRepoCmd.CombinedOutput()
	log.Infof("Helm output: \n%s", out)
	if err != nil {
		return err
	}

	log.Info("Cleaning up RBAC")
	rbac := createSparkOperatorNamespace(namespace)
	defer os.Remove(rbac)
	_, err = KubectlDelete(namespace, rbac)
	return err
}
