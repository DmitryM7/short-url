package repository

import (
	"github.com/DmitryM7/short-url.git/internal/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type LinkRepoDB struct {
	LinkRepo
	DBProvider
	DatabaseDSN string
}

func NewLinkRepoDB(log logger.MyLogger, filePath, dsn string) LinkRepoDB {
	return LinkRepoDB{
		DatabaseDSN: dsn,
		DBProvider:  NewDBProvider(dsn),
		LinkRepo:    NewLinkRepo(filePath, log),
	}
}

func (l *LinkRepoDB) SaveInDB(shorturl, url string) error {
	err := l.DBProvider.Connect()

	if err != nil {
		return err
	}

	err = l.DBProvider.Add(shorturl, url)

	return err
}

func (l *LinkRepoDB) CalcAndCreate(url string) (string, error) {
	shorturl, _ := l.LinkRepo.CalcAndCreate(url)

	if l.DatabaseDSN != "" {
		err := l.SaveInDB(shorturl, url)

		if err != nil {
			l.DBProvider.RollBack()
			return shorturl, err
		}

		l.DBProvider.Commit()
	}

	return shorturl, nil
}

func (l *LinkRepoDB) CalcAndCreateManualCommit(url string) (string, error) {
	shorturl, _ := l.LinkRepo.CalcAndCreate(url)

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
		var id int
		var shorturl, url string
		err = rows.Scan(&id, &shorturl, &url)

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
