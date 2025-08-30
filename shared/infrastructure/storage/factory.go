package infrastorage

import (
	"fmt"

	"shared/config"
	"shared/domain/observability"
	"shared/domain/storage"
	"shared/infrastructure/storage/adapters/fs"
	"shared/infrastructure/storage/adapters/s3"
)

type Factory struct {
	logger  observability.Logger
	metrics observability.Metrics
}

func NewFactory(logger observability.Logger, metrics observability.Metrics) storage.StorageFactory {
	if logger == nil || metrics == nil {
		panic("logger and metrics are required for storage factory")
	}
	return &Factory{
		logger:  logger,
		metrics: metrics,
	}
}

func (f *Factory) Create(cfg *config.Config) (storage.ObjectStorage, error) {
	switch cfg.Adapters.Storage {
	case "s3":
		f.logger.Info("Creating S3 storage adapter",
			"bucket", cfg.Storage.BucketOrPath,
			"region", cfg.Storage.S3.Region)
		return s3.New(&cfg.Storage, f.logger, f.metrics)

	case "filesystem":
		f.logger.Info("Creating filesystem storage adapter",
			"path", cfg.Storage.BucketOrPath)
		return fs.NewStorage(cfg.Storage.BucketOrPath, f.logger, f.metrics)

	default:
		return nil, fmt.Errorf("unsupported storage adapter: %s", cfg.Adapters.Storage)
	}
}
