package storage

import "io"

type FileStore interface {
	Save(name string, reader io.Reader) (path string, size int64, err error)
	Open(path string) (io.ReadCloser, error)
	Delete(path string) error
}
