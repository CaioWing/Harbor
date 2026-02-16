package local

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type LocalStore struct {
	basePath string
}

func New(basePath string) (*LocalStore, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &LocalStore{basePath: basePath}, nil
}

func (s *LocalStore) Save(name string, reader io.Reader) (string, int64, error) {
	dir := filepath.Join(s.basePath, time.Now().Format("2006/01/02"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", 0, fmt.Errorf("create date dir: %w", err)
	}

	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", 0, fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, reader)
	if err != nil {
		os.Remove(path)
		return "", 0, fmt.Errorf("write file: %w", err)
	}

	return path, n, nil
}

func (s *LocalStore) Open(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

func (s *LocalStore) Delete(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}
