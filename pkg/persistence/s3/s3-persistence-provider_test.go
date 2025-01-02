package s3pp

import (
	"strings"
	"testing"

	log "github.com/aludrey/go-aludrey-libs/pkg/log"
	"github.com/stretchr/testify/assert"
)

var (
	mockFileName        string = "pepe.txt"
	mockBucketName      string = "mockBucketName"
	persistenceProvider        = NewS3PersistenceProvider("us-east-2")
	logger                     = log.NewLogger(log.LogrusLoggerConfig{
		Level:        log.InfoLevel,
		ReportCaller: true,
	})
)

func TestDownloadFile(t *testing.T) {
	_, err := persistenceProvider.DownloadFile(logger, mockBucketName, mockFileName)
	assert.NotNil(t, err)
}

func TestUploadLocalFile(t *testing.T) {
	err := persistenceProvider.UploadLocalFile(logger, mockBucketName, mockFileName, mockFileName)
	assert.NotNil(t, err)
}

func TestUploadStream(t *testing.T) {
	hello := strings.NewReader("Hello, Reader!")
	err := persistenceProvider.UploadStream(logger, mockBucketName, mockFileName, hello)
	assert.NotNil(t, err)
}

func TestListFiles(t *testing.T) {
	_, err := persistenceProvider.ListFiles(logger, mockBucketName, mockFileName)
	assert.NotNil(t, err)
}

func TestMoveFile(t *testing.T) {
	err := persistenceProvider.MoveFile(logger, mockBucketName, mockFileName, mockBucketName, mockFileName)
	assert.NotNil(t, err)
}
