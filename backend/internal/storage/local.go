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
		return "", fmt.Errorf("create directory: %w", err)
	}
	f, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return dst, nil
}

func (l *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	path := filepath.Join(l.Dir, filepath.FromSlash(key))
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

func (l *LocalStorage) List(ctx context.Context, prefix string) ([]string, error) {
	base := filepath.Join(l.Dir, filepath.FromSlash(prefix))
	info, err := os.Stat(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("prefix is not a directory: %s", prefix)
	}

	var keys []string
	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error at %s: %w", path, err)
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(l.Dir, path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}
		keys = append(keys, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}
	return keys, nil
}

func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	path := filepath.Join(l.Dir, filepath.FromSlash(key))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func (l *LocalStorage) DeletePrefix(ctx context.Context, prefix string) error {
	dir := filepath.Join(l.Dir, filepath.FromSlash(prefix))
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete prefix: %w", err)
	}
	return nil
}
