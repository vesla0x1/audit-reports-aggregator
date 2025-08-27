package storage

import (
	"shared/config"
	"shared/observability"
	"shared/storage/adapters/s3"
	"shared/storage/types"
)

// createS3Storage creates an S3 storage implementation
// This function is only called by the provider's internal factory
func createS3Storage(cfg *config.Config, logger observability.Logger, metrics observability.Metrics) (types.ObjectStorage, error) {
	return s3.NewClient(&cfg.Storage, logger, metrics)
}
