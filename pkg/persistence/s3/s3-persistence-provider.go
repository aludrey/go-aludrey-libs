package s3pp

import (
	"io"
	"os"
	"path/filepath"

	"github.com/aludrey/go-aludrey-libs/pkg/log"
	"github.com/aludrey/go-aludrey-libs/pkg/persistence"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type s3PersistenceProvider struct {
	s3Downloader *s3manager.Downloader
	s3Client     *s3.S3
	region       string
}

func NewS3PersistenceProvider(region string) persistence.PersistenceProvider {
	return &s3PersistenceProvider{
		region: region,
	}
}

func (p *s3PersistenceProvider) createS3Client(logger log.Logger) error {
	s3Session, err := session.NewSession(&aws.Config{
		Region: aws.String(p.region),
	})
	if err != nil {
		logger.Error("Error creating session: %v", err)
		return err
	}

	p.s3Downloader = s3manager.NewDownloader(s3Session)
	p.s3Client = s3.New(s3Session)

	return nil
}

func (p *s3PersistenceProvider) validateInit(logger log.Logger) error {
	if p.s3Downloader == nil || p.s3Client == nil {
		return p.createS3Client(logger)
	}

	return nil
}

func (p *s3PersistenceProvider) DownloadFile(logger log.Logger, bucketName string, itemFile string) (*os.File, error) {
	err := p.validateInit(logger)
	if err != nil {
		logger.Error("Error validating init: %v", err)
		return nil, err
	}

	filename := filepath.Base(itemFile)

	file, err := os.Create(filename)
	if err != nil {
		return file, err
	}

	_, err = p.s3Downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(itemFile),
	})

	if err != nil {
		logger.Error("Unable to download item %q, %v", itemFile, err)
		return file, err
	}

	return file, nil
}

func (p *s3PersistenceProvider) DeleteFile(logger log.Logger, bucketName string, itemFile string) error {
	err := p.validateInit(logger)
	if err != nil {
		logger.Error("Error validating init: %v", err)
	}

	_, err = p.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(itemFile),
	})

	if err != nil {
		logger.Error("Unable to delete item %q, %v", itemFile, err)
		return err
	}

	return nil
}

func (p *s3PersistenceProvider) UploadLocalFile(logger log.Logger, bucketName string, fileKey string, localFilePath string) error {
	err := p.validateInit(logger)
	if err != nil {
		logger.Error("Error validating init: %v", err)
		return err
	}

	file, err := os.Open(localFilePath)

	if err != nil {
		logger.Error("Unable to open file %q, %v", localFilePath, err)
		return err
	}

	defer file.Close()
	return p.UploadStream(logger, bucketName, fileKey, file)
}

func (p *s3PersistenceProvider) UploadStream(logger log.Logger, bucketName string, fileKey string, streamReader io.ReadSeeker) error {
	err := p.validateInit(logger)
	if err != nil {
		logger.Error("Error validating init: %v", err)
		return err
	}

	_, err = p.s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileKey),
		Body:   streamReader,
	})

	if err != nil {
		logger.Error("Unable to upload item %q, %v", fileKey, err)
		return err
	}

	return nil
}

func (p *s3PersistenceProvider) ListFiles(logger log.Logger, bucketName string, prefix string) ([]string, error) {
	err := p.validateInit(logger)
	if err != nil {
		logger.Error("Error validating init: %v", err)
		return nil, err
	}

	objects, err := p.s3Client.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		logger.Error("Unable to list items in bucket %q, %v", bucketName, err)
		return nil, err
	}

	var aux []string

	for _, item := range objects.Contents {
		aux = append(aux, *item.Key)
	}

	return aux, nil
}

func (p *s3PersistenceProvider) MoveFile(logger log.Logger, bucketName string, itemFile string, destBucketName string, destItemFile string) error {
	err := p.CopyFile(logger, bucketName, itemFile, destBucketName, destItemFile)
	if err != nil {
		return err
	}

	err = p.DeleteFile(logger, bucketName, itemFile)
	if err != nil {
		return err
	}

	return nil
}

func (p *s3PersistenceProvider) CopyFile(logger log.Logger, souceBucketName string, sourceFileKe string, destBucketName string, destFileKey string) error {
	err := p.validateInit(logger)
	if err != nil {
		logger.Error("Error validating init: %v", err)
		return err
	}

	_, err = p.s3Client.CopyObject(&s3.CopyObjectInput{
		Bucket:     aws.String(destBucketName),
		CopySource: aws.String(souceBucketName + "/" + sourceFileKe),
		Key:        aws.String(destFileKey),
	})

	if err != nil {
		logger.Error("Unable to copy item %q, %v", sourceFileKe, err)
		return err
	}

	return nil
}
