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
	// GetURL returns a URL/path from which an already-stored key can be fetched.
	GetURL(key string) string
}
