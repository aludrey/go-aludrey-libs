package persistence

import (
	"io"
	"os"

	"github.com/aludrey/go-aludrey-libs/pkg/log"
)

type PersistenceProvider interface {
	DownloadFile(log log.Logger, repoName string, filePath string) (*os.File, error)
	DeleteFile(log log.Logger, repoName string, filePath string) error
	UploadLocalFile(log log.Logger, repoName string, filePath string, localPath string) error
	UploadStream(log log.Logger, repoName string, filePath string, streamReader io.ReadSeeker) error
	ListFiles(log log.Logger, repoName string, prefix string) ([]string, error)
	MoveFile(log log.Logger, sourceRepoName string, sourceFilePath string, destRepoName string, destFilePath string) error
	CopyFile(log log.Logger, sourceRepoName string, sourceFilePath string, destRepoName string, destFilePath string) error
}
