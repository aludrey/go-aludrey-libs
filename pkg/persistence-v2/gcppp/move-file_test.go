package gcppp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

// moveFileServer answers the two calls MoveFile makes: the copy (rewrite)
// with copyStatus and the delete of the original with deleteStatus.
type moveFileServer struct {
	copyStatus   int
	deleteStatus int
	copied       bool
	deleted      bool
}

func newMoveFileProvider(t *testing.T, fake *moveFileServer) *gcpPersistenceProvider {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/rewriteTo/"):
			if fake.copyStatus != http.StatusOK {
				http.Error(w, `{"error":{"code":`+http.StatusText(fake.copyStatus)+`}}`, fake.copyStatus)
				return
			}
			fake.copied = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"kind":     "storage#rewriteResponse",
				"done":     true,
				"resource": map[string]any{"bucket": "bucket", "name": "processed/a.csv"},
			})
		case r.Method == http.MethodDelete:
			if fake.deleteStatus != http.StatusNoContent {
				http.Error(w, "{}", fake.deleteStatus)
				return
			}
			fake.deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := storage.NewClient(context.Background(),
		option.WithEndpoint(server.URL+"/storage/v1/"),
		option.WithoutAuthentication())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	return &gcpPersistenceProvider{client: client}
}

func TestMoveFile(t *testing.T) {
	fake := &moveFileServer{copyStatus: http.StatusOK, deleteStatus: http.StatusNoContent}
	provider := newMoveFileProvider(t, fake)

	err := provider.MoveFile("bucket", "pending/a.csv", "bucket", "processed/a.csv")

	require.NoError(t, err)
	assert.True(t, fake.copied)
	assert.True(t, fake.deleted)
}

// The original is gone before the copy: nothing to move.
func TestMoveFileSourceMissing(t *testing.T) {
	fake := &moveFileServer{copyStatus: http.StatusNotFound, deleteStatus: http.StatusNoContent}
	provider := newMoveFileProvider(t, fake)

	err := provider.MoveFile("bucket", "pending/a.csv", "bucket", "processed/a.csv")

	assert.ErrorIs(t, err, ErrFileNotFound)
	assert.ErrorContains(t, err, "gs://bucket/pending/a.csv")
	assert.False(t, fake.deleted, "no delete after a failed copy")
}

// The copy worked but someone else removed the original in between: the
// destination exists and the source does not, which is what a move is for.
func TestMoveFileSourceGoneBeforeDelete(t *testing.T) {
	fake := &moveFileServer{copyStatus: http.StatusOK, deleteStatus: http.StatusNotFound}
	provider := newMoveFileProvider(t, fake)

	err := provider.MoveFile("bucket", "pending/a.csv", "bucket", "processed/a.csv")

	assert.NoError(t, err)
	assert.True(t, fake.copied)
}

func TestMoveFileOtherFailures(t *testing.T) {
	t.Run("copy forbidden", func(t *testing.T) {
		provider := newMoveFileProvider(t, &moveFileServer{copyStatus: http.StatusForbidden, deleteStatus: http.StatusNoContent})

		err := provider.MoveFile("bucket", "pending/a.csv", "bucket", "processed/a.csv")

		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrFileNotFound)
	})

	t.Run("delete forbidden", func(t *testing.T) {
		provider := newMoveFileProvider(t, &moveFileServer{copyStatus: http.StatusOK, deleteStatus: http.StatusForbidden})

		err := provider.MoveFile("bucket", "pending/a.csv", "bucket", "processed/a.csv")

		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrFileNotFound)
	})
}
