package repository

type LinkRecord struct {
	ID            int    `json:"-"`
	UserID        int    `json:"-"`
	CorrelationID string `json:"-"`
	ShortURL      string `json:"short_url"`
	URL           string `json:"original_url"`
}
