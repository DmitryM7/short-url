package repository

import (
	"context"
	"database/sql"

	"github.com/DmitryM7/short-url.git/internal/logger"
)

type InDBStorage struct {
	Logger      logger.MyLogger
	DatabaseDSN string
	db          *sql.DB
}

func NewInDBStorage(lg logger.MyLogger, dsn string) (*InDBStorage, error) {
	st := InDBStorage{
		DatabaseDSN: dsn,
		Logger:      lg,
	}

	err := st.connect()

	if err != nil {
		return &st, err
	}

	err = st.createSchema()

	if err != nil {
		return &st, err
	}

	return &st, err
}

func (l *InDBStorage) connect() error {
	db, err := sql.Open("pgx", l.DatabaseDSN)

	if err != nil {
		return err
	}

	if err := db.PingContext(context.Background()); err != nil {
		return err
	}

	l.db = db

	return nil
}

func (l *InDBStorage) createSchema() error {
	var tableName string

	row := l.db.QueryRowContext(context.Background(), "SELECT schemaname from pg_stat_user_tables WHERE relname LIKE 'repo'")

	err := row.Scan(&tableName)

	if err != nil {
		if err == sql.ErrNoRows {
			_, err = l.db.ExecContext(context.Background(), `CREATE TABLE repo ("id" SERIAL PRIMARY KEY,
			                                                                   "shorturl" VARCHAR NOT NULL UNIQUE,
																			"url" VARCHAR NOT NULL UNIQUE)`)
			if err != nil {
				return err
			}
		}

		return err
	}

	return nil
}

func (l *InDBStorage) Get(url string) (string, error) {
	var shorturl string
	row := l.db.QueryRowContext(context.Background(), "SELECT url FROM repo WHERE shorturl=$1", url)
	err := row.Scan(&shorturl)
	return shorturl, err
}

func (l *InDBStorage) GetByURL(url string) (string, error) {
	var shorturl string
	row := l.db.QueryRowContext(context.Background(), "SELECT shorturl FROM repo WHERE url=$1", url)
	err := row.Scan(&shorturl)
	return shorturl, err
}

func (l *InDBStorage) Create(lnkRec LinkRecord) error {
	_, err := l.db.ExecContext(context.Background(), "INSERT INTO repo (shorturl,url) VALUES($1,$2)", lnkRec.ShortURL, lnkRec.URL)

	if err != nil {
		return err
	}
	return nil
}

func (l *InDBStorage) BatchCreate(lnkRecs []LinkRecord) error {
	tx, err := l.db.Begin()

	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(context.Background(), "INSERT INTO repo (shorturl,url) VALUES($1,$2)")

	if err != nil {
		return err
	}

	for _, lnk := range lnkRecs {
		_, err := stmt.ExecContext(context.Background(), lnk.ShortURL, lnk.URL)

		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (l *InDBStorage) Ping() bool {
	if err := l.db.PingContext(context.Background()); err != nil {
		return false
	}

	return true
}
