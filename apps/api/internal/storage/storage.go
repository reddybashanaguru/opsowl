package storage

import (
	"context"
	"io"
	"time"
)

// Client is the storage abstraction for document blobs.
type Client interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}
