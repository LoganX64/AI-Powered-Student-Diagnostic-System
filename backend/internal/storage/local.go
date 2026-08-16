package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage writes files to a local directory tree (key becomes a relative path).
type LocalStorage struct {
	Dir string
}

func NewLocal(dir string) *LocalStorage {
	if dir == "" {
		dir = "./uploads"
	}
	return &LocalStorage{Dir: dir}
}

func (l *LocalStorage) Put(ctx context.Context, key string, r io.Reader) (string, error) {
	dst := filepath.Join(l.Dir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	return dst, nil
}

func (l *LocalStorage) GetURL(key string) string {
	return fmt.Sprintf("file://%s", filepath.Join(l.Dir, filepath.FromSlash(key)))
}
