package repository

type LinkRecord struct {
	ID            int    `json:"-"`
	UserID        int    `json:"-"`
	CorrelationID string `json:"-"`
	ShortURL      string `json:"ShortURL"`
	URL           string `json:"OriginalURL"`
}
