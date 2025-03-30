package repository

import (
	"errors"
	"fmt"

	"github.com/DmitryM7/short-url.git/internal/logger"
)

// ErrRecWasDelete - ошибка которое указывает на то, что запись была удалена
var ErrRecWasDelete = errors.New("RECORD WAS DELETED")

// Типы хранилища
const (
	// DBType - хранилище в СУБД
	DBType = "db"
	// MemType - хранилище в памяти
	MemType = "mem"
	// FileType - хранилище в файле
	FileType = "file"
)

// Тип хранилища
type StorageType string

// Конфигурация хранилища
type StorageConfig struct {
	StorageType StorageType
	Logger      logger.MyLogger
	DatabaseDSN string
	FilePath    string
}

// StorageConfig - фабрика по созданию хранилища
func NewStorage(cfg StorageConfig) (IRepo, error) {
	switch cfg.StorageType {
	case DBType:
		return NewInDBStorage(cfg.Logger, cfg.DatabaseDSN)

	case FileType:
		repo, err := NewInFileStorage(cfg.Logger, cfg.FilePath)
		if err != nil {
			return repo, err
		}
		err = repo.Load()

		if err != nil {
			return repo, fmt.Errorf("CANT LOAD DATA FROM FILE")
		}

		return repo, err
	case MemType:
		return NewInMemoryStorage(cfg.Logger)
	}
	return nil, fmt.Errorf("NO STORAGE TYPE")
}
