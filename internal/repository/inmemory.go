package repository

import (
	"fmt"

	"github.com/DmitryM7/short-url.git/internal/logger"
)

const (
	rLength int64 = 100
)

type InMemoryStorage struct {
	Repo   map[string]string
	Logger logger.MyLogger
}

func NewInMemoryStorage(lg logger.MyLogger) (*InMemoryStorage, error) {
	lg.Infoln("CREATE NEW IN MEMORE STORAGE")

	return &InMemoryStorage{
		Logger: lg,
		Repo:   make(map[string]string, rLength),
	}, nil
}

func (r *InMemoryStorage) Create(lnkRec LinkRecord) error {
	r.Repo[lnkRec.ShortURL] = lnkRec.URL
	return nil
}

func (r *InMemoryStorage) BatchCreate(lnkRecs []LinkRecord) error {
	for _, v := range lnkRecs {
		err := r.Create(v)

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *InMemoryStorage) Get(shorturl string) (string, error) {
	l, err := r.Repo[shorturl]

	if !err {
		return "", fmt.Errorf("CAN'T FIND LINK BY HASH")
	}

	return l, nil
}

func (r *InMemoryStorage) GetByURL(url string) (string, error) {
	for k, v := range r.Repo {
		if v == url {
			return k, nil
		}
	}
	return "", fmt.Errorf("NO URL IN REPO")
}

func (r *InMemoryStorage) Ping() bool {
	return true
}

func (r *InMemoryStorage) Urls(userid int) ([]LinkRecord, error) {
	res := []LinkRecord{}

	for k, v := range r.Repo {
		lnkRec := LinkRecord{
			ShortURL: k,
			URL:      v,
		}
		res = append(res, lnkRec)
	}

	return res, nil
}

func (r *InMemoryStorage) BatchDel(userid int, ursl []string) error {
	return nil
}
