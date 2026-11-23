package persistencev2

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"time"
)

// MaxPresignTTL es el máximo que admiten S3 (SigV4) y Cloud Storage (V4).
const MaxPresignTTL = 7 * 24 * time.Hour

var (
	ErrObjectNotFound  = errors.New("persistencev2: object not found")
	ErrObjectTooLarge  = errors.New("persistencev2: object exceeds max bytes")
	ErrInvalidArgument = errors.New("persistencev2: invalid argument")
)

type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
}

type PersistenceProvider interface {
	DownloadFile(repoName string, filePath string) (*os.File, error)
	DeleteFile(repoName string, filePath string) error
	UploadLocalFile(repoName string, filePath string, localPath string) error
	UploadStream(repoName string, filePath string, streamReader io.ReadSeeker) error
	ListFiles(repoName string, prefix string) ([]string, error)
	MoveFile(sourceRepoName string, sourceFilePath string, destRepoName string, destFilePath string) error
	CopyFile(sourceRepoName string, sourceFilePath string, destRepoName string, destFilePath string) error
	ListObjects(repoName string, prefix string) ([]ObjectInfo, error)
	ReadFile(repoName string, filePath string, maxBytes int64) ([]byte, error)
	PresignDownloadURL(repoName string, filePath string, ttl time.Duration) (string, time.Time, error)
}

func ValidateRepoName(repoName string) error {
	if repoName == "" {
		return fmt.Errorf("%w: repoName is empty", ErrInvalidArgument)
	}
	return nil
}

func ValidateObjectRef(repoName string, filePath string) error {
	if err := ValidateRepoName(repoName); err != nil {
		return err
	}
	if filePath == "" {
		return fmt.Errorf("%w: filePath is empty", ErrInvalidArgument)
	}
	return nil
}

func ValidateReadFile(repoName string, filePath string, maxBytes int64) error {
	if err := ValidateObjectRef(repoName, filePath); err != nil {
		return err
	}
	if maxBytes <= 0 {
		return fmt.Errorf("%w: maxBytes must be greater than 0, got %d", ErrInvalidArgument, maxBytes)
	}
	return nil
}

func ValidatePresign(repoName string, filePath string, ttl time.Duration) error {
	if err := ValidateObjectRef(repoName, filePath); err != nil {
		return err
	}
	if ttl <= 0 || ttl > MaxPresignTTL {
		return fmt.Errorf("%w: ttl must be in (0, %s], got %s", ErrInvalidArgument, MaxPresignTTL, ttl)
	}
	return nil
}

// ReadAllLimited lee maxBytes+1 como mucho para detectar objetos más grandes
// sin cargarlos enteros en memoria.
func ReadAllLimited(body io.ReadCloser, location string, maxBytes int64) ([]byte, error) {
	data, readErr := io.ReadAll(io.LimitReader(body, maxBytes+1))
	closeErr := body.Close()

	if readErr != nil {
		readErr = fmt.Errorf("reading %s: %w", location, readErr)
	}
	if closeErr != nil {
		closeErr = fmt.Errorf("closing %s: %w", location, closeErr)
	}
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: %s is larger than %d bytes", ErrObjectTooLarge, location, maxBytes)
	}
	return data, nil
}

// SignedURLExpiry calcula la expiración desde la propia URL firmada: los SDK
// redondean la fecha de firma y la validez a segundos, así que estimarla con
// time.Now()+ttl puede adelantarse hasta un segundo a la real.
func SignedURLExpiry(rawURL string, dateParam string, expiresParam string) (time.Time, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing signed url: %w", err)
	}
	query := parsed.Query()

	signedAt, err := time.Parse("20060102T150405Z", query.Get(dateParam))
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing %s of signed url: %w", dateParam, err)
	}
	seconds, err := strconv.ParseInt(query.Get(expiresParam), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing %s of signed url: %w", expiresParam, err)
	}

	return signedAt.Add(time.Duration(seconds) * time.Second), nil
}
