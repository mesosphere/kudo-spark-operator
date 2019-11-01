package tests

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMain(m *testing.M) {
	utils.InstallKudo()
	defer utils.UninstallKudo()

	m.Run()
}

func TestSparkOperatorInstallation(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err.Error())
	}

	k8sNamespace, err := spark.K8sClients.CoreV1().Namespaces().Get(spark.Namespace, v1.GetOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	log.Infof("Spark operator is installed in namespace %s", k8sNamespace.Name)
}

func TestSparkOperatorInstallationWithCustomNamespace(t *testing.T) {
	customNamespace := "custom-test-namespace"
	spark := utils.SparkOperatorInstallation{
		Namespace: customNamespace,
	}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err.Error())
	}

	k8sNamespace, err := spark.K8sClients.CoreV1().Namespaces().Get(spark.Namespace, v1.GetOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}

	if k8sNamespace.Name != customNamespace {
		t.Errorf("Actual namespace is %s, while %s was expected", k8sNamespace.Name, customNamespace)
	}
}

func TestJobSubmission(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err)
	}

	job := utils.SparkJob{
		Name:     "linear-regression",
		Template: "spark-linear-regression-job.yaml",
	}

	err = spark.SubmitJob(&job)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = spark.WaitUntilSucceeded(job)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestSparkHistoryServerInstallation(t *testing.T) {
	awsAccessKey := utils.GetenvOr("AWS_ACCESS_KEY_ID", "")
	awsAccessSecret := utils.GetenvOr("AWS_SECRET_ACCESS_KEY", "")
	awsSessionToken := utils.GetenvOr("AWS_SESSION_TOKEN", "")
	awsBucketName := utils.GetenvOr("AWS_BUCKET_NAME", "infinity-artifacts-ci")
	awsFolderPath := "/autodelete7d/spark-operator-history-server-test/"

	// Make sure folder is deleted
	err := utils.AwsS3DeleteFolder(awsBucketName, awsFolderPath)
	if err != nil {
		t.Fatal(err.Error())
	}
	// Make sure folder is created
	err = utils.AwsS3CreateFolder(awsBucketName, awsFolderPath)
	if err != nil {
		t.Fatal(err.Error())
	}

	awsBucketPath := "s3a://" + awsBucketName + awsFolderPath

	historyParams := make(map[string]string)
	historyParams["enableHistoryServer"] = "true"
	historyParams["historyServerFsLogDirectory"] = awsBucketPath

	if len(awsSessionToken) > 0 {
		historyParams["historyServerOpts"] = "-Dspark.hadoop.fs.s3a.access.key=" + awsAccessKey +
			" -Dspark.hadoop.fs.s3a.secret.key=" + awsAccessSecret +
			" -Dspark.hadoop.fs.s3a.session.token=" + awsSessionToken +
			" -Dspark.hadoop.fs.s3a.aws.credentials.provider=org.apache.hadoop.fs.s3a.TemporaryAWSCredentialsProvider" +
			" -Dspark.hadoop.fs.s3a.impl=org.apache.hadoop.fs.s3a.S3AFileSystem"
	} else {
		historyParams["historyServerOpts"] = "-Dspark.hadoop.fs.s3a.access.key=" + awsAccessKey +
			" -Dspark.hadoop.fs.s3a.secret.key=" + awsAccessSecret +
			" -Dspark.hadoop.fs.s3a.impl=org.apache.hadoop.fs.s3a.S3AFileSystem"
	}

	spark := utils.SparkOperatorInstallation{
		Params: historyParams,
	}
	err = spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err.Error())
	}

	awsParams := map[string]interface{}{
		"AwsBucketPath":   awsBucketPath,
		"AwsAccessKey":    awsAccessKey,
		"AwsAccessSecret": awsAccessSecret,
		"AwsSessionToken": awsSessionToken,
	}
	job := utils.SparkJob{
		Name:     "history-server-linear-regression",
		Params:   awsParams,
		Template: "spark-linear-regression-history-server-job.yaml",
	}

	// Submit a SparkApplication
	err = spark.SubmitJob(&job)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = spark.WaitUntilSucceeded(job)
	if err != nil {
		t.Error(err.Error())
	}

	// Find out History Server POD name
	historyServerPodName, err := utils.Kubectl(
		"get",
		"pods",
		"--namespace="+spark.Namespace,
		"--output=jsonpath={.items[?(@.metadata.labels.app\\.kubernetes\\.io/name==\"history-server\")].metadata.name}",
	)
	if err != nil {
		t.Error(err.Error())
	}

	// Find out the Job ID for the submitted SparkApplication
	jobID, err := utils.Kubectl(
		"get",
		"pods",
		"--namespace="+spark.Namespace,
		"--output=jsonpath={.items[*].metadata.labels.spark-app-selector}",
	)
	if err != nil {
		t.Error(err.Error())
	}

	// Get an application detail from History Server
	err = utils.RetryWithTimeout(2*time.Minute, 5*time.Second, func() error {
		historyServerResponse, err := utils.Kubectl(
			"exec",
			historyServerPodName,
			"--namespace="+spark.Namespace,
			"--",
			"/usr/bin/curl",
			"http://localhost:18080/api/v1/applications/"+jobID+"/jobs",
		)
		if err != nil {
			return err
		}

		if len(historyServerResponse) > 0 &&
			!strings.Contains(historyServerResponse, "no such app") {
			log.Infof("Job Id '%s' is successfully recorded in History Server", jobID)
			return nil
		} else {
			return errors.New("Expecting Job Id '" + jobID + "' to be recorded in History Server")
		}
	})

	if err != nil {
		t.Errorf("The Job Id '%s' haven't appeared in History Server", jobID)
	}
	utils.AwsS3DeleteFolder(awsBucketName, awsFolderPath)
}
