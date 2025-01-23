package repository

type LinkRecord struct {
	UserID        int    `json:"UserID"`
	CorrelationID string `json:"-"`
	ShortURL      string `json:"ShortURL"`
	URL           string `json:"OriginalURL"`
}
