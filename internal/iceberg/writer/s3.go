package writer

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Client defines the interface for S3/MinIO operations.
type S3Client interface {
	// Upload uploads data to the specified bucket and key.
	Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, contentType string) error

	// Delete deletes an object from the bucket.
	Delete(ctx context.Context, bucket, key string) error

	// Exists checks if an object exists.
	Exists(ctx context.Context, bucket, key string) (bool, error)

	// EnsureBucket ensures the bucket exists.
	EnsureBucket(ctx context.Context, bucket string) error
}

// S3Config holds S3/MinIO configuration.
type S3Config struct {
	// Endpoint is the S3/MinIO endpoint (e.g., "localhost:9000").
	Endpoint string

	// AccessKey is the access key.
	AccessKey string

	// SecretKey is the secret key.
	SecretKey string

	// UseSSL enables SSL for the connection.
	UseSSL bool

	// Region is the S3 region (optional for MinIO).
	Region string
}

// MinIOClient implements S3Client using the MinIO SDK.
type MinIOClient struct {
	client *minio.Client
	logger *slog.Logger
}

// NewMinIOClient creates a new MinIO S3 client.
func NewMinIOClient(cfg S3Config, logger *slog.Logger) (*MinIOClient, error) {
	if logger == nil {
		logger = slog.Default()
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinIOClient{
		client: client,
		logger: logger.With("component", "s3-client"),
	}, nil
}

// Upload uploads data to the specified bucket and key.
func (c *MinIOClient) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	info, err := c.client.PutObject(ctx, bucket, key, data, size, opts)
	if err != nil {
		return fmt.Errorf("upload object: %w", err)
	}

	c.logger.Debug("object uploaded",
		"bucket", bucket,
		"key", key,
		"size", info.Size,
	)

	return nil
}

// Delete deletes an object from the bucket.
func (c *MinIOClient) Delete(ctx context.Context, bucket, key string) error {
	err := c.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	c.logger.Debug("object deleted",
		"bucket", bucket,
		"key", key,
	)

	return nil
}

// Exists checks if an object exists.
func (c *MinIOClient) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := c.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		// Check if it's a "not found" error
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("stat object: %w", err)
	}
	return true, nil
}

// EnsureBucket ensures the bucket exists, creating it if necessary.
func (c *MinIOClient) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := c.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket exists: %w", err)
	}

	if exists {
		return nil
	}

	err = c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}

	c.logger.Info("bucket created", "bucket", bucket)
	return nil
}

// GetObjectURL returns the URL for an object.
func (c *MinIOClient) GetObjectURL(bucket, key string) string {
	return fmt.Sprintf("s3://%s/%s", bucket, key)
}

// Ensure MinIOClient implements S3Client.
var _ S3Client = (*MinIOClient)(nil)
