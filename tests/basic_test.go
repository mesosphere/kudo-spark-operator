package tests

import (
	"fmt"
	"os"
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
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsAccessSecret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsSessionToken := os.Getenv("AWS_SESSION_TOKEN")
	awsBucketName, present := os.LookupEnv("AWS_BUCKET_NAME")
	if !present {
		t.Fatal("AWS_BUCKET_NAME is not configured")
	}
	awsFolderPath, present := os.LookupEnv("AWS_BUCKET_PATH")
	if !present {
		t.Fatal("AWS_BUCKET_PATH is not configured")
	}

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

	awsBucketPath := "s3a://" + awsBucketName + "/" + awsFolderPath

	historyParams := make(map[string]string)
	historyParams["enableHistoryServer"] = "true"
	historyParams["historyServerFsLogDirectory"] = awsBucketPath

	historyParams["historyServerOpts"] = "-Dspark.hadoop.fs.s3a.access.key=" + awsAccessKey +
		" -Dspark.hadoop.fs.s3a.secret.key=" + awsAccessSecret +
		" -Dspark.hadoop.fs.s3a.impl=org.apache.hadoop.fs.s3a.S3AFileSystem"

	if len(awsSessionToken) > 0 {
		historyParams["historyServerOpts"] = historyParams["historyServerOpts"] +
			" -Dspark.hadoop.fs.s3a.session.token=" + awsSessionToken +
			" -Dspark.hadoop.fs.s3a.aws.credentials.provider=org.apache.hadoop.fs.s3a.TemporaryAWSCredentialsProvider"
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
	instanceName := fmt.Sprint(utils.OperatorName, "-history-server")
	historyServerPodName, err := utils.Kubectl(
		"get",
		"pods",
		"--namespace="+spark.Namespace,
		"--output=jsonpath={.items[?(@.metadata.labels.app\\.kubernetes\\.io/name==\""+instanceName+"\")].metadata.name}",
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
		}
		return fmt.Errorf("Expecting Job Id '%s' to be recorded in History Server", jobID)
	})

	if err != nil {
		t.Errorf("The Job Id '%s' haven't appeared in History Server", jobID)
	}
	utils.AwsS3DeleteFolder(awsBucketName, awsFolderPath)
}

func TestVolumeMounts(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	err := spark.InstallSparkOperator()
	defer spark.CleanUp()

	if err != nil {
		t.Fatal(err)
	}

	jobName := "mock-task-runner"
	volumeName := "test-volume"
	mountPath := "/opt/spark/work-dir"
	job := utils.SparkJob{
		Name:     jobName,
		Template: "spark-mock-task-runner-job.yaml",
		Params: map[string]interface{}{
			"args":       []string{"1", "60"},
			"VolumeName": volumeName,
			"MountPath":  mountPath,
		},
	}

	err = spark.SubmitJob(&job)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = utils.RetryWithTimeout(2*time.Minute, 5*time.Second, func() error {
		lsCmdResponse, err := utils.Kubectl(
			"exec",
			utils.DriverPodName(jobName),
			"--namespace="+spark.Namespace,
			"--",
			"ls",
			"-ltr",
			mountPath+"/tmp",
		)
		if err != nil {
			return err
		}

		if len(lsCmdResponse) > 0 &&
			strings.Contains(lsCmdResponse, "spark") {
			log.Infof("Successfully mounted '%s' and volume is writable", volumeName)
			return nil
		}
		return fmt.Errorf("Expecting '%s' to be mounted", volumeName)
	})

	if err != nil {
		t.Errorf("Unable to mount volume '%s'", volumeName)
	}

	err = spark.WaitUntilSucceeded(job)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestPythonSupport(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	if err := spark.InstallSparkOperator(); err != nil {
		t.Fatal(err)
	}
	defer spark.CleanUp()

	jobName := "spark-pi-python"
	job := utils.SparkJob{
		Name:     jobName,
		Template: fmt.Sprintf("%s.yaml", jobName),
	}

	if err := spark.SubmitJob(&job); err != nil {
		t.Fatal(err)
	}

	if err := spark.WaitForOutput(job, "Pi is roughly 3.1"); err != nil {
		t.Fatal(err)
	}
}

func TestRSupport(t *testing.T) {
	spark := utils.SparkOperatorInstallation{}
	if err := spark.InstallSparkOperator(); err != nil {
		t.Fatal(err)
	}
	defer spark.CleanUp()

	jobName := "spark-r-als"
	job := utils.SparkJob{
		Name:           jobName,
		Template:       fmt.Sprintf("%s.yaml", jobName),
		ExecutorsCount: 3,
	}

	if err := spark.SubmitJob(&job); err != nil {
		t.Fatal(err)
	}

	if err := spark.WaitForOutput(job, "3   2.997274"); err != nil {
		t.Fatal(err)
	}

	if err := spark.WaitUntilSucceeded(job); err != nil {
		t.Fatal(err)
	}
}
