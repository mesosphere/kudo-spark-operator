package utils

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"time"
)

const DefaultNamespace = "kudo-spark-operator-testing"
const OperatorName = "kudo-spark-operator"
const rootDirName = "tests"

var OperatorImage = getenvOr("OPERATOR_IMAGE", "gcr.io/spark-operator/spark-operator")
var SparkImage = getenvOr("SPARK_IMAGE", "gcr.io/spark-operator/spark:v2.4.4-gcs-prometheus")
var SparkVersion = getenvOr("SPARK_VERSION", "2.4.4")
var TestDir = getenvOr("TEST_DIR", goUpToRootDir())
var KubeConfig = getenvOr("KUBECONFIG", filepath.Join(os.Getenv("HOME"), ".kube", "config"))

func init() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
}

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

func retry(timeout time.Duration, interval time.Duration, fn func() error) error {
	timeoutPoint := time.Now().Add(timeout)
	var err error

	for err = fn(); err != nil && timeoutPoint.After(time.Now()); {
		log.Warn(err.Error())
		time.Sleep(interval)
		log.Warn("Retrying...")
		err = fn()
	}
	return err
}
