package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"net/url"
)

type MinIOClient struct {
	client       *minio.Client  // internal endpoint — used for upload/delete/bucket ops
	publicClient *minio.Client  // public endpoint — used for presigned URL generation only
	bucket       string
}

func NewMinIO(endpoint, publicEndpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	m := &MinIOClient{client: client, bucket: bucket}

	// When a browser-accessible public endpoint differs from the internal Docker endpoint,
	// create a second client signed for that host. PresignedGetObject is pure HMAC math —
	// no live network connection is needed — so this client works from inside Docker.
	// Region must be set explicitly to prevent the SDK from making a bucket-location
	// lookup to the public endpoint (which is unreachable from inside the container).
	if publicEndpoint != "" && publicEndpoint != endpoint {
		region, _ := client.GetBucketLocation(context.Background(), bucket)
		if region == "" {
			region = "us-east-1"
		}
		pub, err := minio.New(publicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: useSSL,
			Region: region,
		})
		if err != nil {
			return nil, err
		}
		m.publicClient = pub
	}
	return m, nil
}

func (m *MinIOClient) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (m *MinIOClient) PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	signingClient := m.client
	if m.publicClient != nil {
		signingClient = m.publicClient
	}
	u, err := signingClient.PresignedGetObject(ctx, m.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (m *MinIOClient) Delete(ctx context.Context, key string) error {
	return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

// EnsureBucket creates the bucket if it does not already exist.
func (m *MinIOClient) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
	}
	return nil
}
