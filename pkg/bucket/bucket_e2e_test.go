package bucket

import (
	"context"
	"os"
	"testing"
)

func TestUploadFileE2E_withSimpleFileName(t *testing.T) {
	testFileName := "pepe.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = UploadFile(context.Background(), config.BucketTest, testFileName, testFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Couldn't remove test file: %v", err)
	}

	files, err := ListFiles(context.Background(), config.BucketTest, "")

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

	err = DeleteFile(context.Background(), config.BucketTest, testFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestUploadFileE2E_withPrefix(t *testing.T) {
	testFileName := "pepe2.txt"
	destFileName := "test/pepe2.txt"
	prefix := "test/"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = UploadFile(context.Background(), config.BucketTest, testFileName, destFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Couldn't remove test file: %v", err)
	}

	files, err := ListFiles(context.Background(), config.BucketTest, prefix)

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

	err = DeleteFile(context.Background(), config.BucketTest, destFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestMoveFileE2E(t *testing.T) {
	fileName := "movefilemock.txt"
	movedFileName := "moved/movefilemock.txt"

	file, err := os.Create(fileName)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	file.Close()

	err = UploadFile(context.Background(), config.BucketTest, fileName, fileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = MoveFile(context.Background(), config.BucketTest, fileName, config.BucketTest, movedFileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = os.Remove(fileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}
}

func TestDownloadFileE2E(t *testing.T) {
	testFileName := "testdownload.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = UploadFile(context.Background(), config.BucketTest, testFileName, testFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Coudn't delete local test file: %v", err)
	}

	file, err = DownloadFile(context.Background(), config.BucketTest, testFileName)
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

	err = DeleteFile(context.Background(), config.BucketTest, testFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestListFilesE2E_withSimpleFileName(t *testing.T) {
	testFileName := "listfiles.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = UploadFile(context.Background(), config.BucketTest, testFileName, testFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Coudn't delete local test file: %v", err)
	}

	files, err := ListFiles(context.Background(), config.BucketTest, "")

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

	err = DeleteFile(context.Background(), config.BucketTest, testFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestListFilesE2E_withPrefix(t *testing.T) {
	testFileName := "listfiles2.txt"
	destFileName := "test/listfiles2.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Couldn't create test file: %v", err)
	}

	file.Close()

	err = UploadFile(context.Background(), config.BucketTest, testFileName, destFileName)

	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Coudn't delete local test file: %v", err)
	}

	files, err := ListFiles(context.Background(), config.BucketTest, "")

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

	err = DeleteFile(context.Background(), config.BucketTest, testFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestCopyFileE2E(t *testing.T) {
	testFileName := "copyfilemock.txt"
	copiedTestFileName := "copied/copyfilemock.txt"

	file, err := os.Create(testFileName)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	file.Close()

	err = UploadFile(context.Background(), config.BucketTest, testFileName, testFileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = CopyFile(context.Background(), config.BucketTest, testFileName, config.BucketTest, copiedTestFileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = os.Remove(testFileName)

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	err = DeleteFile(context.Background(), config.BucketTest, testFileName)
	if err != nil {
		t.Errorf("Failed to remove original file %v", err)
	}

	err = DeleteFile(context.Background(), config.BucketTest, copiedTestFileName)
	if err != nil {
		t.Errorf("Failed to remove copied file: %v", err)
	}
}
