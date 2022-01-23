package filestore

import (
	"io"
	"time"
)

type FileStore interface {
	// GetFiles returns the list of file that are stored in this file store.
	GetFiles() ([]*StoredFile, error)
	// GetFileContents returns the contents of the file that are stored in this file store.
	GetFileContents(path string) (io.ReadCloser, error)
	// WriteFileContents writes the contents to the file store. Creates the file if it doesn't exist. Also creates any
	// parent directories that do not exist.
	WriteFileContents(path string, contentReader io.Reader) error
	// DeleteFile deletes the file in this file store.
	DeleteFile(path string) error
	// DeleteAllFiles deletes all files in this file store.
	DeleteAllFiles() error
	// GetModifiedTime returns the time the file was last modified.
	GetModifiedTime(path string) (time.Time, error)
	// FileExists returns true if the file exists in this file store.
	FileExists(path string) (bool, error)
}

type StoredFile struct {
	Path  string
	IsDir bool
}
