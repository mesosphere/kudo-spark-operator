package utils

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"
)

type SparkOperatorInstallation struct {
	Namespace    string
	InstanceName string
	Clients      *kubernetes.Clientset
}

type SparkInstance struct {
	Name      string
	Namespace string
	Status    string
}

func InstallSparkOperator() (*SparkOperatorInstallation, error) {
	return InstallSparkOperatorWithNamespace(DefaultNamespace)
}

func InstallSparkOperatorWithNamespace(namespace string) (*SparkOperatorInstallation, error) {
	clientSet, err := GetK8sClientSet()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	spark := SparkOperatorInstallation{
		Namespace: namespace,
		Clients:   clientSet,
	}
	spark.CleanUp()

	installScript := exec.Command("bash", path.Join(TestDir, "../scripts/install_operator.sh"))
	if strings.Contains(OperatorImage, ":") {
		operatorImage := strings.Split(OperatorImage, ":")
		installScript.Env = append(os.Environ(),
			"NAMESPACE="+namespace,
			"OPERATOR_IMAGE_NAME="+operatorImage[0],
			"OPERATOR_VERSION="+operatorImage[1])
	} else {
		installScript.Env = append(os.Environ(),
			"NAMESPACE="+namespace,
			"OPERATOR_IMAGE_NAME="+OperatorImage)
	}
	out, err := installScript.CombinedOutput()
	log.Infof("KUDO Spark operation installation script output:\n%s", out)
	if err != nil {
		return nil, err
	}

	instanceNameRE, _ := regexp.Compile(`instance\.kudo\.dev/v1alpha1/(spark-\w{6}) created`)
	parsedInstanceName := instanceNameRE.FindStringSubmatch(string(out))
	spark.InstanceName = parsedInstanceName[1]

	spark.waitForInstanceStatus("COMPLETE")

	return &spark, nil
}

func (spark *SparkOperatorInstallation) CleanUp() {
	// So far multiple Spark operator instances in one namespace is not a supported configuration, so whole namespace will be cleaned
	log.Infof("Uninstalling ALL kudo spark operator instances and versions from %s", spark.Namespace)
	instances, _ := getInstanceNames(spark.Namespace)
	versions, _ := getOperatorVersions(spark.Namespace)

	if instances != nil {
		for _, instance := range instances {
			DeleteResource(spark.Namespace, "instance.kudo.dev", instance)
		}
	}

	if versions != nil {
		for _, version := range versions {
			DeleteResource(spark.Namespace, "operatorversion.kudo.dev", version)
		}
	}

	DeleteResource(spark.Namespace, "operator.kudo.dev", "spark")
	//DropNamespace(spark.Clients, spark.Namespace)
}

func (spark *SparkOperatorInstallation) waitForInstanceStatus(targetStatus string) error {
	log.Infof("Waiting for %s/%s to reach status %s", spark.Namespace, spark.InstanceName, targetStatus)
	return retry(3*time.Minute, 1*time.Second, func() error {
		instance, err := spark.getInstance()
		if err == nil && instance.Status != targetStatus {
			err = errors.New(fmt.Sprintf("%s status is %s, but waiting for %s", instance.Name, instance.Status, targetStatus))
		}
		return err
	})
}

func (spark *SparkOperatorInstallation) getInstance() (SparkInstance, error) {
	status, err := Kubectl("get", "instances.kudo.dev", spark.InstanceName, "--namespace", spark.Namespace, `-o=jsonpath={.status.aggregatedStatus.status}`)
	status = strings.Trim(status, `'`)

	if err != nil {
		return SparkInstance{}, err
	}

	return SparkInstance{
		Name:      spark.InstanceName,
		Namespace: spark.Namespace,
		Status:    status,
	}, nil
}

func getInstanceNames(namespace string) ([]string, error) {
	jsonpathExpr := `-o=jsonpath={range .items[?(@.metadata.labels.kudo\.dev/operator=="spark")]}{.metadata.name}{"\n"}`
	out, err := Kubectl("get", "instances.kudo.dev", "--namespace", namespace, jsonpathExpr)

	if err != nil {
		return nil, err
	}

	if len(out) > 0 {
		names := strings.Split(out, "\n")
		return names, nil
	} else {
		return nil, nil
	}
}

func getOperatorVersions(namespace string) ([]string, error) {
	jsonpathExpr := `-o=jsonpath={range .items[?(@.metadata.labels.kudo\.dev/operator=="spark")]}{.spec.operatorVersion.name}{"\n"}`
	out, err := Kubectl("get", "instances.kudo.dev", "--namespace", namespace, jsonpathExpr)

	if err != nil {
		return nil, err
	}

	if len(out) > 0 {
		versions := make([]string, 1)
		for _, version := range strings.Split(out, "\n") {
			for _, contained := range versions {
				if contained == version {
					continue
				}
			}
			versions = append(versions, version)
		}
		return versions, nil
	} else {
		return nil, nil
	}
}
