package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type DBProvider struct {
	Dsn string
	Db  *sql.DB
	Tr  *sql.Tx
}

func NewDBProvider(dsn string) DBProvider {
	DBProvider := DBProvider{Dsn: dsn}
	DBProvider.Connect()
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

	b.Db = db

	return nil
}

func (b *DBProvider) RollBack() {
	if b.Tr != nil {
		b.Tr.Rollback()
		b.Tr = nil
	}
}

func (b *DBProvider) Commit() {
	if b.Tr != nil {
		b.Tr.Commit()
		b.Tr = nil
	}
}

func (b *DBProvider) CreateSchema() error {
	var tableName string

	row := b.Db.QueryRowContext(context.Background(), "SELECT schemaname from pg_stat_user_tables WHERE relname LIKE 'repo'")

	err := row.Scan(&tableName)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	_, err = b.Db.ExecContext(context.Background(), `CREATE TABLE repo ("id" SERIAL PRIMARY KEY,"shorturl" VARCHAR NOT NULL UNIQUE,"url" VARCHAR NOT NULL)`)
	if err != nil {
		return err
	}

	return nil

}

func (b *DBProvider) AddQuery(query string, shorturl string, url string) error {
	var err error = nil

	if b.Tr == nil {
		b.Tr, err = b.Db.Begin()
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

func (b *DBProvider) Load() (*sql.Rows, error) {

	err := b.CreateSchema()

	if err != nil {
		return nil, err
	}

	rows, err := b.Db.QueryContext(context.Background(), "SELECT id,shorturl,url FROM repo")

	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return rows, nil

}

func (b *DBProvider) Close() {

	b.Db.Close()

}

func (b *DBProvider) Ping() error {
	if b.Db == nil {
		return fmt.Errorf("NO DATABASE")
	} else {
		return nil
	}
}
