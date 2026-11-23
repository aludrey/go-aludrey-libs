package persistencev2

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateReadFile(t *testing.T) {
	cases := []struct {
		name     string
		repo     string
		key      string
		maxBytes int64
		wantErr  bool
	}{
		{"valid", "bucket", "a.txt", 1, false},
		{"empty repo", "", "a.txt", 1, true},
		{"empty key", "bucket", "", 1, true},
		{"zero maxBytes", "bucket", "a.txt", 0, true},
		{"negative maxBytes", "bucket", "a.txt", -1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateReadFile(tc.repo, tc.key, tc.maxBytes)
			if tc.wantErr {
				assert.ErrorIs(t, err, ErrInvalidArgument)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestValidatePresign(t *testing.T) {
	cases := []struct {
		name    string
		repo    string
		key     string
		ttl     time.Duration
		wantErr bool
	}{
		{"valid", "bucket", "a.txt", time.Minute, false},
		{"max ttl", "bucket", "a.txt", MaxPresignTTL, false},
		{"empty repo", "", "a.txt", time.Minute, true},
		{"empty key", "bucket", "", time.Minute, true},
		{"zero ttl", "bucket", "a.txt", 0, true},
		{"negative ttl", "bucket", "a.txt", -time.Second, true},
		{"ttl over max", "bucket", "a.txt", MaxPresignTTL + time.Second, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePresign(tc.repo, tc.key, tc.ttl)
			if tc.wantErr {
				assert.ErrorIs(t, err, ErrInvalidArgument)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestValidateRepoName(t *testing.T) {
	assert.NoError(t, ValidateRepoName("bucket"))
	assert.ErrorIs(t, ValidateRepoName(""), ErrInvalidArgument)
}

type fakeBody struct {
	io.Reader
	closeErr error
	closed   bool
}

func (b *fakeBody) Close() error {
	b.closed = true
	return b.closeErr
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestReadAllLimited(t *testing.T) {
	t.Run("exact size", func(t *testing.T) {
		body := &fakeBody{Reader: strings.NewReader("hello")}
		data, err := ReadAllLimited(body, "loc", 5)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data))
		assert.True(t, body.closed)
	})

	t.Run("too large", func(t *testing.T) {
		body := &fakeBody{Reader: strings.NewReader("hello!")}
		data, err := ReadAllLimited(body, "loc", 5)
		assert.Nil(t, data)
		assert.ErrorIs(t, err, ErrObjectTooLarge)
		assert.True(t, body.closed)
	})

	t.Run("close error is propagated", func(t *testing.T) {
		closeErr := errors.New("close failed")
		body := &fakeBody{Reader: strings.NewReader("hello"), closeErr: closeErr}
		data, err := ReadAllLimited(body, "loc", 5)
		assert.Nil(t, data)
		assert.ErrorIs(t, err, closeErr)
	})

	t.Run("read and close errors are joined", func(t *testing.T) {
		readErr := errors.New("read failed")
		closeErr := errors.New("close failed")
		body := &fakeBody{Reader: failingReader{err: readErr}, closeErr: closeErr}
		data, err := ReadAllLimited(body, "loc", 5)
		assert.Nil(t, data)
		assert.ErrorIs(t, err, readErr)
		assert.ErrorIs(t, err, closeErr)
		assert.True(t, body.closed)
	})
}
func TestSignedURLExpiry(t *testing.T) {
	expiresAt, err := SignedURLExpiry("https://host/b/k?X-Amz-Date=20261123T100000Z&X-Amz-Expires=90", "X-Amz-Date", "X-Amz-Expires")

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 11, 23, 10, 1, 30, 0, time.UTC), expiresAt)
}

func TestSignedURLExpiryErrors(t *testing.T) {
	for _, rawURL := range []string{
		"://bad",
		"https://host/b/k?X-Amz-Expires=90",
		"https://host/b/k?X-Amz-Date=20261123T100000Z",
		"https://host/b/k?X-Amz-Date=20261123T100000Z&X-Amz-Expires=abc",
	} {
		_, err := SignedURLExpiry(rawURL, "X-Amz-Date", "X-Amz-Expires")
		assert.Error(t, err, rawURL)
	}
}
