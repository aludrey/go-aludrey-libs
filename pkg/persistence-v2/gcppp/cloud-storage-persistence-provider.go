package gcppp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	persistence "github.com/aludrey/go-aludrey-libs/pkg/persistence-v2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GCPConfig struct {
	Type           string `json:"type"`
	ProjectID      string `json:"project_id"`
	PrivateKeyID   string `json:"private_key_id"`
	PrivateKey     string `json:"private_key"`
	ClientEmail    string `json:"client_email"`
	ClientID       string `json:"client_id"`
	AuthURI        string `json:"auth_uri"`
	TokenURI       string `json:"token_uri"`
	AuthProvider   string `json:"auth_provider_x509_cert_url"`
	ClientCertURL  string `json:"client_x509_cert_url"`
	UniverseDomain string `json:"universe_domain"`
}

type gcpPersistenceProvider struct {
	client *storage.Client
}

func NewGCPPersistenceProvider(config GCPConfig) (persistence.PersistenceProvider, error) {
	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON(jsonConfig))
	if err != nil {
		return nil, err
	}

	return &gcpPersistenceProvider{
		client: client,
	}, nil
}

func (p *gcpPersistenceProvider) DownloadFile(bucketName string, itemFile string) (*os.File, error) {
	reader, err := p.client.Bucket(bucketName).Object(itemFile).NewReader(context.Background())
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(itemFile)

	file, err := os.Create(filename)
	if err != nil {
		return file, err
	}

	defer file.Close()
	_, err = io.Copy(file, reader)

	return file, err
}

func (p *gcpPersistenceProvider) DeleteFile(bucketName string, itemFile string) error {
	return p.client.Bucket(bucketName).Object(itemFile).Delete(context.Background())
}

func (p *gcpPersistenceProvider) UploadLocalFile(bucketName string, fileKey string, localFilePath string) error {
	file, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return p.UploadStream(bucketName, fileKey, file)
}

func (p *gcpPersistenceProvider) UploadStream(bucketName string, fileKey string, streamReader io.ReadSeeker) error {
	ctx := context.Background()
	writer := p.client.Bucket(bucketName).Object(fileKey).NewWriter(ctx)
	_, err := io.Copy(writer, streamReader)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	return ctx.Err()
}

func (p *gcpPersistenceProvider) ListFiles(bucketName string, prefix string) ([]string, error) {
	objects := p.client.Bucket(bucketName).Objects(context.Background(), &storage.Query{
		Prefix: prefix,
	})
	if objects == nil {
		return nil, errors.New("no objects found")
	}
	next, err := objects.Next()
	results := make([]string, 0)
	for err == nil && next != nil {
		results = append(results, next.Name)
		next, err = objects.Next()
	}
	if err != iterator.Done {
		return nil, err
	}
	return results, nil
}

func (p *gcpPersistenceProvider) MoveFile(bucketName string, itemFile string, destBucketName string, destItemFile string) error {
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

func (p *gcpPersistenceProvider) CopyFile(souceBucketName string, sourceFileKe string, destBucketName string, destFileKey string) error {
	_, err := p.client.Bucket(destBucketName).Object(destFileKey).CopierFrom(p.client.Bucket(souceBucketName).Object(sourceFileKe)).Run(context.Background())
	return err
}
