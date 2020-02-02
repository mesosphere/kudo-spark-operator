package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	log "github.com/sirupsen/logrus"
)

const AwsAccessKeyId = "AWS_ACCESS_KEY_ID"
const AwsSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
const AwsSessionToken = "AWS_SESSION_TOKEN"
const AwsBucketName = "AWS_BUCKET_NAME"
const AwsBucketPath = "AWS_BUCKET_PATH"

// this method looks up for the following AWS environment variables:
//
// - AWS_ACCESS_KEY_ID (required)
// - AWS_SECRET_ACCESS_KEY (required)
// - AWS_SESSION_TOKEN (optional)
//
// returns a map with env variable as a key with a value converted to a []byte.
// the error is returned if one of the required variables is not set.
func GetAwsCredentials() (map[string][]byte, error) {
	awsAccessKeyId, err := checkEnvVar(AwsAccessKeyId)
	if err != nil {
		return nil, err
	}

	awsSecretAccessKey, err := checkEnvVar(AwsSecretAccessKey)
	if err != nil {
		return nil, err
	}

	awsEnvVars := map[string][]byte{
		AwsAccessKeyId:     []byte(awsAccessKeyId),
		AwsSecretAccessKey: []byte(awsSecretAccessKey),
	}

	// support for Temporary Security Credentials
	if awsSessionToken, isPresent := os.LookupEnv(AwsSessionToken); isPresent && len(awsSecretAccessKey) > 0 {
		awsEnvVars[AwsSessionToken] = []byte(awsSessionToken)
	}
	return awsEnvVars, nil
}

// method returns the name of S3 bucket from AWS_BUCKET_NAME env variable
func GetS3BucketName() (string, error) {
	return checkEnvVar(AwsBucketName)
}

// method returns the S3 bucket path from AWS_BUCKET_PATH env variable
func GetS3BucketPath() (string, error) {
	return checkEnvVar(AwsBucketPath)
}

// method checks env variable and returns its value
// returns error if variable is not set or empty
func checkEnvVar(varName string) (string, error) {
	value, isPresent := os.LookupEnv(varName)
	if !isPresent || len(value) == 0 {
		return "", fmt.Errorf("%s env variable is not set", varName)
	}
	return value, nil
}

// AwsS3CreateFolder will create a object in bucketName with folderPath as Key
func AwsS3CreateFolder(bucketName string, folderPath string) error {
	file := strings.NewReader("")
	folderPath = strings.Trim(folderPath, "/")
	filePath := fmt.Sprintf("/%s/.tmp", folderPath)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})

	if err != nil {
		return err
	}

	// Create S3 Service Client
	svcClient := s3.New(sess)

	// Put a tmp file to S3 bucket
	_, err = svcClient.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filePath),
		Body:   file,
	})
	if err != nil {
		log.Errorf("Unable to put Key '%s' in bucket '%s', %v", filePath, bucketName, err)
		return err
	}
	log.Infof("Successfully created folder '%s' in bucket '%s'", folderPath, bucketName)
	return nil
}

// AwsS3DeleteFolder deletes all objects at folderPath in bucketName
func AwsS3DeleteFolder(bucketName string, folderPath string) error {
	folderPath = strings.Trim(folderPath, "/")
	folderPath = folderPath + "/"

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})

	if err != nil {
		return err
	}

	// Create S3 Service Client
	svcClient := s3.New(sess)

	// Fetch all the items present in the given folder
	response, err := svcClient.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(folderPath),
	})
	if err != nil {
		log.Errorf("Unable to list items in bucket '%s' at key '%s', %v", bucketName, folderPath, err)
		return err
	}

	for _, item := range response.Contents {
		// Delete an item
		itemKey := "/" + *item.Key
		_, err = svcClient.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(itemKey),
		})

		if err != nil {
			log.Errorf("Unable to delete key '%s' in bucket '%s', %v", itemKey, bucketName, err)
			return err
		}

		err = svcClient.WaitUntilObjectNotExists(&s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(itemKey),
		})

		if err != nil {
			log.Errorf("Error occurred while waiting for object '%s' to be deleted, %v", itemKey, err)
			return err
		}

		log.Infof("Object '%s' successfully deleted\n", itemKey)
	}
	return nil
}
