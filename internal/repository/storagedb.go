package repository

import (
	"context"
	"database/sql"
	"fmt"

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

	if l.DatabaseDSN == "" {
		l.DatabaseDSN = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
			`localhost`,
			`video`,
			`!QAZ2wsx123`,
			`video`)
	}

	db, err := sql.Open("pgx", l.DatabaseDSN)

	if err != nil {
		return nil, err
	}

	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		return nil, err
	}

	return db, err
}
