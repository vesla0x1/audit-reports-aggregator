package storage

import (
	"shared/config"
	"shared/domain/base"
)

const storageProviderKey = "storage"

func GetProvider() *base.Provider[ObjectStorage] {
	return base.GetProvider[ObjectStorage](storageProviderKey)
}

func Initialize(cfg *config.Config, factory StorageFactory) error {
	return GetProvider().Initialize(cfg, factory)
}

func GetStorage() (ObjectStorage, error) {
	return GetProvider().Get()
}

func MustGetStorage() ObjectStorage {
	return GetProvider().MustGet()
}

func IsInitialized() bool {
	return GetProvider().IsInitialized()
}
