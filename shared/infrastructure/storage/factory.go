package infrastorage

import (
	"fmt"
	"shared/config"
	"shared/domain/storage"
	"shared/infrastructure/storage/adapters/s3"
	"strings"
)

type Factory struct {
	//logProvider observability.LogProvider
}

func NewFactory() storage.StorageFactory {
	return &Factory{}
}

func (f *Factory) CreateStorage(cfg *config.Config) (storage.ObjectStorage, error) {
	switch strings.ToLower(cfg.GetStorageProvider()) {
	case "s3":
		return s3.NewStorage(&cfg.Storage)
	default:
		// TODO use logger
		return nil, fmt.Errorf("unsupported provider: %s", cfg.GetStorageProvider())
	}

}
