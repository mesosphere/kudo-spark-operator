package utils

import (
	"os"
	"path/filepath"
)

const DefaultNamespace = "kudo-spark-testing"
const OperatorName = "kudo-spark-operator"

var SparkDockerImage = getenvOr("SPARK_DOCKER_IMAGE", "gcr.io/spark-operator/spark:v2.4.0-gcs-prometheus")
var TestDir = getenvOr("TEST_DIR", ".")
var KubeConfig = getenvOr("KUBECONFIG", filepath.Join(os.Getenv("HOME"), ".kube", "config"))

func getenvOr(key string, defaultValue string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		val = defaultValue
	}
	return val
}
