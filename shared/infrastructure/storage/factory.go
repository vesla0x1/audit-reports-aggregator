package storage

import (
	"fmt"
	"shared/application/ports"
	"shared/infrastructure/config"
)

func CreateStorage(cfg *config.Config, obs ports.Observability) (ports.Storage, error) {
	logger, err := obs.LoggerScoped("storage.factory")
	if err != nil {
		return nil, fmt.Errorf("failed to get logger from observability: %w", err)
	}

	switch cfg.Adapters.Storage {
	case "s3":
		logger.Info("Creating S3 storage adapter",
			"bucket", cfg.Storage.BucketOrPath,
			"region", cfg.Storage.S3.Region)
		return NewS3Storage(&cfg.Storage, obs)

	case "filesystem":
		logger.Info("Creating filesystem storage adapter",
			"path", cfg.Storage.BucketOrPath)
		return NewFSStorage(cfg.Storage.BucketOrPath, obs)

	default:
		return nil, fmt.Errorf("unsupported storage adapter: %s", cfg.Adapters.Storage)
	}
}
