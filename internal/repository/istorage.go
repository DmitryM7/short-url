package repository

type IStorage interface {
	Create(lnkRec LinkRecord) error
	Get(shorturl string) (string, error)
	GetByURL(url string) (string, error)
	BatchCreate(lnkRecs []LinkRecord) error
	Urls(userid int) ([]LinkRecord, error)
	BatchDel(userid int, urls []string) error
	Ping() bool
}
