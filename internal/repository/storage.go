package repository

import (
	"errors"
	"fmt"

	"github.com/DmitryM7/short-url.git/internal/logger"
)

var ErrRecWasDelete = errors.New("RECORD WAS DELETE")

const (
	DBType   = "db"
	MemType  = "mem"
	FileType = "file"
)

type StorageType string

type StorageConfig struct {
	StorageType StorageType
	Logger      logger.MyLogger
	DatabaseDSN string
	FilePath    string
}

func NewStorage(cfg StorageConfig) (IStorage, error) {
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
