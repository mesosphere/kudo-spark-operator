package tests

import (
	"fmt"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

// note: this shouldn't be changed as per this section:
//https://github.com/mesosphere/spark-on-k8s-operator/blob/master/docs/user-guide.md#mounting-secrets
const hadoopTokenFileName = "hadoop.token"

var (
	resourceFolder         = "resources/hdfs-kerberos"
	namespace              = "hdfs-kerberos"
	hdfsKerberosDeployment = []Resource{
		{
			name: "configmaps",
			path: "configmaps",
		},
		{
			name: "volumes",
			path: "volumes",
		},
		{
			name: "kerberos",
			path: "kerberos-deployment.yaml",
			wait: true,
		},
		{
			name: "hdfs-namenode",
			path: "namenode-deployment.yaml",
			wait: true,
		},
		{
			name: "hdfs-datanode",
			path: "datanode-deployment.yaml",
			wait: true,
		},
	}
	eventLogDir       = "hdfs://namenode.hdfs-kerberos.svc.cluster.local:9000/history"
	hadoopTokenSecret = "hadoop-token"
	hadoopTokenPath   = fmt.Sprint("/tmp/", hadoopTokenFileName)
	waitTimeout       = 5 * time.Minute
)

type Resource struct {
	name string
	path string
	wait bool
}

type HdfsIntegrationSuite struct {
	operator utils.SparkOperatorInstallation
	suite.Suite
}

func TestHdfsIntegrationSuite(t *testing.T) {
	suite.Run(t, new(HdfsIntegrationSuite))
}

func (suite *HdfsIntegrationSuite) SetupSuite() {
	if _, err := utils.Kubectl("create", "ns", namespace); err != nil {
		suite.FailNow("Error while creating namespace", err)
	}
	// deploy KDC and HDFS
	for _, resource := range hdfsKerberosDeployment {
		if _, err := utils.Kubectl("apply", "-f", fmt.Sprint(resourceFolder, "/", resource.path), "-n", namespace); err != nil {
			suite.FailNowf("Error while creating \"%s\"", resource.name, err)
		}
		if resource.wait {
			if _, err := utils.Kubectl("wait", fmt.Sprint("deployments/", resource.name),
				"--for=condition=available",
				fmt.Sprintf("--timeout=%v", waitTimeout),
				"-n", namespace); err != nil {
				suite.FailNowf("Error while waiting for resource \"%s\" to be deployed", resource.name, err)
			}
		}
	}
	// get the name of a Namenode pod
	nameNodePod, err := utils.Kubectl("get", "pods",
		"--selector=name=hdfs-namenode", "--output=jsonpath={.items[*].metadata.name}", "-n", namespace)
	if err != nil {
		suite.FailNow("Error while getting Namenode pod name", err)
	}
	// run init script to copy test data to HDFS and export delegation token
	if _, err := utils.Kubectl("exec", nameNodePod, "-n", namespace, "--", "init.sh"); err != nil {
		suite.FailNow("Error while running initialization script", err)
	}

	// copy delegation token from the pod to a local filesystem
	if _, err := utils.Kubectl("cp", fmt.Sprint(nameNodePod, ":", hadoopTokenPath[1:]),
		hadoopTokenPath, "-n", namespace); err != nil {
		suite.FailNow("Error while copying the delegation token", err)
	}
	if _, err := utils.Kubectl("create", "secret",
		"generic", hadoopTokenSecret, "--from-file", hadoopTokenPath, "-n", namespace); err != nil {
		suite.FailNow("Error while creating a secret for delegation token", err)
	}
}

// invoked before each test
func (suite *HdfsIntegrationSuite) BeforeTest(suiteName, testName string) {
	suite.operator = utils.SparkOperatorInstallation{
		Namespace:            namespace,
		SkipNamespaceCleanUp: true,
	}
	if testName == "Test_Spark_Hdfs_Kerberos_SparkHistoryServer" {
		operatorParams := map[string]string{
			"enableHistoryServer":         "true",
			"historyServerFsLogDirectory": eventLogDir,
			"delegationTokenSecret":       hadoopTokenSecret,
		}
		suite.operator.Params = operatorParams
	}
	if err := suite.operator.InstallSparkOperator(); err != nil {
		suite.FailNow(err.Error())
	}
}

func (suite *HdfsIntegrationSuite) AfterTest(suiteName, testName string) {
	// remove the operator and wait until all resources are deleted
	utils.Kubectl("delete", "instance.kudo.dev", suite.operator.InstanceName,
		"-n", suite.operator.Namespace, "--wait=true")
}

func (suite *HdfsIntegrationSuite) Test_Spark_Hdfs_Kerberos() {
	jobName := "spark-hdfs-kerberos"
	sparkJob := utils.SparkJob{
		Name:     jobName,
		Template: fmt.Sprintf("%s.yaml", jobName),
	}
	if err := suite.operator.SubmitJob(&sparkJob); err != nil {
		suite.FailNow(err.Error())
	}

	if err := suite.operator.WaitUntilSucceeded(sparkJob); err != nil {
		suite.FailNow(err.Error())
	}
}

func (suite *HdfsIntegrationSuite) Test_Spark_Hdfs_Kerberos_SparkHistoryServer() {
	jobName := "spark-hdfs-kerberos"
	sparkJob := utils.SparkJob{
		Name:     jobName,
		Template: fmt.Sprintf("%s.yaml", jobName),
		Params: map[string]interface{}{
			"SparkConf": map[string]string{
				"spark.eventLog.enabled": "true",
				"spark.eventLog.dir":     eventLogDir,
			},
		},
	}
	if err := suite.operator.SubmitJob(&sparkJob); err != nil {
		suite.FailNow(err.Error())
	}

	if err := suite.operator.WaitUntilSucceeded(sparkJob); err != nil {
		suite.FailNow(err.Error())
	}

	// check the logs to verify the app has been parsed by Spark History Server
	historyServerPodName, _ := utils.Kubectl("get", "pods", "--namespace", suite.operator.Namespace,
		"--selector=app.kubernetes.io/name=spark-history-server", "--output=jsonpath={.items[*].metadata.name}")

	logRecord := fmt.Sprintf("FsHistoryProvider: Finished parsing %s/spark-", eventLogDir)
	utils.Retry(func() error {
		contains, err := utils.PodLogContains(suite.operator.Namespace, historyServerPodName, logRecord)
		if err != nil {
			return err
		} else if !contains {
			return fmt.Errorf("text is not present in the logs, retrying")
		}
		return nil
	})
}

func (suite *HdfsIntegrationSuite) TearDownSuite() {
	utils.Kubectl("delete", "ns", namespace)
}
