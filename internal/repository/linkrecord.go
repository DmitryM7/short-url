package repository

type LinkRecord struct {
	UserID        int    `json:"-"`
	CorrelationID string `json:"-"`
	ShortURL      string `json:"Short_URL"`
	URL           string `json:"Original_URL"`
}
