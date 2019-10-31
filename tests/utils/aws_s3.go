package utils

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	log "github.com/sirupsen/logrus"
)

func AwsS3CreateFolder(bucketName string, folderPath string) bool {
	file := strings.NewReader("")
	// Make sure escape('/') is at both ends of the path
	folderPath = "/" + strings.TrimPrefix(folderPath, "/")
	folderPath = strings.TrimSuffix(folderPath, "/") + "/"
	filePath := folderPath + ".tmp"

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})

	// Setup the S3 Upload Manager.
	uploader := s3manager.NewUploader(sess)

	// Upload the demo file to S3 bucket which will create a folder
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filePath),
		Body:   file,
	})
	if err != nil {
		log.Errorf("Unable to create folder '%s' in bucket '%s', %v", folderPath, bucketName, err)
		return false
	}
	log.Infof("Successfully created folder '%s' in bucket '%s'", folderPath, bucketName)
	return true
}

func AwsS3DeleteFolder(bucketName string, folderPath string) bool {
	// Make sure escape('/') is at only right end of the path
	folderPath = strings.TrimPrefix(folderPath, "/")
	folderPath = strings.TrimSuffix(folderPath, "/") + "/"

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})

	// Create S3 Service Client
	svcClient := s3.New(sess)

	// Fetch all the items present in the given folder
	response, err := svcClient.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(folderPath),
	})
	if err != nil {
		log.Errorf("Unable to list items in bucket '%s' at key '%s', %v", bucketName, folderPath, err)
		return false
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
			return false
		}

		err = svcClient.WaitUntilObjectNotExists(&s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(itemKey),
		})

		if err != nil {
			log.Errorf("Error occurred while waiting for object '%s' to be deleted, %v", itemKey, err)
			return false
		}

		log.Infof("Object '%s' successfully deleted\n", itemKey)
	}
	return true
}
