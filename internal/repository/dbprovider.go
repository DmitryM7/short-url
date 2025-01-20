package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type DBProvider struct {
	Dsn string
	DB  *sql.DB
	Tr  *sql.Tx
}

func NewDBProvider(dsn string) DBProvider {
	DBProvider := DBProvider{Dsn: dsn}

	DBProvider.Connect() //nolint: errcheck //Successful connect will be chech in step "Connect"

	return DBProvider
}

func (b *DBProvider) Connect() error {
	db, err := sql.Open("pgx", b.Dsn)

	if err != nil {
		return err
	}

	if err := db.PingContext(context.Background()); err != nil {
		return err
	}

	b.DB = db

	return nil
}

func (b *DBProvider) RollBack() {
	if b.Tr != nil {
		b.Tr.Rollback() //nolint: errcheck //No matter
		b.Tr = nil
	}
}

func (b *DBProvider) Commit() {
	if b.Tr != nil {
		b.Tr.Commit() //nolint: errcheck //No matter
		b.Tr = nil
	}
}

func (b *DBProvider) Add(shorturl, url string) error {
	var err error = nil

	if b.Tr == nil {
		b.Tr, err = b.DB.Begin()
	}

	if err != nil {
		return err
	}

	_, err = b.Tr.ExecContext(context.Background(), "INSERT INTO repo (shorturl,url) VALUES($1,$2)", shorturl, url)
	if err != nil {
		return err
	}

	return nil
}

func (b *DBProvider) GetByURL(url string) (string, error) {
	var shorturl string
	row := b.DB.QueryRowContext(context.Background(), "SELECT shorturl FROM repo WHERE url=$1", url)
	err := row.Scan(&shorturl)
	return shorturl, err
}

func (b *DBProvider) Load() (*sql.Rows, error) {
	rows, err := b.DB.QueryContext(context.Background(), "SELECT id,shorturl,url FROM repo")

	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return rows, nil
}

func (b *DBProvider) Close() {
	b.DB.Close()
}

func (b *DBProvider) Ping() error {
	if b.DB == nil {
		return fmt.Errorf("NO DATABASE")
	} else {
		return nil
	}
}
