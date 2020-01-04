package utils

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const DefaultNamespace = "kudo-spark-operator-testing"
const OperatorName = "spark"
const DefaultInstanceName = "test-instance"
const DefaultServiceAccountSuffix = "-spark-service-account"
const rootDirName = "tests"
const cmdLogFormat = ">%s %v\n%s"
const DefaultRetryInterval = 5 * time.Second
const DefaultRetryTimeout = 10 * time.Minute

var OperatorImage = GetenvOr("OPERATOR_IMAGE", "mesosphere/kudo-spark-operator:spark-2.4.3-hadoop-2.9-k8s")
var SparkImage = GetenvOr("SPARK_IMAGE", "mesosphere/spark:spark-2.4.3-hadoop-2.9-k8s")
var SparkVersion = GetenvOr("SPARK_VERSION", "2.4.4")
var TestDir = GetenvOr("TEST_DIR", goUpToRootDir())
var KubeConfig = GetenvOr("KUBECONFIG", filepath.Join(os.Getenv("HOME"), ".kube", "config"))

func init() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	log.Info("  -- Test run parameters --")
	log.Infof("Operator image:\t\t\t%s", OperatorImage)
	log.Infof("Spark image:\t\t\t%s", SparkImage)
	log.Infof("Spark version:\t\t\t%s", SparkVersion)
	log.Infof("Test directory:\t\t\t%s", TestDir)
	log.Infof("k8s config path:\t\t\t%s", KubeConfig)
}

func GetenvOr(key string, defaultValue string) string {
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

func Retry(fn func() error) error {
	return RetryWithTimeout(DefaultRetryTimeout, DefaultRetryInterval, fn)
}

func RetryWithTimeout(timeout time.Duration, interval time.Duration, fn func() error) error {
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

func runAndLogCommandOutput(cmd *exec.Cmd) (string, error) {
	out, err := cmd.CombinedOutput()

	if err == nil {
		log.Infof(cmdLogFormat, cmd.Path, cmd.Args, out)
	} else {
		log.Errorf(cmdLogFormat, cmd.Path, cmd.Args, out)
	}
	return strings.TrimSpace(string(out)), err
}
