package tests

import (
	"fmt"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

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
	utils.Kubectl("create", "ns", namespace)
	// deploy KDC and HDFS
	for _, resource := range hdfsKerberosDeployment {
		utils.Kubectl("apply", "-f", fmt.Sprint(resourceFolder, "/", resource.path), "-n", namespace)

		if resource.wait {
			utils.Kubectl("wait", fmt.Sprint("deployments/", resource.name),
				"--for=condition=available", "--timeout=3m", "-n", namespace)
		}
	}
	nameNodePod, err := utils.Kubectl("get", "pods", "--selector=name=hdfs-namenode",
		"--output=jsonpath={.items[*].metadata.name}", "-n", namespace)
	if err != nil {
		suite.FailNow("Error while getting a name of Namenode pod", err)
	}
	// run init script to copy test data to HDFS and export delegation token
	utils.Kubectl("exec", nameNodePod, "-n", namespace, "--", "init.sh")

	// copy delegation token from the pod to a local filesystem
	utils.Kubectl("cp", fmt.Sprint(nameNodePod, ":", "tmp/hadoop.token"), "/tmp/hadoop.token", "-n", namespace)
	utils.Kubectl("create", "secret", "generic", hadoopTokenSecret, "--from-file", "/tmp/hadoop.token", "-n", namespace)
}

// invoked before each test
func (suite *HdfsIntegrationSuite) BeforeTest(suiteName, testName string) {
	fmt.Println(suiteName, testName)

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
	utils.Kubectl("kudo", "uninstall",
		"--instance", suite.operator.InstanceName,
		"--namespace", suite.operator.Namespace)
	// TODO: temporary workaround: explicitly waiting for uninstall to clean all the resources before the next test starts.
	time.Sleep(15 * time.Second)
}

func (suite *HdfsIntegrationSuite) Test_Spark_Hdfs_Kerberos() {
	jobName := "spark-hdfs-kerberos"
	sparkPi := utils.SparkJob{
		Name:     jobName,
		Template: fmt.Sprintf("%s.yaml", jobName),
	}
	if err := suite.operator.SubmitJob(&sparkPi); err != nil {
		suite.FailNow(err.Error())
	}

	if err := suite.operator.WaitUntilSucceeded(sparkPi); err != nil {
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
	// TODO: verify the job appears in SHS logs
}

func (suite *HdfsIntegrationSuite) TearDownSuite() {
	utils.Kubectl("delete", "ns", namespace)
}
