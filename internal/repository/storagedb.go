package repository

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/stdlib"
	"go.uber.org/zap"
)

type LinkRepoDB struct {
	LinkRepo
	DatabaseDSN string
}

func NewLinkRepoDB(logger *zap.SugaredLogger) LinkRepoDB {
	return LinkRepoDB{

		LinkRepo: LinkRepo{
			Logger: logger,
		},
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
	fmt.Println(l.DatabaseDSN)
	defer db.Close()

	return db, err
}
