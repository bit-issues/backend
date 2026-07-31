package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type Service struct {
	bucketName    string
	keyPrefix     string
	presignExpiry time.Duration

	client *minio.Client

	logger *zap.Logger
}

func NewService(config Config, client *minio.Client, logger *zap.Logger) (*Service, error) {
	u, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse storage URL: %w", err)
	}

	return &Service{
		bucketName:    u.Hostname(),
		keyPrefix:     strings.TrimPrefix(u.Path, "/"),
		presignExpiry: config.LinksTTL,

		client: client,

		logger: logger,
	}, nil
}

func (s *Service) PresignedPutObject(ctx context.Context, key string) (string, error) {
	u, err := s.client.PresignedPutObject(
		ctx,
		s.bucketName,
		s.objectKey(key),
		s.presignExpiry,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create presigned url: %w", err)
	}

	return u.String(), nil
}

func (s *Service) PutObject(ctx context.Context, key string, filePath string) error {
	_, err := s.client.FPutObject(ctx, s.bucketName, s.objectKey(key), filePath, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	return nil
}

// PresignedGetObject returns a presigned URL for downloading an object.
func (s *Service) PresignedGetObject(ctx context.Context, key string) (string, error) {
	u, err := s.client.PresignedGetObject(
		ctx,
		s.bucketName,
		s.objectKey(key),
		s.presignExpiry,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create presigned url: %w", err)
	}

	return u.String(), nil
}

func (s *Service) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucketName, s.objectKey(key), minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (s *Service) objectKey(key string) string {
	if s.keyPrefix == "" {
		return key
	}
	return strings.TrimSuffix(s.keyPrefix, "/") + "/" + strings.TrimPrefix(key, "/")
}
