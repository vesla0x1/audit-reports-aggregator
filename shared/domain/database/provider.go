package database

import (
	"shared/config"
	"shared/domain/base"
)

const databaseProviderKey = "database"

func GetProvider() *base.Provider[Database] {
	return base.GetProvider[Database](databaseProviderKey)
}

func Initialize(cfg *config.Config, factory DatabaseFactory) error {
	return GetProvider().Initialize(cfg, factory)
}

func GetDatabase() (Database, error) {
	return GetProvider().Get()
}

func MustGetDatabase() Database {
	return GetProvider().MustGet()
}

func IsInitialized() bool {
	return GetProvider().IsInitialized()
}

func Close() error {
	db, err := GetDatabase()
	if err != nil {
		return err
	}
	return db.Close()
}
