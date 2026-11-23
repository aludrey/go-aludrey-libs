package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	persistence "github.com/aludrey/go-aludrey-libs/pkg/persistence-v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

func (p *s3PersistenceProvider) createS3Client() error {
	s3Session, err := session.NewSession(&aws.Config{
		Region: aws.String(p.region),
	})
	if err != nil {
		return err
	}

	p.s3Downloader = s3manager.NewDownloader(s3Session)
	p.s3Client = s3.New(s3Session)

	return nil
}

func (p *s3PersistenceProvider) validateInit() error {
	if p.s3Downloader == nil || p.s3Client == nil {
		return p.createS3Client()
	}

	return nil
}

func (p *s3PersistenceProvider) DownloadFile(bucketName string, itemFile string) (*os.File, error) {
	err := p.validateInit()
	if err != nil {
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
		return file, err
	}

	return file, nil
}

func (p *s3PersistenceProvider) DeleteFile(bucketName string, itemFile string) error {
	err := p.validateInit()
	if err != nil {
		return err
	}

	_, err = p.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(itemFile),
	})

	if err != nil {
		return err
	}

	return nil
}

func (p *s3PersistenceProvider) UploadLocalFile(bucketName string, fileKey string, localFilePath string) error {
	err := p.validateInit()
	if err != nil {
		return err
	}

	file, err := os.Open(localFilePath)

	if err != nil {
		return err
	}

	defer file.Close()
	return p.UploadStream(bucketName, fileKey, file)
}

func (p *s3PersistenceProvider) UploadStream(bucketName string, fileKey string, streamReader io.ReadSeeker) error {
	err := p.validateInit()
	if err != nil {
		return err
	}

	_, err = p.s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileKey),
		Body:   streamReader,
	})

	if err != nil {
		return err
	}

	return nil
}

func (p *s3PersistenceProvider) ListFiles(bucketName string, prefix string) ([]string, error) {
	objects, err := p.ListObjects(bucketName, prefix)
	if err != nil {
		return nil, err
	}

	var aux []string

	for _, item := range objects {
		aux = append(aux, item.Key)
	}

	return aux, nil
}

func (p *s3PersistenceProvider) ListObjects(bucketName string, prefix string) ([]persistence.ObjectInfo, error) {
	if err := persistence.ValidateRepoName(bucketName); err != nil {
		return nil, err
	}

	err := p.validateInit()
	if err != nil {
		return nil, err
	}

	objects := make([]persistence.ObjectInfo, 0)
	err = p.s3Client.ListObjectsV2PagesWithContext(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	}, func(page *s3.ListObjectsV2Output, _ bool) bool {
		for _, item := range page.Contents {
			objects = append(objects, persistence.ObjectInfo{
				Key:          aws.StringValue(item.Key),
				Size:         aws.Int64Value(item.Size),
				LastModified: aws.TimeValue(item.LastModified),
				// S3 devuelve el ETag entre comillas; Cloud Storage no.
				ETag: strings.Trim(aws.StringValue(item.ETag), `"`),
			})
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("listing s3://%s/%s: %w", bucketName, prefix, err)
	}

	return objects, nil
}

func (p *s3PersistenceProvider) ReadFile(bucketName string, itemFile string, maxBytes int64) ([]byte, error) {
	if err := persistence.ValidateReadFile(bucketName, itemFile, maxBytes); err != nil {
		return nil, err
	}

	err := p.validateInit()
	if err != nil {
		return nil, err
	}

	location := "s3://" + bucketName + "/" + itemFile
	output, err := p.s3Client.GetObjectWithContext(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(itemFile),
	})
	if isS3NotFound(err) {
		return nil, fmt.Errorf("%w: %s: %w", persistence.ErrObjectNotFound, location, err)
	}
	if err != nil {
		return nil, fmt.Errorf("getting %s: %w", location, err)
	}

	return persistence.ReadAllLimited(output.Body, location, maxBytes)
}

// Con credenciales temporales (rol, SSO) la URL deja de funcionar cuando
// éstas expiran, aunque no se haya cumplido ttl.
func (p *s3PersistenceProvider) PresignDownloadURL(bucketName string, itemFile string, ttl time.Duration) (string, time.Time, error) {
	if err := persistence.ValidatePresign(bucketName, itemFile, ttl); err != nil {
		return "", time.Time{}, err
	}

	err := p.validateInit()
	if err != nil {
		return "", time.Time{}, err
	}

	req, _ := p.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(itemFile),
	})

	url, err := req.Presign(ttl)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("presigning s3://%s/%s: %w", bucketName, itemFile, err)
	}

	expiresAt, err := persistence.SignedURLExpiry(url, "X-Amz-Date", "X-Amz-Expires")
	if err != nil {
		return "", time.Time{}, fmt.Errorf("presigning s3://%s/%s: %w", bucketName, itemFile, err)
	}

	return url, expiresAt, nil
}

// GetObject responde NoSuchKey; las operaciones sin cuerpo (HeadObject), NotFound.
func isS3NotFound(err error) bool {
	var awsErr awserr.Error
	if !errors.As(err, &awsErr) {
		return false
	}
	return awsErr.Code() == s3.ErrCodeNoSuchKey || awsErr.Code() == "NotFound"
}

func (p *s3PersistenceProvider) MoveFile(bucketName string, itemFile string, destBucketName string, destItemFile string) error {
	err := p.CopyFile(bucketName, itemFile, destBucketName, destItemFile)
	if err != nil {
		return err
	}

	err = p.DeleteFile(bucketName, itemFile)
	if err != nil {
		return err
	}

	return nil
}

func (p *s3PersistenceProvider) CopyFile(souceBucketName string, sourceFileKe string, destBucketName string, destFileKey string) error {
	err := p.validateInit()
	if err != nil {
		return err
	}

	_, err = p.s3Client.CopyObject(&s3.CopyObjectInput{
		Bucket:     aws.String(destBucketName),
		CopySource: aws.String(souceBucketName + "/" + sourceFileKe),
		Key:        aws.String(destFileKey),
	})

	if err != nil {
		return err
	}

	return nil
}
