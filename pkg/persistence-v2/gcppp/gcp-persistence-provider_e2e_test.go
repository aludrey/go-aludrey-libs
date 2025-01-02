package gcppp_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/aludrey/go-aludrey-libs/pkg/persistence-v2/gcppp"
	"github.com/aludrey/go-aludrey-libs/pkg/secret"
)

func LoadConfig() gcppp.GCPConfig {
	secret, err := secret.GetSecretValue("dev", "go-aludrey-libs", "cloud_storage", "us-east-2")
	if err != nil {
		panic(err)
	}
	cnf := gcppp.GCPConfig{}
	err = json.Unmarshal([]byte(secret), &cnf)
	if err != nil {
		panic(err)
	}
	return cnf
}

func TestUploadLocalFileE2E_withSimpleFileName(t *testing.T) {
	persistenceProvider, err := gcppp.NewGCPPersistenceProvider(LoadConfig())
	if err != nil {
		t.Errorf("Error creating persistence provider: %v", err)
	}
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
	persistenceProvider, err := gcppp.NewGCPPersistenceProvider(LoadConfig())
	if err != nil {
		t.Errorf("Error creating persistence provider: %v", err)
	}

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
	persistenceProvider, err := gcppp.NewGCPPersistenceProvider(LoadConfig())
	if err != nil {
		t.Errorf("Error creating persistence provider: %v", err)
	}

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
	persistenceProvider, err := gcppp.NewGCPPersistenceProvider(LoadConfig())
	if err != nil {
		t.Errorf("Error creating persistence provider: %v", err)
	}

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
	persistenceProvider, err := gcppp.NewGCPPersistenceProvider(LoadConfig())
	if err != nil {
		t.Errorf("Error creating persistence provider: %v", err)
	}

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
	persistenceProvider, err := gcppp.NewGCPPersistenceProvider(LoadConfig())
	if err != nil {
		t.Errorf("Error creating persistence provider: %v", err)
	}

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

	err = persistenceProvider.DeleteFile("aludrey-dev-bucket-test", destFileName)

	if err != nil {
		t.Errorf("Couldn't delete file: %v", err)
	}
}

func TestCopyFileE2E(t *testing.T) {
	persistenceProvider, err := gcppp.NewGCPPersistenceProvider(LoadConfig())
	if err != nil {
		t.Errorf("Error creating persistence provider: %v", err)
	}

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
	persistenceProvider, err := gcppp.NewGCPPersistenceProvider(LoadConfig())
	if err != nil {
		t.Errorf("Error creating persistence provider: %v", err)
	}

	testFileName := "uploadstream-v2.txt"
	content := strings.NewReader("Hello, World!")

	err = persistenceProvider.UploadStream("aludrey-dev-bucket-test", testFileName, content)

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
