package s3

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	persistence "github.com/aludrey/go-aludrey-libs/pkg/persistence-v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

const e2eBucket = "aludrey-dev-bucket-test"

func TestUploadLocalFileE2E_withSimpleFileName(t *testing.T) {
	testFileName := "pepe-v2.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = persistenceProvider.UploadLocalFile("aludrey-dev-bucket-test", testFileName, testFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Couldn't remove test file: %v", err)
	}

	files, err := persistenceProvider.ListFiles("aludrey-dev-bucket-test", "")

	if err != nil {
		t.Errorf("Couln't list files: %v", err)
	}

	if len(files) == 0 {
		t.Errorf("No files found got 0 files, expected more than 0")
	}

	found := false

	for _, file := range files {
		if file == testFileName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("File %s not found", testFileName)
	}

	err = persistenceProvider.DeleteFile("aludrey-dev-bucket-test", testFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestUploadLocalFileE2E_withPrefix(t *testing.T) {
	testFileName := "pepe2-v2.txt"
	destFileName := "test/pepe2-v2.txt"
	prefix := "test/"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = persistenceProvider.UploadLocalFile("aludrey-dev-bucket-test", destFileName, testFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Couldn't remove test file: %v", err)
	}

	files, err := persistenceProvider.ListFiles("aludrey-dev-bucket-test", prefix)

	if err != nil {
		t.Errorf("Couln't list files: %v", err)
	}

	if len(files) == 0 {
		t.Errorf("No files found got 0 files, expected more than 0")
	}

	found := false

	for _, file := range files {
		if file == destFileName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("File %s not found", testFileName)
	}

	err = persistenceProvider.DeleteFile("aludrey-dev-bucket-test", destFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestMoveFileE2E(t *testing.T) {
	fileName := "movefilemock-v2.txt"
	movedFileName := "moved/movefilemock-v2.txt"

	file, err := os.Create(fileName)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	file.Close()

	err = persistenceProvider.UploadLocalFile("aludrey-dev-bucket-test", fileName, fileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = persistenceProvider.MoveFile("aludrey-dev-bucket-test", fileName, "aludrey-dev-bucket-test", movedFileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = os.Remove(fileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}
}

func TestDownloadFileE2E(t *testing.T) {
	testFileName := "testdownload-v2.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = persistenceProvider.UploadLocalFile("aludrey-dev-bucket-test", testFileName, testFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Coudn't delete local test file: %v", err)
	}

	file, err = persistenceProvider.DownloadFile("aludrey-dev-bucket-test", testFileName)
	if err != nil {
		t.Errorf("Error downloading file: %v", err)
	}

	if file == nil {
		t.Errorf("File not found: %v", file)
	}

	err = os.Remove(testFileName)
	if err != nil {
		t.Errorf("Coudn't delete local test file: %v", err)
	}

	err = persistenceProvider.DeleteFile("aludrey-dev-bucket-test", testFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestListFilesE2E_withSimpleFileName(t *testing.T) {
	testFileName := "listfiles-v2.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = persistenceProvider.UploadLocalFile("aludrey-dev-bucket-test", testFileName, testFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Coudn't delete local test file: %v", err)
	}

	files, err := persistenceProvider.ListFiles("aludrey-dev-bucket-test", "")

	if err != nil {
		t.Errorf("Couln't list files: %v", err)
	}

	if len(files) == 0 {
		t.Errorf("No files found got 0 files, expected more than 0")
	}

	found := false

	for _, file := range files {
		if file == testFileName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("File %s not found", testFileName)
	}

	err = persistenceProvider.DeleteFile("aludrey-dev-bucket-test", testFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestListFilesE2E_withPrefix(t *testing.T) {
	testFileName := "listfiles2-v2.txt"
	destFileName := "test/listfiles2-v2.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = persistenceProvider.UploadLocalFile("aludrey-dev-bucket-test", destFileName, testFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Coudn't delete local test file: %v", err)
	}

	files, err := persistenceProvider.ListFiles("aludrey-dev-bucket-test", "")

	if err != nil {
		t.Errorf("Couln't list files: %v", err)
	}

	if len(files) == 0 {
		t.Errorf("No files found got 0 files, expected more than 0")
	}

	found := false

	for _, file := range files {
		if file == destFileName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("File %s not found", testFileName)
	}

	err = persistenceProvider.DeleteFile("aludrey-dev-bucket-test", testFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestCopyFileE2E(t *testing.T) {
	testFileName := "copyfilemock-v2.txt"
	copiedTestFileName := "copied/copyfilemock-v2.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	file.Close()

	err = persistenceProvider.UploadLocalFile("aludrey-dev-bucket-test", testFileName, testFileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = persistenceProvider.CopyFile("aludrey-dev-bucket-test", testFileName, "aludrey-dev-bucket-test", copiedTestFileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = persistenceProvider.DeleteFile("aludrey-dev-bucket-test", testFileName)
	if err != nil {
		t.Errorf("Failed to remove original file %v", err)
	}

	err = persistenceProvider.DeleteFile("aludrey-dev-bucket-test", copiedTestFileName)
	if err != nil {
		t.Errorf("Failed to remove copied file: %v", err)
	}
}

func TestUploadStreamE2E(t *testing.T) {
	testFileName := "uploadstream-v2.txt"
	content := strings.NewReader("Hello, World!")

	err := persistenceProvider.UploadStream("aludrey-dev-bucket-test", testFileName, content)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	files, err := persistenceProvider.ListFiles("aludrey-dev-bucket-test", "")

	if err != nil {
		t.Errorf("Couln't list files: %v", err)
	}

	if len(files) == 0 {
		t.Errorf("No files found got 0 files, expected more than 0")
	}

	found := false

	for _, file := range files {
		if file == testFileName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("File %s not found", testFileName)
	}

	err = persistenceProvider.DeleteFile("aludrey-dev-bucket-test", testFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestListObjectsE2E_moreThan1000(t *testing.T) {
	prefix := e2ePrefix(t)
	// Más de 1000 para forzar una segunda página en S3.
	expected := make([]string, 1005)
	for i := range expected {
		expected[i] = fmt.Sprintf("%sobj-%04d.txt", prefix, i)
	}
	var group errgroup.Group
	group.SetLimit(32)
	for _, key := range expected {
		group.Go(func() error {
			return persistenceProvider.UploadStream(e2eBucket, key, strings.NewReader(key))
		})
	}
	require.NoError(t, group.Wait())

	objects, err := persistenceProvider.ListObjects(e2eBucket, prefix)
	require.NoError(t, err)
	keys := make([]string, 0, len(objects))
	for _, object := range objects {
		keys = append(keys, object.Key)
		assert.Equal(t, int64(len(object.Key)), object.Size, object.Key)
		assert.False(t, object.LastModified.IsZero(), object.Key)
		assert.NotEmpty(t, object.ETag, object.Key)
	}
	assert.ElementsMatch(t, expected, keys)

	files, err := persistenceProvider.ListFiles(e2eBucket, prefix)
	require.NoError(t, err)
	assert.ElementsMatch(t, expected, files)
}

func TestListObjectsE2E_missingPrefix(t *testing.T) {
	objects, err := persistenceProvider.ListObjects(e2eBucket, e2ePrefix(t))
	require.NoError(t, err)
	assert.Empty(t, objects)
}

func TestReadFileE2E(t *testing.T) {
	key, payload := uploadRandomPayload(t)

	data, err := persistenceProvider.ReadFile(e2eBucket, key, int64(len(payload)))
	require.NoError(t, err)
	assert.Equal(t, sha256.Sum256(payload), sha256.Sum256(data))

	data, err = persistenceProvider.ReadFile(e2eBucket, key, int64(len(payload)-1))
	assert.Nil(t, data)
	assert.ErrorIs(t, err, persistence.ErrObjectTooLarge)

	data, err = persistenceProvider.ReadFile(e2eBucket, key+".missing", 1024)
	assert.Nil(t, data)
	assert.ErrorIs(t, err, persistence.ErrObjectNotFound)
}

func TestPresignDownloadURLE2E(t *testing.T) {
	key, payload := uploadRandomPayload(t)

	signedURL, expiresAt, err := persistenceProvider.PresignDownloadURL(e2eBucket, key, time.Minute)
	require.NoError(t, err)
	assert.True(t, expiresAt.After(time.Now()))

	status, body := httpGet(t, signedURL)
	require.Equal(t, http.StatusOK, status, string(body))
	assert.Equal(t, sha256.Sum256(payload), sha256.Sum256(body))
}

func TestPresignDownloadURLE2E_expired(t *testing.T) {
	key, _ := uploadRandomPayload(t)

	signedURL, expiresAt, err := persistenceProvider.PresignDownloadURL(e2eBucket, key, 2*time.Second)
	require.NoError(t, err)

	time.Sleep(time.Until(expiresAt) + 2*time.Second)

	status, body := httpGet(t, signedURL)
	assert.Equal(t, http.StatusForbidden, status, string(body))
	assert.Contains(t, strings.ToLower(string(body)), "expired")
}

// e2ePrefix devuelve un prefijo único y borra al final todo lo subido bajo él.
func e2ePrefix(t *testing.T) string {
	t.Helper()
	prefix := "e2e-persistence-v2/" + uuid.NewString() + "/"
	t.Cleanup(func() {
		files, err := persistenceProvider.ListFiles(e2eBucket, prefix)
		if err != nil {
			t.Errorf("listing %s for cleanup: %v", prefix, err)
			return
		}
		var group errgroup.Group
		group.SetLimit(32)
		for _, file := range files {
			group.Go(func() error {
				return persistenceProvider.DeleteFile(e2eBucket, file)
			})
		}
		if err := group.Wait(); err != nil {
			t.Errorf("cleaning %s: %v", prefix, err)
		}
	})
	return prefix
}

func uploadRandomPayload(t *testing.T) (string, []byte) {
	t.Helper()
	payload := make([]byte, 64*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err)

	key := e2ePrefix(t) + "payload.bin"
	require.NoError(t, persistenceProvider.UploadStream(e2eBucket, key, bytes.NewReader(payload)))
	return key, payload
}

func httpGet(t *testing.T, rawURL string) (int, []byte) {
	t.Helper()
	resp, err := http.Get(rawURL)
	require.NoError(t, err)
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	require.NoError(t, readErr)
	require.NoError(t, closeErr)
	return resp.StatusCode, body
}
