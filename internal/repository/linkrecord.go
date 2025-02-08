package repository

type LinkRecord struct {
	ID            int    `json:"-"`
	UserID        int    `json:"-"`
	ShortURL      string `json:"short_url"`
	URL           string `json:"original_url"`
	CorrelationID string `json:"-"`
	DeletedFlag   bool   `json:"-"`
}
