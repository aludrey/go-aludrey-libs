package bucket

import (
	"bytes"
	"context"
	"log"
	os "os"
	"path/filepath"

	"github.com/aludrey/go-aludrey-libs/pkg/commons"
	"github.com/aludrey/go-aludrey-libs/pkg/entity"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	s3Downloader *s3manager.Downloader
	s3Client     *s3.S3
	s3Session    *session.Session
	config       *entity.Config = commons.GetConfig()
)

func createS3Client() error {
	s3Session, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AwsRegion),
	})
	if err != nil {
		log.Printf("Error creating session: %v", err)
		return err
	}

	s3Downloader = s3manager.NewDownloader(s3Session)
	s3Client = s3.New(s3Session)

	return nil
}

func validateInit() error {
	if s3Session == nil || s3Downloader == nil || s3Client == nil {
		// We want the library to be backwards compatible but we also want to inform the user that the library has been replaced
		log.Printf("Bucket library was replaced with the generic persistence library. Please use the persistence library instead.")
		return createS3Client()
	}

	return nil
}

func DownloadFile(ctx context.Context, bucketName string, itemFile string) (*os.File, error) {
	err := validateInit()
	if err != nil {
		log.Printf("Error validating init: %v", err)
		return nil, err
	}

	filename := filepath.Base(itemFile)

	file, err := os.Create(filename)
	if err != nil {
		return file, err
	}

	_, err = s3Downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(itemFile),
	})

	if err != nil {
		log.Printf("Unable to download item %q, %v", itemFile, err)
		return file, err
	}

	return file, nil
}

func DeleteFile(ctx context.Context, bucketName string, itemFile string) error {
	err := validateInit()
	if err != nil {
		log.Printf("Error validating init: %v", err)
	}

	_, err = s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(itemFile),
	})

	if err != nil {
		log.Printf("Unable to delete item %q, %v", itemFile, err)
		return err
	}

	return nil
}

func UploadFile(ctx context.Context, bucketName string, localItem string, itemFile string) error {
	err := validateInit()
	if err != nil {
		log.Printf("Error validating init: %v", err)
		return err
	}

	file, err := os.Open(localItem)

	if err != nil {
		log.Printf("Unable to open file %q, %v", localItem, err)
		return err
	}

	defer file.Close()

	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)

	file.Read(buffer)

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(itemFile),
		Body:   bytes.NewReader(buffer),
	})

	if err != nil {
		log.Printf("Unable to upload item %q, %v", localItem, err)
		return err
	}

	return nil
}

func ListFiles(ctx context.Context, bucketName string, prefix string) ([]string, error) {
	err := validateInit()
	if err != nil {
		log.Printf("Error validating init: %v", err)
		return nil, err
	}

	objects, err := s3Client.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		log.Printf("Unable to list items in bucket %q, %v", bucketName, err)
		return nil, err
	}

	var aux []string

	for _, item := range objects.Contents {
		aux = append(aux, *item.Key)
	}

	return aux, nil
}

func MoveFile(ctx context.Context, bucketName string, itemFile string, destBucketName string, destItemFile string) error {
	err := CopyFile(ctx, bucketName, itemFile, destBucketName, destItemFile)
	if err != nil {
		return err
	}

	err = DeleteFile(ctx, bucketName, itemFile)
	if err != nil {
		return err
	}

	return nil
}

func CopyFile(ctx context.Context, bucketName string, itemFile string, destBucketName string, destItemFile string) error {
	err := validateInit()
	if err != nil {
		log.Printf("Error validating init: %v", err)
		return err
	}

	_, err = s3Client.CopyObject(&s3.CopyObjectInput{
		Bucket:     aws.String(destBucketName),
		CopySource: aws.String(bucketName + "/" + itemFile),
		Key:        aws.String(destItemFile),
	})

	if err != nil {
		log.Printf("Unable to copy item %q, %v", itemFile, err)
		return err
	}

	return nil
}
