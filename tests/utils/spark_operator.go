package utils

import (
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	operator "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/client/clientset/versioned"
	petname "github.com/dustinkirkland/golang-petname"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

const operatorDir = "../operators/repository/spark/operator"

type SparkOperatorInstallation struct {
	Namespace            string
	InstanceName         string
	SkipNamespaceCleanUp bool
	K8sClients           *kubernetes.Clientset
	SparkClients         *operator.Clientset
	Params               map[string]string
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
		spark.InstanceName = GenerateInstanceName()
	}

	if !spark.SkipNamespaceCleanUp {
		spark.CleanUp()

		_, err = CreateNamespace(spark.K8sClients, spark.Namespace)
		if err != nil {
			return err
		}
	}

	log.Infof("Installing KUDO spark operator in %s", spark.Namespace)

	// Handle parameters
	if spark.Params == nil {
		spark.Params = make(map[string]string)
	}
	if strings.Contains(OperatorImage, ":") {
		// handle the case, when using local docker registry (e.g. registry:5000/kudo-spark-operator:2.4.5-1.0.0)
		tagIndex := strings.LastIndex(OperatorImage, ":")
		spark.Params["operatorImageName"] = OperatorImage[0:tagIndex]
		spark.Params["operatorVersion"] = OperatorImage[tagIndex+1:]
	} else {
		spark.Params["operatorImageName"] = OperatorImage
	}

	err = installKudoPackage(spark.Namespace, operatorDir, spark.InstanceName, spark.Params)
	if err != nil {
		return err
	}

	return spark.waitForInstanceStatus("COMPLETE")
}

func (spark *SparkOperatorInstallation) CleanUp() {
	// So far multiple Spark operator instances in one namespace is not a supported configuration, so whole namespace will be cleaned
	log.Infof("Uninstalling ALL kudo spark operator instances and versions from %s", spark.Namespace)
	instances, _ := getInstanceNames(spark.Namespace)

	if instances != nil {
		for _, instance := range instances {
			unistallKudoPackage(spark.Namespace, instance)
		}
	}
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
	status, err := Kubectl("get", "instances.kudo.dev", spark.InstanceName, "--namespace", spark.Namespace, `-o=jsonpath={.spec..status}`)
	status = strings.Trim(status, `'`)

	return status, err
}

func (spark *SparkOperatorInstallation) WaitForJobState(job SparkJob, state v1beta2.ApplicationStateType) error {
	log.Infof("Waiting for SparkApplication \"%s\" to reach \"%s\" state", job.Name, state)
	err := Retry(func() error {
		app, err := spark.SparkClients.SparkoperatorV1beta2().SparkApplications(spark.Namespace).Get(job.Name, v1.GetOptions{})
		if err != nil {
			return err
		} else if app.Status.AppState.State != state {
			return errors.New(fmt.Sprintf("SparkApplication \"%s\" state is %s", job.Name, app.Status.AppState.State))
		}
		return nil
	})

	if err == nil {
		log.Infof("SparkApplication \"%s\" is now \"%s\"", job.Name, state)
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

func (spark *SparkOperatorInstallation) GetOperatorPodName() (string, error) {
	return Kubectl("get", "pod",
		"--selector", "app.kubernetes.io/name=spark",
		"--namespace", spark.Namespace,
		"-o=jsonpath={.items[*].metadata.name}")
}

func GenerateInstanceName() string {
	return fmt.Sprintf("spark-%s", petname.Generate(2, "-"))
}
