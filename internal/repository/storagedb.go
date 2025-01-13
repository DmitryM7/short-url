package repository

import (
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type LinkRepoDB struct {
	LinkRepo
	DBProvider
	DatabaseDSN string
}

func NewLinkRepoDB(logger *zap.SugaredLogger, filePath string, dsn string) LinkRepoDB {
	return LinkRepoDB{
		DatabaseDSN: dsn,
		DBProvider:  NewDBProvider(dsn),
		LinkRepo:    NewLinkRepo(filePath, logger),
	}
}

func (l *LinkRepoDB) SaveInDB(shorturl, url string) error {
	err := l.DBProvider.Connect()

	if err != nil {
		return err
	}

	err = l.DBProvider.AddQuery("INSERT INTO repo (shorturl,url) VALUES($1,$2)", shorturl, url)

	return err

}

func (l *LinkRepoDB) CalcAndCreate(url string) string {

	shorturl := l.LinkRepo.CalcAndCreate(url)

	if l.DatabaseDSN != "" {

		err := l.SaveInDB(shorturl, url)

		if err != nil {
			l.DBProvider.RollBack()
			return shorturl
		}

		l.DBProvider.Commit()
	}

	return shorturl

}

func (l *LinkRepoDB) CalcAndCreateManualCommit(url string) (string, error) {

	shorturl := l.LinkRepo.CalcAndCreate(url)

	if l.DatabaseDSN != "" {

		err := l.SaveInDB(shorturl, url)

		return shorturl, err

	}

	return shorturl, nil

}

func (l *LinkRepoDB) Load() error {

	/************************************************
	 * Если не указан DSN, то грузим из файла       *
	 ************************************************/

	if l.DatabaseDSN == "" {
		return l.LinkRepo.Load()
	}

	rows, err := l.DBProvider.Load()

	if err != nil || rows.Err() != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var shorturl, url string
		err = rows.Scan(&shorturl, &url)

		if err != nil {
			return err
		}

		l.Create(shorturl, url)
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

	return 0, nil

}
