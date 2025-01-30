package repository

import (
	"fmt"
	"hash/crc32"
)

type StorageService struct {
	storage IStorage
}

func NewStorageService(cfg StorageConfig) (StorageService, error) {
	repo, err := NewStorage(cfg)

	if err != nil {
		return StorageService{}, err
	}

	return StorageService{storage: repo}, nil
}

func (s *StorageService) BatchCreate(lnkRecs []LinkRecord) ([]LinkRecord, error) {
	for k, v := range lnkRecs {
		lnkRecs[k].ShortURL = s.сalcShortURL(v.URL)
	}

	err := s.storage.BatchCreate(lnkRecs)

	if err != nil {
		return lnkRecs, err
	}

	return lnkRecs, nil
}

func (s *StorageService) BatchDel(userid int, urls []string) error {
	return s.storage.BatchDel(userid, urls)
}

func (s *StorageService) сalcShortURL(url string) string {
	return fmt.Sprintf("%08x", crc32.Checksum([]byte(url), crc32.MakeTable(crc32.IEEE)))
}

func (s *StorageService) Create(lnkRec LinkRecord) (string, error) {
	shortURL := s.сalcShortURL(lnkRec.URL)
	lnkRec.ShortURL = shortURL
	return shortURL, s.storage.Create(lnkRec)
}

func (s *StorageService) Get(shorturl string) (string, error) {
	return s.storage.Get(shorturl)
}

func (s *StorageService) GetByURL(url string) (string, error) {
	return s.storage.GetByURL(url)
}

func (s *StorageService) Ping() bool {
	return s.storage.Ping()
}

func (s *StorageService) Urls(userid int) ([]LinkRecord, error) {
	return s.storage.Urls(userid)
}
