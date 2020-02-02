package tests

import (
	"fmt"
	. "github.com/google/uuid"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/suite"
	"testing"
)

const sparkApplicationName = "spark-s3-readwrite"
const s3FileName = "README.md"

type S3TestSuite struct {
	operator utils.SparkOperatorInstallation
	suite.Suite
	awsCredentials map[string][]byte
	s3BucketName   string
	s3BucketPath   string
}

func TestS3Suite(t *testing.T) {
	suite.Run(t, new(S3TestSuite))
}

func (suite *S3TestSuite) SetupSuite() {
	installOperator(suite)
	createAwsCredentialsSecret(suite)
	if s3BucketName, err := utils.GetS3BucketName(); err != nil {
		suite.FailNow("Failed to setup suite", err)
	} else {
		suite.s3BucketName = s3BucketName
	}
	if s3BucketPath, err := utils.GetS3BucketPath(); err != nil {
		suite.FailNow("Failed to setup suite", err)
	} else {
		suite.s3BucketPath = s3BucketPath
	}
}

func (suite *S3TestSuite) TearDownSuite() {
	suite.operator.CleanUp()
}

func installOperator(suite *S3TestSuite) {
	suite.operator = utils.SparkOperatorInstallation{}

	if err := suite.operator.InstallSparkOperator(); err != nil {
		suite.FailNow(err.Error())
	}
}

func createAwsCredentialsSecret(suite *S3TestSuite) {
	awsCredentials, err := utils.GetAwsCredentials()
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.awsCredentials = awsCredentials
	if err := utils.CreateSecretEncoded(
		suite.operator.K8sClients,
		utils.DefaultAwsSecretName,
		suite.operator.Namespace,
		awsCredentials,
	); err != nil {
		suite.FailNow("Error while creating a Secret", err)
	}
}

// test verifies read/write access to S3 bucket using EnvironmentVariableCredentialsProvider for authentication
// by reading a file from S3 bucket, counting the lines and writing the result to another S3 location.
// AWS environment variables are propagated to a driver and executors via a Secret object
func (suite *S3TestSuite) TestS3ReadWriteAccess() {

	testFolder := fmt.Sprint("s3a://", suite.s3BucketName, "/", suite.s3BucketPath, "/", sparkApplicationName)
	fileFolder := fmt.Sprint(testFolder, "/", s3FileName)
	uuid := New().String()
	writeToFolder := fmt.Sprint(testFolder, "/", uuid)

	params := map[string]interface{}{
		// the name of a Secret with AWS credentials
		"AwsSecretName": utils.DefaultAwsSecretName,
		"ReadUrl":       fileFolder,
		"WriteUrl":      writeToFolder,
	}
	if _, present := suite.awsCredentials[utils.AwsSessionToken]; present {
		params["AwsSessionToken"] = "true"
	}

	sparkS3App := utils.SparkJob{
		Name:     sparkApplicationName,
		Template: fmt.Sprintf("%s.yaml", sparkApplicationName),
		Params:   params,
	}

	if err := suite.operator.SubmitJob(&sparkS3App); err != nil {
		suite.FailNow("Error submitting SparkApplication", err)
	}

	if err := suite.operator.WaitUntilSucceeded(sparkS3App); err != nil {
		driverLog, _ := suite.operator.DriverLog(sparkS3App)
		log.Info("Driver logs:\n", driverLog)
		suite.FailNow(err.Error())
	}

	if driverLogContains, err := suite.operator.DriverLogContains(sparkS3App, "Wrote 105 lines"); err != nil {
		log.Warn("Error while getting pod logs", err)
	} else {
		suite.True(driverLogContains)
	}

	// clean up S3 folder
	if err := utils.AwsS3DeleteFolder(suite.s3BucketName, fmt.Sprint(testFolder, "/", uuid)); err != nil {
		log.Warn(err)
	}
}
