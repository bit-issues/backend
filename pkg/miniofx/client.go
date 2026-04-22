package miniofx

import (
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewClient(config Config) (*minio.Client, error) {
	endpoint := config.Endpoint

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewEnvAWS(),
		Secure: true,
		Region: config.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return client, nil
}
