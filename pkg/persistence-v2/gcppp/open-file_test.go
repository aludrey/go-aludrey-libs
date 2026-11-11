package gcppp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

// newTestProvider apunta el cliente a un servidor local que sirve un único
// objeto: gs://bucket/runs/a.json.
func newTestProvider(t *testing.T) *gcpPersistenceProvider {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bucket/runs/a.json" {
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	client, err := storage.NewClient(context.Background(),
		option.WithEndpoint(server.URL+"/storage/v1/"),
		option.WithoutAuthentication())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	return &gcpPersistenceProvider{client: client}
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
