package repository

import (
	"context"
	"fmt"
	"hash/crc32"
)

// IRepo - интерфейс взаимодействия с поставщиком данных
type IRepo interface {
	Create(ctx context.Context, lnkRec LinkRecord) error
	Get(ctx context.Context, shorturl string) (string, error)
	GetByURL(ctx context.Context, url string) (string, error)
	BatchCreate(ctx context.Context, lnkRecs []LinkRecord) error
	Urls(ctx context.Context, userid int) ([]LinkRecord, error)
	BatchDel(ctx context.Context, userid int, urls []string)
	Ping() bool
}

// StorageService - хранилище данных
type StorageService struct {
	storage IRepo
}

// NewStorageService - констурктор хранилища данных
func NewStorageService(cfg StorageConfig) (IStorage, error) {
	repo, err := NewStorage(cfg)

	if err != nil {
		return &StorageService{}, err
	}

	return &StorageService{storage: repo}, nil
}

// BatchCreate - пакетное создание ссылок
func (s *StorageService) BatchCreate(ctx context.Context, lnkRecs []LinkRecord) ([]LinkRecord, error) {
	for k, v := range lnkRecs {
		lnkRecs[k].ShortURL = s.сalcShortURL(v.URL)
	}

	err := s.storage.BatchCreate(ctx, lnkRecs)

	if err != nil {
		return lnkRecs, err
	}

	return lnkRecs, nil
}

// BatchDel - пакетное удаление
func (s *StorageService) BatchDel(ctx context.Context, userid int, urls []string) {
	s.storage.BatchDel(ctx, userid, urls)
}

func (s *StorageService) сalcShortURL(url string) string {
	return fmt.Sprintf("%08x", crc32.Checksum([]byte(url), crc32.MakeTable(crc32.IEEE)))
}

// Create - создает короткую ссылку
func (s *StorageService) Create(ctx context.Context, lnkRec LinkRecord) (string, error) {
	shortURL := s.сalcShortURL(lnkRec.URL)
	lnkRec.ShortURL = shortURL
	return shortURL, s.storage.Create(ctx, lnkRec)
}

// Get - возвращает длинную ссылку по короткой
func (s *StorageService) Get(ctx context.Context, shorturl string) (string, error) {
	return s.storage.Get(ctx, shorturl)
}

// GetByURL - возвращает сохраненную длинную сслыку по короткой
func (s *StorageService) GetByURL(ctx context.Context, url string) (string, error) {
	return s.storage.GetByURL(ctx, url)
}

// Ping - проверяет соединение с поставщиком данных
func (s *StorageService) Ping() bool {
	return s.storage.Ping()
}

// Urls - возращает ссылки пользователя
func (s *StorageService) Urls(ctx context.Context, userid int) ([]LinkRecord, error) {
	return s.storage.Urls(ctx, userid)
}
