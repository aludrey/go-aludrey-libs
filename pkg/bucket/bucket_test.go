package bucket

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	mockFileName   string = "pepe.txt"
	mockBucketName string = "mockBucketName"
)

func TestDownloadFile(t *testing.T) {
	_, err := DownloadFile(context.Background(), mockBucketName, mockFileName)
	assert.NotNil(t, err)
}

func TestUploadFile(t *testing.T) {
	err := UploadFile(context.Background(), mockBucketName, mockFileName, mockFileName)
	assert.NotNil(t, err)
}

func TestListFiles(t *testing.T) {
	_, err := ListFiles(context.Background(), mockBucketName, mockFileName)
	assert.NotNil(t, err)
}

func TestMoveFile(t *testing.T) {
	err := MoveFile(context.Background(), mockBucketName, mockFileName, mockBucketName, mockFileName)
	assert.NotNil(t, err)
}
