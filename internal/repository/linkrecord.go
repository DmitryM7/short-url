package repository

type LinkRecord struct {
	ID            int    `json:"Id"`
	UserID        int    `json:"UserId"`
	CorrelationID string `json:"-"`
	ShortURL      string `json:"ShortURL"`
	URL           string `json:"OriginalURL"`
}
