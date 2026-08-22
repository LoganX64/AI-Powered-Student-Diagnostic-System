package storage

import (
	"context"
	"io"
)

// Storage is the object-storage interface. Local disk and Cloudinary both
// implement it so the exam video tier can swap backends without touching handlers.
type Storage interface {
	// Put stores the reader's contents under key and returns a retrievable URL.
	Put(ctx context.Context, key string, r io.Reader) (url string, err error)

	// Get retrieves the object at key. Caller must close the returned ReadCloser.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// List returns all object keys under the given prefix.
	List(ctx context.Context, prefix string) ([]string, error)

	// Delete removes the object at key.
	Delete(ctx context.Context, key string) error

	// DeletePrefix removes all objects under the given prefix.
	DeletePrefix(ctx context.Context, prefix string) error
}
