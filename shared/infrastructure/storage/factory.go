package infrastorage

import (
	"fmt"
	"log"
	"shared/config"
	"shared/domain/observability"
	"shared/domain/storage"
	"shared/infrastructure/storage/adapters/s3"
	"strings"
)

type Factory struct {
	logger  observability.Logger
	metrics observability.Metrics
}

func NewFactoryWithObservability(logger observability.Logger, metrics observability.Metrics) storage.StorageFactory {
	if logger == nil || metrics == nil {
		log.Fatalf("logger and metrics must be provided")
		return nil
	}
	return &Factory{
		logger:  logger,
		metrics: metrics,
	}
}

func (f *Factory) CreateStorage(cfg *config.Config) (storage.ObjectStorage, error) {
	switch strings.ToLower(cfg.GetStorageProvider()) {
	case "s3":
		return s3.New(&cfg.Storage, f.logger, f.metrics)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.GetStorageProvider())
	}

}
