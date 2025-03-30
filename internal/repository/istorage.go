package repository

import "context"

// Интерфейс взаимодействия с хранилищем данных
type IStorage interface {
	Create(ctx context.Context, lnkRec LinkRecord) (string, error)
	Get(ctx context.Context, shorturl string) (string, error)
	GetByURL(ctx context.Context, url string) (string, error)
	BatchCreate(ctx context.Context, lnkRecs []LinkRecord) ([]LinkRecord, error)
	Urls(ctx context.Context, userid int) ([]LinkRecord, error)
	BatchDel(ctx context.Context, userid int, urls []string)
	Ping() bool
}
