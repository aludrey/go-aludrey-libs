package s3

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	persistence "github.com/aludrey/go-aludrey-libs/pkg/persistence-v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	mockFileName        string = "pepe-v2.txt"
	mockBucketName      string = "mockBucketName"
	persistenceProvider        = NewS3PersistenceProvider("us-east-2")
)

func TestDownloadFile(t *testing.T) {
	_, err := persistenceProvider.DownloadFile(mockBucketName, mockFileName)
	assert.NotNil(t, err)
}

func TestUploadLocalFile(t *testing.T) {
	err := persistenceProvider.UploadLocalFile(mockBucketName, mockFileName, mockFileName)
	assert.NotNil(t, err)
}

func TestUploadStream(t *testing.T) {
	hello := strings.NewReader("Hello, Reader!")
	err := persistenceProvider.UploadStream(mockBucketName, mockFileName, hello)
	assert.NotNil(t, err)
}

func TestListFiles(t *testing.T) {
	_, err := persistenceProvider.ListFiles(mockBucketName, mockFileName)
	assert.NotNil(t, err)
}

func TestMoveFile(t *testing.T) {
	err := persistenceProvider.MoveFile(mockBucketName, mockFileName, mockBucketName, mockFileName)
	assert.NotNil(t, err)
}

func newFakeS3Provider(t *testing.T, handler http.HandlerFunc) *s3PersistenceProvider {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String("us-east-2"),
		Endpoint:         aws.String(server.URL),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials("AKIDTEST", "SECRETTEST", ""),
		MaxRetries:       aws.Int(0),
	})
	require.NoError(t, err)

	return &s3PersistenceProvider{
		s3Client:     s3.New(sess),
		s3Downloader: s3manager.NewDownloader(sess),
		region:       "us-east-2",
	}
}

const noSuchKeyXML = `<?xml version="1.0" encoding="UTF-8"?>
<Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`

func objectHandler(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/bucket/dir/a.bin" {
			_, _ = w.Write([]byte(content))
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(noSuchKeyXML))
	}
}

func TestS3ReadFile(t *testing.T) {
	provider := newFakeS3Provider(t, objectHandler("hello"))

	data, err := provider.ReadFile("bucket", "dir/a.bin", 5)

	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestS3ReadFileTooLarge(t *testing.T) {
	provider := newFakeS3Provider(t, objectHandler("hello!"))

	data, err := provider.ReadFile("bucket", "dir/a.bin", 5)

	assert.Nil(t, data)
	assert.ErrorIs(t, err, persistence.ErrObjectTooLarge)
}

func TestS3ReadFileNotFound(t *testing.T) {
	provider := newFakeS3Provider(t, objectHandler("hello"))

	data, err := provider.ReadFile("bucket", "dir/missing.bin", 5)

	assert.Nil(t, data)
	assert.ErrorIs(t, err, persistence.ErrObjectNotFound)
	assert.ErrorContains(t, err, "s3://bucket/dir/missing.bin")
}

func TestS3ReadFileServerErrorIsNotNotFound(t *testing.T) {
	provider := newFakeS3Provider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := provider.ReadFile("bucket", "dir/a.bin", 5)

	require.Error(t, err)
	assert.NotErrorIs(t, err, persistence.ErrObjectNotFound)
}

func TestS3ReadFileInvalidArguments(t *testing.T) {
	provider := NewS3PersistenceProvider("us-east-2")

	for _, tc := range []struct {
		repo, key string
		maxBytes  int64
	}{
		{"", "a", 1},
		{"bucket", "", 1},
		{"bucket", "a", 0},
		{"bucket", "a", -10},
	} {
		_, err := provider.ReadFile(tc.repo, tc.key, tc.maxBytes)
		assert.ErrorIs(t, err, persistence.ErrInvalidArgument)
	}
}

func listHandler(t *testing.T, total int, pageSize int, calls *int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*calls++
		query := r.URL.Query()
		assert.Equal(t, "2", query.Get("list-type"))
		assert.Equal(t, "pre/", query.Get("prefix"))

		start := 0
		if token := query.Get("continuation-token"); token != "" {
			_, err := fmt.Sscanf(token, "token-%d", &start)
			require.NoError(t, err)
		}
		end := min(start+pageSize, total)

		var body strings.Builder
		body.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult><Name>bucket</Name><Prefix>pre/</Prefix>`)
		for i := start; i < end; i++ {
			fmt.Fprintf(&body, `<Contents><Key>pre/obj-%04d</Key><Size>%d</Size><LastModified>2026-11-23T10:00:00.000Z</LastModified><ETag>&quot;etag-%d&quot;</ETag></Contents>`, i, i, i)
		}
		if end < total {
			fmt.Fprintf(&body, `<IsTruncated>true</IsTruncated><NextContinuationToken>token-%d</NextContinuationToken>`, end)
		} else {
			body.WriteString(`<IsTruncated>false</IsTruncated>`)
		}
		body.WriteString(`</ListBucketResult>`)

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(body.String()))
	}
}

func TestS3ListObjectsPaginates(t *testing.T) {
	calls := 0
	provider := newFakeS3Provider(t, listHandler(t, 2501, 1000, &calls))

	objects, err := provider.ListObjects("bucket", "pre/")

	require.NoError(t, err)
	require.Len(t, objects, 2501)
	assert.Equal(t, 3, calls)
	assert.Equal(t, persistence.ObjectInfo{
		Key:          "pre/obj-2500",
		Size:         2500,
		LastModified: time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC),
		ETag:         "etag-2500",
	}, objects[2500])
}

func TestS3ListFilesPaginates(t *testing.T) {
	calls := 0
	provider := newFakeS3Provider(t, listHandler(t, 1001, 1000, &calls))

	files, err := provider.ListFiles("bucket", "pre/")

	require.NoError(t, err)
	require.Len(t, files, 1001)
	assert.Equal(t, "pre/obj-1000", files[1000])
}

func TestS3ListObjectsEmpty(t *testing.T) {
	calls := 0
	provider := newFakeS3Provider(t, listHandler(t, 0, 1000, &calls))

	objects, err := provider.ListObjects("bucket", "pre/")

	require.NoError(t, err)
	assert.Empty(t, objects)
}

func TestS3ListObjectsPropagatesError(t *testing.T) {
	provider := newFakeS3Provider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	objects, err := provider.ListObjects("bucket", "pre/")

	assert.Nil(t, objects)
	assert.Error(t, err)
}

func TestS3ListObjectsInvalidArguments(t *testing.T) {
	_, err := NewS3PersistenceProvider("us-east-2").ListObjects("", "pre/")
	assert.ErrorIs(t, err, persistence.ErrInvalidArgument)
}

func TestS3PresignDownloadURL(t *testing.T) {
	provider := newFakeS3Provider(t, objectHandler("hello"))

	before := time.Now()
	signedURL, expiresAt, err := provider.PresignDownloadURL("bucket", "dir/a.bin", 90*time.Second)

	require.NoError(t, err)
	parsed, err := url.Parse(signedURL)
	require.NoError(t, err)
	assert.Equal(t, "/bucket/dir/a.bin", parsed.Path)
	assert.Equal(t, "90", parsed.Query().Get("X-Amz-Expires"))
	assert.NotEmpty(t, parsed.Query().Get("X-Amz-Signature"))
	assert.WithinDuration(t, before.Add(90*time.Second), expiresAt, 2*time.Second)

	resp, err := http.Get(signedURL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestS3PresignDownloadURLInvalidTTL(t *testing.T) {
	provider := NewS3PersistenceProvider("us-east-2")

	for _, ttl := range []time.Duration{0, -time.Second, persistence.MaxPresignTTL + time.Second} {
		signedURL, expiresAt, err := provider.PresignDownloadURL("bucket", "a.bin", ttl)
		assert.ErrorIs(t, err, persistence.ErrInvalidArgument)
		assert.Empty(t, signedURL)
		assert.True(t, expiresAt.IsZero())
	}
}

func TestIsS3NotFound(t *testing.T) {
	assert.True(t, isS3NotFound(awserr.New(s3.ErrCodeNoSuchKey, "missing", nil)))
	assert.True(t, isS3NotFound(awserr.New("NotFound", "missing", nil)))
	assert.True(t, isS3NotFound(fmt.Errorf("wrapped: %w", awserr.New(s3.ErrCodeNoSuchKey, "missing", nil))))
	assert.False(t, isS3NotFound(awserr.New("AccessDenied", "denied", nil)))
	assert.False(t, isS3NotFound(errors.New("plain")))
	assert.False(t, isS3NotFound(nil))
}
