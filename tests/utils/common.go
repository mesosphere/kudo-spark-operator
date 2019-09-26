package utils

import (
	"os"
	"path"
	"path/filepath"
)

const DefaultNamespace = "kudo-spark-operator-testing"
const OperatorName = "kudo-spark-operator"
const rootDirName = "tests"

var SparkDockerImage = getenvOr("SPARK_DOCKER_IMAGE", "gcr.io/spark-operator/spark:v2.4.0-gcs-prometheus")
var TestDir = getenvOr("TEST_DIR", goUpToRootDir())
var KubeConfig = getenvOr("KUBECONFIG", filepath.Join(os.Getenv("HOME"), ".kube", "config"))

func getenvOr(key string, defaultValue string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		val = defaultValue
	}
	return val
}

func goUpToRootDir() string {
	workDir, _ := os.Getwd()
	for path.Base(workDir) != rootDirName {
		workDir = path.Dir(workDir)
		if workDir == "/" {
			panic("Can't find root test directory")
		}
	}
	return workDir
}
