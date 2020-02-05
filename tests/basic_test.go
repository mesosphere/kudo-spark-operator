package tests

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
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
	awsBucketName, err := utils.GetS3BucketName()
	if err != nil {
		t.Fatal(err)
	}
	awsFolderPath, err := utils.GetS3BucketPath()
	if err != nil {
		t.Fatal(err)
	}

	awsFolderPath = fmt.Sprintf("%s/%s/%s", awsFolderPath, "spark-history-server", uuid.New().String())
	// Make sure folder is created
	if err := utils.AwsS3CreateFolder(awsBucketName, awsFolderPath); err != nil {
		t.Fatal(err.Error())
	}
	defer utils.AwsS3DeleteFolder(awsBucketName, awsFolderPath)

	awsBucketPath := "s3a://" + awsBucketName + "/" + awsFolderPath

	awsCredentials, err := utils.GetAwsCredentials()
	if err != nil {
		t.Fatal(err)
	}

	clientSet, err := utils.GetK8sClientSet()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := utils.CreateNamespace(clientSet, utils.DefaultNamespace); err != nil {
		t.Fatal(err)
	}

	// create a Secret with Spark configuration holding AWS credentials
	// which will be used by Spark History Server to authenticate with S3
	var sparkConf = strings.Join(
		[]string{
			fmt.Sprintf("spark.hadoop.fs.s3a.access.key %s", awsCredentials[utils.AwsAccessKeyId]),
			fmt.Sprintf("spark.hadoop.fs.s3a.secret.key %s", awsCredentials[utils.AwsSecretAccessKey]),
			fmt.Sprintf("spark.hadoop.fs.s3a.session.token %s", awsCredentials[utils.AwsSessionToken]),
		},
		"\n",
	)

	sparkConfSecretName := "spark-conf"
	sparkConfSecretKey := "spark-defaults.conf"
	sparkConfSecretData := map[string][]byte{
		sparkConfSecretKey: []byte(sparkConf),
	}

	if err := utils.CreateSecretEncoded(clientSet, sparkConfSecretName, utils.DefaultNamespace, sparkConfSecretData); err != nil {
		t.Fatal("Error while creating a Secret", err)
	}

	// configure Spark Operator parameters
	operatorParams := map[string]string{
		"enableHistoryServer":          "true",
		"historyServerFsLogDirectory":  awsBucketPath,
		"historyServerOpts":            "-Dspark.hadoop.fs.s3a.impl=org.apache.hadoop.fs.s3a.S3AFileSystem",
		"historyServerSparkConfSecret": sparkConfSecretName,
	}

	// in case we are using temporary security credentials
	if len(string(awsCredentials[utils.AwsSessionToken])) > 0 {
		operatorParams["historyServerOpts"] =
			strings.Join(
				[]string{
					operatorParams["historyServerOpts"],
					"-Dspark.hadoop.fs.s3a.aws.credentials.provider=org.apache.hadoop.fs.s3a.TemporaryAWSCredentialsProvider",
				},
				" ",
			)

	}

	spark := utils.SparkOperatorInstallation{
		SkipNamespaceCleanUp: true,
		Params:               operatorParams,
	}

	if err := spark.InstallSparkOperator(); err != nil {
		t.Fatal(err.Error())
	}
	defer spark.CleanUp()

	// create a Secret for SparkApplication
	if err := utils.CreateSecretEncoded(clientSet, utils.DefaultAwsSecretName, utils.DefaultNamespace, awsCredentials); err != nil {
		t.Fatal("Error while creating a Secret", err)
	}

	sparkAppParams := map[string]interface{}{
		"AwsBucketPath": awsBucketPath,
		"AwsSecretName": utils.DefaultAwsSecretName,
	}

	if _, isPresent := awsCredentials[utils.AwsSessionToken]; isPresent {
		sparkAppParams["AwsSessionToken"] = "true"
	}

	job := utils.SparkJob{
		Name:     "history-server-linear-regression",
		Params:   sparkAppParams,
		Template: "spark-linear-regression-history-server-job.yaml",
	}

	// Submit a SparkApplication
	if err := spark.SubmitJob(&job); err != nil {
		t.Fatal(err.Error())
	}

	if err := spark.WaitUntilSucceeded(job); err != nil {
		t.Error(err.Error())
	}

	// Find out History Server POD name
	instanceName := fmt.Sprint(utils.OperatorName, "-history-server")
	historyServerPodName, err := utils.Kubectl("get", "pods",
		fmt.Sprintf("--namespace=%s", spark.Namespace),
		"--field-selector=status.phase=Running",
		fmt.Sprintf("--selector=app.kubernetes.io/name=%s", instanceName),
		"--output=jsonpath={.items[*].metadata.name}")
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
	err = utils.RetryWithTimeout(utils.DefaultRetryTimeout, utils.DefaultRetryInterval, func() error {
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
		log.Infof("Spark History Server logs:")
		utils.Kubectl("logs", "-n", spark.Namespace, historyServerPodName)
		log.Info("Driver logs:")
		utils.Kubectl("logs", "-n", spark.Namespace, utils.DriverPodName(job.Name))
	}
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

	err = utils.RetryWithTimeout(utils.DefaultRetryTimeout, utils.DefaultRetryInterval, func() error {
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
