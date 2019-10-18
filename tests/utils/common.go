package utils

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"time"
)

const DefaultNamespace = "kudo-spark-operator-testing"
const DefaultInstanceName = "test-instance"
const rootDirName = "tests"

var OperatorImage = getenvOr("OPERATOR_IMAGE", "mesosphere/kudo-spark-operator:spark-2.4.3-hadoop-2.9-k8s")
var SparkImage = getenvOr("SPARK_IMAGE", "mesosphere/spark:spark-2.4.3-hadoop-2.9-k8s")
var SparkVersion = getenvOr("SPARK_VERSION", "2.4.3")
var TestDir = getenvOr("TEST_DIR", goUpToRootDir())
var KubeConfig = getenvOr("KUBECONFIG", filepath.Join(os.Getenv("HOME"), ".kube", "config"))

func init() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	log.Info("  -- Test run parameters --")
	log.Infof("Operator image:\t\t%s", OperatorImage)
	log.Infof("Spark image:\t\t\t%s", SparkImage)
	log.Infof("Spark version:\t\t%s", SparkVersion)
	log.Infof("Test directory:\t\t%s", TestDir)
	log.Infof("k8s config path:\t\t%s", KubeConfig)
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
		log.Warnf("Retrying... Timeout in %d seconds", int(timeoutPoint.Sub(time.Now()).Seconds()))
		err = fn()
	}
	return err
}
