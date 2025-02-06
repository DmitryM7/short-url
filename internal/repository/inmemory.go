package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/DmitryM7/short-url.git/internal/logger"
)

const (
	rLength int64 = 100
)

type InMemoryStorage struct {
	Repo   map[string]string
	Logger logger.MyLogger
	m      sync.RWMutex
}

func NewInMemoryStorage(lg logger.MyLogger) (*InMemoryStorage, error) {
	lg.Infoln("CREATE NEW IN MEMORE STORAGE")

	return &InMemoryStorage{
		Logger: lg,
		Repo:   make(map[string]string, rLength),
	}, nil
}

func (r *InMemoryStorage) Create(ctx context.Context, lnkRec LinkRecord) error {
	r.m.Lock()
	r.Repo[lnkRec.ShortURL] = lnkRec.URL
	r.m.Unlock()
	return nil
}

func (r *InMemoryStorage) BatchCreate(ctx context.Context, lnkRecs []LinkRecord) error {
	for _, v := range lnkRecs {
		err := r.Create(ctx, v)

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *InMemoryStorage) Get(ctx context.Context, shorturl string) (string, error) {
	r.m.RLock()
	l, err := r.Repo[shorturl]
	r.m.RUnlock()

	if !err {
		return "", fmt.Errorf("CAN'T FIND LINK BY HASH")
	}

	return l, nil
}

func (r *InMemoryStorage) GetByURL(ctx context.Context, url string) (string, error) {
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

func (r *InMemoryStorage) Urls(ctx context.Context, userid int) ([]LinkRecord, error) {
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

func (r *InMemoryStorage) BatchDel(ctx context.Context, userid int, ursl []string) error {
	return nil
}
