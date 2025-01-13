package repository

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type LinkRepoDB struct {
	LinkRepo
	DatabaseDSN string
}

func NewLinkRepoDB(logger *zap.SugaredLogger, filePath string, dsn string) LinkRepoDB {
	return LinkRepoDB{
		DatabaseDSN: dsn,
		LinkRepo:    NewLinkRepo(filePath, logger),
	}
}

func (l *LinkRepoDB) Connect() (*sql.DB, error) {

	db, err := sql.Open("pgx", l.DatabaseDSN)

	if err != nil {
		return nil, err
	}

	if err := db.PingContext(context.Background()); err != nil {
		return nil, err
	}

	return db, err
}

func (l *LinkRepoDB) CreateSchema(db *sql.DB) error {
	var tableName string

	row := db.QueryRowContext(context.Background(), "SELECT schemaname from pg_stat_user_tables WHERE relname LIKE 'repo'")

	err := row.Scan(&tableName)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	_, err = db.ExecContext(context.Background(), `CREATE TABLE repo ("id" SERIAL PRIMARY KEY,"shorturl" VARCHAR NOT NULL UNIQUE,"url" VARCHAR NOT NULL)`)
	if err != nil {
		return err
	}

	return nil

}

func (l *LinkRepoDB) Load() error {

	/************************************************
	 * Если не указан DSN, то грузим из файла       *
	 ************************************************/

	if l.DatabaseDSN == "" {
		return l.LinkRepo.Load()
	}

	db, err := l.Connect()
	if err != nil {
		return err
	}

	err = l.CreateSchema(db)

	if err != nil {
		return err
	}

	rows, err := db.QueryContext(context.Background(), "SELECT id,shorturl,url FROM repo")

	if err != nil {
		return err
	}

	if rows.Err() != nil {
		return rows.Err()
	}

	defer rows.Close()

	for rows.Next() {
		var id int
		var k, s string

		err = rows.Scan(&id, &k, &s)

		if err != nil {
			return err
		}

		l.repo[k] = s
	}

	return nil
}

func (l *LinkRepoDB) Unload() (int, error) {

	/************************************************
	 * Если не указан DSN, то выгружаем в файл      *
	 ************************************************/

	if l.DatabaseDSN == "" {
		return l.LinkRepo.Unload()
	}

	var cLines int
	db, err := l.Connect()

	if err != nil {
		return cLines, err
	}

	_, err = db.ExecContext(context.Background(), `TRUNCATE repo`)

	if err != nil {
		return cLines, err
	}

	tx, err := db.Begin()

	if err != nil {
		return cLines, err
	}

	for k, v := range l.repo {

		_, err := tx.ExecContext(context.Background(), "INSERT INTO repo (shorturl,url) VALUES($1,$2)", k, v)

		if err != nil {
			tx.Rollback()
			return 0, err
		}

		cLines++

	}

	tx.Commit()

	return cLines, err
}
