package utils

import (
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	operator "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

const operatorDir = "../kudo-operator/operator"

type SparkOperatorInstallation struct {
	Namespace    string
	InstanceName string
	K8sClients   *kubernetes.Clientset
	SparkClients *operator.Clientset
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
	spark.K8sClients = clientSet

	sparkClientSet, err := getSparkOperatorClientSet()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	spark.SparkClients = sparkClientSet

	// Set default namespace and instance name not specified
	if spark.Namespace == "" {
		spark.Namespace = DefaultNamespace
	}
	if spark.InstanceName == "" {
		spark.InstanceName = DefaultInstanceName
	}

	spark.CleanUp()

	log.Infof("Installing KUDO spark operator in %s", spark.Namespace)

	_, err = CreateNamespace(spark.K8sClients, spark.Namespace)
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

	err = KubectlApply(spark.Namespace, "../specs/spark-driver-rbac.yaml")
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
	DropNamespace(spark.K8sClients, spark.Namespace)
}

func getSparkOperatorClientSet() (*operator.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfig)
	if err != nil {
		panic(err.Error())
	}

	return operator.NewForConfig(config)
}

func (spark *SparkOperatorInstallation) waitForInstanceStatus(targetStatus string) error {
	log.Infof("Waiting for %s/%s to reach status %s", spark.Namespace, spark.InstanceName, targetStatus)
	return Retry(func() error {
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

func (spark *SparkOperatorInstallation) WaitForJobState(job SparkJob, state v1beta2.ApplicationStateType) error {
	log.Infof("Waiting for spark application %s to reach %s state", job.Name, state)
	err := Retry(func() error {
		app, err := spark.SparkClients.SparkoperatorV1beta2().SparkApplications(spark.Namespace).Get(job.Name, v1.GetOptions{})
		if err != nil {
			return err
		} else if app.Status.AppState.State != state {
			return errors.New(fmt.Sprintf("%s state is %s", job.Name, app.Status.AppState.State))
		}
		return nil
	})

	if err == nil {
		log.Infof("Spark application %s is now %s", job.Name, state)
	}

	return err
}

func (spark *SparkOperatorInstallation) GetExecutorState(job SparkJob) (map[string]v1beta2.ExecutorState, error) {
	log.Infof("Getting %s executors status", job.Name)
	app, err := spark.SparkClients.SparkoperatorV1beta2().SparkApplications(spark.Namespace).Get(job.Name, v1.GetOptions{})
	if err != nil {
		return nil, err
	} else {
		for k, v := range app.Status.ExecutorState {
			log.Infof("%s is %s", k, v)
		}
		return app.Status.ExecutorState, err
	}
}

func (spark *SparkOperatorInstallation) DeleteJob(job SparkJob) {
	log.Infof("Deleting job %s", job.Name)
	gracePeriod := int64(0)
	propagationPolicy := v1.DeletePropagationForeground
	options := v1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
		PropagationPolicy:  &propagationPolicy,
	}
	spark.SparkClients.SparkoperatorV1beta2().SparkApplications(spark.Namespace).Delete(job.Name, &options)
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
