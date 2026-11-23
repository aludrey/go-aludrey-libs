package cloudstorage

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	persistence "github.com/aludrey/go-aludrey-libs/pkg/persistence-v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func newFakeGCSProvider(t *testing.T, handler http.HandlerFunc) *gcpPersistenceProvider {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := storage.NewClient(context.Background(),
		option.WithEndpoint(server.URL+"/storage/v1/"),
		option.WithoutAuthentication())
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("closing storage client: %v", err)
		}
	})

	return &gcpPersistenceProvider{client: client}
}

// newTestProvider sirve un único objeto: gs://bucket/runs/a.json.
func newTestProvider(t *testing.T) *gcpPersistenceProvider {
	t.Helper()

	return newFakeGCSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bucket/runs/a.json" {
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	})
}

func TestOpenFile(t *testing.T) {
	provider := newTestProvider(t)

	reader, err := provider.OpenFile(context.Background(), "bucket", "runs/a.json")
	require.NoError(t, err)
	defer reader.Close()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(content))
}

func TestOpenFileNotFound(t *testing.T) {
	provider := newTestProvider(t)

	reader, err := provider.OpenFile(context.Background(), "bucket", "runs/missing.json")

	assert.Nil(t, reader)
	assert.ErrorIs(t, err, ErrFileNotFound)
	assert.ErrorContains(t, err, "gs://bucket/runs/missing.json")
}

func TestOpenFileCanceledContext(t *testing.T) {
	provider := newTestProvider(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reader, err := provider.OpenFile(ctx, "bucket", "runs/a.json")

	assert.Nil(t, reader)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestNewGCPProviderRejectsInvalidCredentials(t *testing.T) {
	provider, err := NewGCPProvider(GCPConfig{Type: "service_account"})

	assert.Nil(t, provider)
	assert.Error(t, err)
}

func objectServer(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bucket/dir/a.bin" {
			_, _ = w.Write([]byte(content))
			return
		}
		http.NotFound(w, r)
	}
}

func TestGCSReadFile(t *testing.T) {
	provider := newFakeGCSProvider(t, objectServer("hello"))

	data, err := provider.ReadFile("bucket", "dir/a.bin", 5)

	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestGCSReadFileTooLarge(t *testing.T) {
	provider := newFakeGCSProvider(t, objectServer("hello!"))

	data, err := provider.ReadFile("bucket", "dir/a.bin", 5)

	assert.Nil(t, data)
	assert.ErrorIs(t, err, persistence.ErrObjectTooLarge)
}

func TestGCSReadFileNotFound(t *testing.T) {
	provider := newFakeGCSProvider(t, objectServer("hello"))

	data, err := provider.ReadFile("bucket", "dir/missing.bin", 5)

	assert.Nil(t, data)
	assert.ErrorIs(t, err, persistence.ErrObjectNotFound)
	assert.ErrorIs(t, err, storage.ErrObjectNotExist)
	assert.ErrorContains(t, err, "gs://bucket/dir/missing.bin")
}

func TestGCSReadFileServerErrorIsNotNotFound(t *testing.T) {
	provider := newFakeGCSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	_, err := provider.ReadFile("bucket", "dir/a.bin", 5)

	require.Error(t, err)
	assert.NotErrorIs(t, err, persistence.ErrObjectNotFound)
}

func TestGCSReadFileInvalidArguments(t *testing.T) {
	provider := newFakeGCSProvider(t, objectServer("hello"))

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

func listServer(t *testing.T, total int, pageSize int, calls *int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storage/v1/b/bucket/o" {
			http.NotFound(w, r)
			return
		}
		*calls++
		query := r.URL.Query()
		assert.Equal(t, "pre/", query.Get("prefix"))

		start := 0
		if token := query.Get("pageToken"); token != "" {
			var err error
			start, err = strconv.Atoi(token)
			require.NoError(t, err)
		}
		end := min(start+pageSize, total)

		items := make([]map[string]any, 0, end-start)
		for i := start; i < end; i++ {
			items = append(items, map[string]any{
				"bucket":  "bucket",
				"name":    fmt.Sprintf("pre/obj-%04d", i),
				"size":    strconv.Itoa(i),
				"updated": "2026-11-23T10:00:00.000Z",
				"etag":    fmt.Sprintf("etag-%d", i),
			})
		}
		response := map[string]any{"kind": "storage#objects", "items": items}
		if end < total {
			response["nextPageToken"] = strconv.Itoa(end)
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}
}

func TestGCSListObjectsPaginates(t *testing.T) {
	calls := 0
	provider := newFakeGCSProvider(t, listServer(t, 2501, 1000, &calls))

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

func TestGCSListFilesPaginates(t *testing.T) {
	calls := 0
	provider := newFakeGCSProvider(t, listServer(t, 1001, 1000, &calls))

	files, err := provider.ListFiles("bucket", "pre/")

	require.NoError(t, err)
	require.Len(t, files, 1001)
	assert.Equal(t, "pre/obj-1000", files[1000])
}

func TestGCSListEmptyPrefix(t *testing.T) {
	calls := 0
	provider := newFakeGCSProvider(t, listServer(t, 0, 1000, &calls))

	objects, err := provider.ListObjects("bucket", "pre/")
	require.NoError(t, err)
	assert.Empty(t, objects)

	files, err := provider.ListFiles("bucket", "pre/")
	require.NoError(t, err)
	assert.NotNil(t, files)
	assert.Empty(t, files)
}

func TestGCSListObjectsPropagatesError(t *testing.T) {
	provider := newFakeGCSProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	objects, err := provider.ListObjects("bucket", "pre/")
	assert.Nil(t, objects)
	assert.Error(t, err)

	files, err := provider.ListFiles("bucket", "pre/")
	assert.Nil(t, files)
	assert.Error(t, err)
}

func TestGCSListObjectsInvalidArguments(t *testing.T) {
	provider := newFakeGCSProvider(t, objectServer("hello"))

	_, err := provider.ListObjects("", "pre/")
	assert.ErrorIs(t, err, persistence.ErrInvalidArgument)
}

// Firmar URLs no hace llamadas de red: basta una cuenta de servicio con una
// clave generada en el test.
func newSigningProvider(t *testing.T) Provider {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	provider, err := NewGCPProvider(GCPConfig{
		Type:         "service_account",
		ProjectID:    "test-project",
		PrivateKeyID: "test-key-id",
		PrivateKey:   string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		ClientEmail:  "signer@test-project.iam.gserviceaccount.com",
		ClientID:     "123",
		TokenURI:     "https://oauth2.googleapis.com/token",
	})
	require.NoError(t, err)
	return provider
}

func TestGCSPresignDownloadURL(t *testing.T) {
	provider := newSigningProvider(t)

	before := time.Now()
	signedURL, expiresAt, err := provider.PresignDownloadURL("bucket", "dir/a.bin", 90*time.Second)

	require.NoError(t, err)
	parsed, err := url.Parse(signedURL)
	require.NoError(t, err)
	assert.Equal(t, "/bucket/dir/a.bin", parsed.Path)
	assert.Equal(t, "GOOG4-RSA-SHA256", parsed.Query().Get("X-Goog-Algorithm"))
	assert.Contains(t, []string{"89", "90"}, parsed.Query().Get("X-Goog-Expires"))
	assert.Contains(t, parsed.Query().Get("X-Goog-Credential"), "signer@test-project.iam.gserviceaccount.com")
	assert.NotEmpty(t, parsed.Query().Get("X-Goog-Signature"))
	assert.False(t, expiresAt.After(before.Add(90*time.Second)))
	assert.WithinDuration(t, before.Add(90*time.Second), expiresAt, 2*time.Second)
}

func TestGCSPresignDownloadURLInvalidTTL(t *testing.T) {
	provider := newSigningProvider(t)

	for _, ttl := range []time.Duration{0, -time.Second, persistence.MaxPresignTTL + time.Second} {
		signedURL, expiresAt, err := provider.PresignDownloadURL("bucket", "a.bin", ttl)
		assert.ErrorIs(t, err, persistence.ErrInvalidArgument)
		assert.Empty(t, signedURL)
		assert.True(t, expiresAt.IsZero())
	}
}
