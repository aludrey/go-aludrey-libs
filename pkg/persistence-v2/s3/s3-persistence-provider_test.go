package s3pp

import (
	"strings"
	"testing"
	"github.com/stretchr/testify/assert"
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
