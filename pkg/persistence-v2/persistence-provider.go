package persistencev2

import (
	"io"
	"os"
)

type PersistenceProvider interface {
	DownloadFile(repoName string, filePath string) (*os.File, error)
	DeleteFile(repoName string, filePath string) error
	UploadLocalFile(repoName string, filePath string, localPath string) error
	UploadStream(repoName string, filePath string, streamReader io.ReadSeeker) error
	ListFiles(repoName string, prefix string) ([]string, error)
	MoveFile(sourceRepoName string, sourceFilePath string, destRepoName string, destFilePath string) error
	CopyFile(sourceRepoName string, sourceFilePath string, destRepoName string, destFilePath string) error
}
