package cloudstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	persistence "github.com/aludrey/go-aludrey-libs/pkg/persistence-v2"
	"google.golang.org/api/googleapi"
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

// ErrFileNotFound indica que el objeto pedido no existe en el bucket.
var ErrFileNotFound = errors.New("cloudstorage: file not found")

// Provider es el PersistenceProvider de Cloud Storage más las operaciones
// que sólo ofrece esta implementación.
type Provider interface {
	persistence.PersistenceProvider
	// OpenFile abre el objeto para leerlo en streaming, sin escribirlo en
	// disco (DownloadFile lo escribe en el directorio de trabajo, lo que no
	// sirve para lecturas concurrentes del mismo objeto). Quien llama debe
	// cerrar el reader. Si el objeto no existe devuelve ErrFileNotFound.
	OpenFile(ctx context.Context, bucketName string, fileKey string) (io.ReadCloser, error)
}

type gcpPersistenceProvider struct {
	client *storage.Client
}

func NewGCPPersistenceProvider(config GCPConfig) (persistence.PersistenceProvider, error) {
	return NewGCPProvider(config)
}

// NewGCPProvider es NewGCPPersistenceProvider con el tipo Provider, para
// usar además OpenFile.
func NewGCPProvider(config GCPConfig) (Provider, error) {
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

func (p *gcpPersistenceProvider) OpenFile(ctx context.Context, bucketName string, fileKey string) (io.ReadCloser, error) {
	reader, err := p.client.Bucket(bucketName).Object(fileKey).NewReader(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return nil, fmt.Errorf("%w: gs://%s/%s", ErrFileNotFound, bucketName, fileKey)
	}
	if err != nil {
		return nil, err
	}
	return reader, nil
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
	objects, err := p.ListObjects(bucketName, prefix)
	if err != nil {
		return nil, err
	}
	results := make([]string, 0, len(objects))
	for _, object := range objects {
		results = append(results, object.Key)
	}
	return results, nil
}

func (p *gcpPersistenceProvider) ListObjects(bucketName string, prefix string) ([]persistence.ObjectInfo, error) {
	if err := persistence.ValidateRepoName(bucketName); err != nil {
		return nil, err
	}

	query := &storage.Query{Prefix: prefix}
	if err := query.SetAttrSelection([]string{"Name", "Size", "Updated", "Etag"}); err != nil {
		return nil, fmt.Errorf("listing gs://%s/%s: %w", bucketName, prefix, err)
	}

	it := p.client.Bucket(bucketName).Objects(context.Background(), query)
	objects := make([]persistence.ObjectInfo, 0)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return objects, nil
		}
		if err != nil {
			return nil, fmt.Errorf("listing gs://%s/%s: %w", bucketName, prefix, err)
		}
		objects = append(objects, persistence.ObjectInfo{
			Key:          attrs.Name,
			Size:         attrs.Size,
			LastModified: attrs.Updated,
			ETag:         attrs.Etag,
		})
	}
}

func (p *gcpPersistenceProvider) ReadFile(bucketName string, itemFile string, maxBytes int64) ([]byte, error) {
	if err := persistence.ValidateReadFile(bucketName, itemFile, maxBytes); err != nil {
		return nil, err
	}

	location := "gs://" + bucketName + "/" + itemFile
	reader, err := p.client.Bucket(bucketName).Object(itemFile).NewReader(context.Background())
	if isObjectNotFound(err) {
		return nil, fmt.Errorf("%w: %s: %w", persistence.ErrObjectNotFound, location, err)
	}
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", location, err)
	}

	return persistence.ReadAllLimited(reader, location, maxBytes)
}

// Firma con la clave privada de la cuenta de servicio del provider. El
// cliente trunca la validez a segundos enteros: la URL puede durar hasta un
// segundo menos que ttl.
func (p *gcpPersistenceProvider) PresignDownloadURL(bucketName string, itemFile string, ttl time.Duration) (string, time.Time, error) {
	if err := persistence.ValidatePresign(bucketName, itemFile, ttl); err != nil {
		return "", time.Time{}, err
	}

	url, err := p.client.Bucket(bucketName).SignedURL(itemFile, &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  http.MethodGet,
		Expires: time.Now().Add(ttl),
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing gs://%s/%s: %w", bucketName, itemFile, err)
	}

	expiresAt, err := persistence.SignedURLExpiry(url, "X-Goog-Date", "X-Goog-Expires")
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing gs://%s/%s: %w", bucketName, itemFile, err)
	}

	return url, expiresAt, nil
}

// MoveFile copia el objeto y borra el original. Si el original no existe
// devuelve ErrFileNotFound (envolviendo el error del driver). Si la copia
// salió bien pero al borrar el original ya no estaba (otro proceso lo movió
// o borró en el medio), el movimiento se da por hecho: el destino existe y
// el origen ya no.
func (p *gcpPersistenceProvider) MoveFile(bucketName string, itemFile string, destBucketName string, destItemFile string) error {
	err := p.CopyFile(bucketName, itemFile, destBucketName, destItemFile)
	if isObjectNotFound(err) {
		return fmt.Errorf("%w: gs://%s/%s: %w", ErrFileNotFound, bucketName, itemFile, err)
	}
	if err != nil {
		return err
	}

	err = p.DeleteFile(bucketName, itemFile)
	if err != nil && !isObjectNotFound(err) {
		return err
	}

	return nil
}

// isObjectNotFound reconoce un objeto inexistente tanto en el error tipado
// del cliente (lecturas, borrados) como en el 404 crudo de la API (copias).
func isObjectNotFound(err error) bool {
	if errors.Is(err, storage.ErrObjectNotExist) {
		return true
	}
	var apiErr *googleapi.Error
	return errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound
}

func (p *gcpPersistenceProvider) CopyFile(souceBucketName string, sourceFileKe string, destBucketName string, destFileKey string) error {
	_, err := p.client.Bucket(destBucketName).Object(destFileKey).CopierFrom(p.client.Bucket(souceBucketName).Object(sourceFileKe)).Run(context.Background())
	return err
}
