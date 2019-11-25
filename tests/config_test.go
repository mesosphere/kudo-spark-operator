package tests

import (
	"errors"
	"fmt"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"path"
	"strings"
	"testing"
)

func TestMountSparkConfigMap(t *testing.T) {
	err := testMountedConfigMap(
		"sparkConfigMap",
		"resources/test-mount-config-map/spark-defaults.conf",
		"spark-test-configmap",
		"/etc/spark/conf",
		"SPARK_CONF_DIR")
	if err != nil {
		t.Error(err)
	}
}

func TestMountHadoopConfigMap(t *testing.T) {
	err := testMountedConfigMap(
		"hadoopConfigMap",
		"resources/test-mount-config-map/core-site.xml",
		"hadoop-test-configmap",
		"/etc/hadoop/conf",
		"HADOOP_CONF_DIR")
	if err != nil {
		t.Error(err)
	}
}

func testMountedConfigMap(sparkAppConfigParam string, confFilepath string, configMapName string, remoteConfDir string, confDirEnvVarName string) error {

	_, confFilename := path.Split(confFilepath)

	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		return err
	}

	// Create a configmap for spark-defaults.com
	err = utils.CreateConfigMap(spark.K8sClients, configMapName, spark.Namespace)
	if err != nil {
		return err
	}
	defer utils.DeleteConfigName(spark.K8sClients, configMapName, spark.Namespace)

	err = utils.AddFileToConfigMap(spark.K8sClients, configMapName, spark.Namespace, confFilename, confFilepath)
	if err != nil {
		return err
	}

	job := utils.SparkJob{
		Name:     "mount-spark-configmap-test",
		Template: "spark-mock-task-runner-job-mount-config.yaml",
		Params: map[string]interface{}{
			"args":              []string{"1", "600"},
			sparkAppConfigParam: configMapName,
		},
	}

	err = spark.SubmitAndWaitForExecutors(&job)
	if err != nil {
		return err
	}

	// Making sure driver and executor pods have correct volume mounted
	executors, _ := spark.ExecutorPods(job)
	driver, _ := spark.DriverPod(job)

	for _, pod := range append(executors, driver) {
		if !hasConfigMap(pod, configMapName) {
			return errors.New(fmt.Sprintf("Couldn't find volume %s mounted on pod %s", configMapName, pod.Name))
		}

		// Check that *_CONF_DIR is set correctly
		if !utils.IsEnvVarPresentInPod(v1.EnvVar{Name: confDirEnvVarName, Value: remoteConfDir}, pod) {
			return errors.New(fmt.Sprintf("%s is not set to %s on pod %s", confDirEnvVarName, remoteConfDir, pod.Name))
		}

		// Check that the config file is available in the container
		sameContent, err := hasSimilarContents(pod, path.Join(remoteConfDir, confFilename), confFilepath)
		if err != nil {
			return errors.New(fmt.Sprintf("Couldn't compare spark configuration file: %v", err))
		}
		if !sameContent {
			return errors.New(fmt.Sprintf("The content of %s differs locally and in pod %s/%s", confFilename, pod.Namespace, pod.Name))
		} else {
			log.Infof("%s was mounted correctly!", confFilename)
		}
	}

	return nil
}

func hasConfigMap(pod v1.Pod, configMapName string) bool {
	for _, v := range pod.Spec.Volumes {
		if v.ConfigMap != nil && v.ConfigMap.Name == configMapName {
			log.Infof("Found volume %s: %s in pod %s/%s", v.Name, v.ConfigMap.Name, pod.Namespace, pod.Name)
			return true
		}
	}
	return false
}

func hasSimilarContents(pod v1.Pod, remotePath string, localPath string) (bool, error) {
	localContent, err := ioutil.ReadFile(localPath)
	if err != nil {
		return false, err
	}

	var remoteContent string

	err = utils.Retry(func() error {
		remoteContent, err = utils.Kubectl("exec", "-n", pod.Namespace, pod.Name, "--", "cat", remotePath)
		return err
	})

	return strings.Compare(strings.TrimSpace(string(localContent)), strings.TrimSpace(remoteContent)) == 0, nil
}
