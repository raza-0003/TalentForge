package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage stores files in an S3 bucket (or any S3-compatible endpoint such as
// MinIO). It implements Storage.
type S3Storage struct {
	client *s3.Client
	bucket string
}

// NewS3Storage builds an S3Storage. Credentials are resolved by the default AWS
// chain (env vars, shared config, IAM role). endpoint is optional and, when set,
// switches to path-style addressing for S3-compatible servers.
func NewS3Storage(ctx context.Context, bucket, region, endpoint string) (*S3Storage, error) {
	if bucket == "" {
		return nil, fmt.Errorf("s3 storage: bucket is required")
	}

	var opts []func(*config.LoadOptions) error
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})
	return &S3Storage{client: client, bucket: bucket}, nil
}

// Save uploads r to the object at key.
func (s *S3Storage) Save(ctx context.Context, key string, r io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		return fmt.Errorf("s3 put %q: %w", key, err)
	}
	return nil
}

// Open returns a reader for the object at key. The caller must Close it.
func (s *S3Storage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get %q: %w", key, err)
	}
	return out.Body, nil
}
