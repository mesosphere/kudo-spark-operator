package utils

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

const operatorDir = "../kudo-operator"

type SparkOperatorInstallation struct {
	Namespace    string
	InstanceName string
	Clients      *kubernetes.Clientset
	Params       map[string]string
}

func (spark *SparkOperatorInstallation) InstallSparkOperator() error {
	if !isKudoInstalled() {
		return errors.New("can't install Spark operator without KUDO")
	}

	clientSet, err := GetK8sClientSet()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	spark.Clients = clientSet

	// Set default namespace and instance name not specified
	if spark.Namespace == "" {
		spark.Namespace = DefaultNamespace
	}
	if spark.InstanceName == "" {
		spark.InstanceName = DefaultInstanceName
	}

	spark.CleanUp()

	_, err = CreateNamespace(spark.Clients, spark.Namespace)
	if err != nil {
		return err
	}

	// We install CRDs manually for now. It's a temporary workaround soon to be removed.
	KubectlApply(spark.Namespace, "../specs/spark-applications-crds.yaml")

	// Handle parameters
	if spark.Params == nil {
		spark.Params = make(map[string]string)
	}
	if strings.Contains(OperatorImage, ":") {
		operatorImage := strings.Split(OperatorImage, ":")
		spark.Params["operatorImageName"] = operatorImage[0]
		spark.Params["operatorVersion"] = operatorImage[1]
	} else {
		spark.Params["operatorImageName"] = OperatorImage
	}

	err = installKudoPackage(spark.Namespace, operatorDir, spark.InstanceName, spark.Params)
	if err != nil {
		return err
	}

	_, err = KubectlApply(spark.Namespace, "../specs/spark-driver-rbac.yaml")
	if err != nil {
		return err
	}

	return spark.waitForInstanceStatus("COMPLETE")
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
	DropNamespace(spark.Clients, spark.Namespace)
}

func (spark *SparkOperatorInstallation) waitForInstanceStatus(targetStatus string) error {
	log.Infof("Waiting for %s/%s to reach status %s", spark.Namespace, spark.InstanceName, targetStatus)
	return retry(3*time.Minute, 1*time.Second, func() error {
		status, err := spark.getInstanceStatus()
		if err == nil && status != targetStatus {
			err = errors.New(fmt.Sprintf("%s status is %s, but waiting for %s", spark.InstanceName, status, targetStatus))
		}
		return err
	})
}

func (spark *SparkOperatorInstallation) getInstanceStatus() (string, error) {
	status, err := Kubectl("get", "instances.kudo.dev", spark.InstanceName, "--namespace", spark.Namespace, `-o=jsonpath={.status.aggregatedStatus.status}`)
	status = strings.Trim(status, `'`)

	return status, err
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
		var versions []string
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
