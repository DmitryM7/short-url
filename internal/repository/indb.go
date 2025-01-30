package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/DmitryM7/short-url.git/internal/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type InDBStorage struct {
	Logger      logger.MyLogger
	DatabaseDSN string
	db          *sql.DB
}

func NewInDBStorage(lg logger.MyLogger, dsn string) (*InDBStorage, error) {
	lg.Infoln("CREATE NEW DB STORAGE")
	st := InDBStorage{
		DatabaseDSN: dsn,
		Logger:      lg,
	}

	err := st.connect()

	if err != nil {
		return &st, fmt.Errorf("CANT CONNECT TO DB [%v]", err)
	}

	err = st.createSchema()

	if err != nil {
		return &st, fmt.Errorf("CAN'T CREATE SCHEMA [%v]", err)
	}

	return &st, err
}

func (l *InDBStorage) connect() error {
	db, err := sql.Open("pgx", l.DatabaseDSN)

	if err != nil {
		return fmt.Errorf("CANT do sql.open: [%v]", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		return fmt.Errorf("CANT PING DB: [%v]", err)
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
																			   "userid" INT,																			
			                                                                   "shorturl" VARCHAR NOT NULL UNIQUE,
																			   "url" VARCHAR NOT NULL UNIQUE,
																			   "is_deleted" BOOLEAN
																			 )`)
			if err != nil {
				return fmt.Errorf("CAN'T CREATE NEW TABLE repo [%v]", err)
			}
		}

		return err
	}

	return nil
}

func (l *InDBStorage) Get(url string) (string, error) {
	var id int
	var shorturl string
	var isDeleted bool
	row := l.db.QueryRowContext(context.Background(), "SELECT id,url,is_deleted FROM repo WHERE shorturl=$1", url)
	err := row.Scan(&id, &shorturl, &isDeleted)
	if isDeleted {
		return shorturl, ErrRecWasDelete
	}
	return shorturl, err
}

func (l *InDBStorage) GetByURL(url string) (string, error) {
	var shorturl string
	row := l.db.QueryRowContext(context.Background(), "SELECT shorturl FROM repo WHERE url=$1", url)
	err := row.Scan(&shorturl)
	return shorturl, err
}

func (l *InDBStorage) Create(lnkRec LinkRecord) error {
	_, err := l.db.ExecContext(context.Background(), `INSERT INTO repo (userid,shorturl,url) VALUES($1,$2,$3)`,
		lnkRec.UserID,
		lnkRec.ShortURL,
		lnkRec.URL)

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
		return fmt.Errorf("CAN'T PREPARE CONTEXT IN BATCH: [%v]", err)
	}

	for _, lnk := range lnkRecs {
		_, err := stmt.ExecContext(context.Background(), lnk.ShortURL, lnk.URL)

		if err != nil {
			return fmt.Errorf("CAN'T EXEC PREPARED QUERY IN BATCH: [%v]", err)
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

func (l *InDBStorage) Urls(userid int) ([]LinkRecord, error) {
	res := []LinkRecord{}
	rows, err := l.db.QueryContext(context.Background(), `SELECT id,userid,shorturl,url FROM repo WHERE userid=$1`,
		userid)

	if err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("CAN'T EXEC QUERY IN URLs [%v]", err)
		}
	}

	defer rows.Close()

	for rows.Next() {
		lnkRec := LinkRecord{}
		err = rows.Scan(&lnkRec.ID, &lnkRec.UserID, &lnkRec.ShortURL, &lnkRec.URL)
		if err != nil {
			return nil, fmt.Errorf("CAN'T SCAN IN Urls: [%v]", err)
		}
		res = append(res, lnkRec)
	}

	if err = rows.Err(); err != nil {
		return res, fmt.Errorf("ROW ERROR IN Urls: [%v]", err)
	}

	return res, nil
}

func (l *InDBStorage) BatchDel(userid int, urls []string) error {

	tx, err := l.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(context.Background(), "UPDATE repo SET is_deleted=true WHERE userid=$1 AND shorturl=$2")

	if err != nil {
		return fmt.Errorf("CAN'T PREPARE SQL IN BATCH DELETE: [%v]", err)
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	inputCh := l.generatorUrlsDel(doneCh, urls)

	l.Logger.Infoln(userid)

	channels := l.fanOut(doneCh, inputCh, userid, stmt)

	l.fanIn(doneCh, channels...)

	return tx.Commit()
}

func (l *InDBStorage) generatorUrlsDel(doneCh chan struct{}, input []string) chan string {
	inputCh := make(chan string)

	go func() {
		defer close(inputCh)

		for _, url := range input {
			inputCh <- url
		}

	}()

	return inputCh
}

func (l *InDBStorage) fanOut(doneCh chan struct{}, inputCh chan string, userid int, stmt *sql.Stmt) []chan bool {
	numWorkers := 10
	// каналы, в которые отправляются результаты
	channels := make([]chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		channels[i] = l.urlsDel(doneCh, inputCh, userid, stmt)
	}

	// возвращаем слайс каналов
	return channels
}

func (l *InDBStorage) urlsDel(doneCh chan struct{}, inputCh chan string, userid int, stmt *sql.Stmt) chan bool {
	res := make(chan bool)

	go func() {
		defer close(res)

		for url := range inputCh {
			_, err := stmt.ExecContext(context.Background(), userid, url)

			if err != nil {
				l.Logger.Infoln(err)
			}
			select {
			case <-doneCh:
				return
			case res <- err != nil:
			}
		}
	}()

	return res
}

func (l *InDBStorage) fanIn(doneCh chan struct{}, resultChs ...chan bool) chan bool {
	finalCh := make(chan bool)

	var wg sync.WaitGroup

	for _, ch := range resultChs {
		chClosure := ch
		wg.Add(1)

		go func() {

			defer wg.Done()

			// получаем данные из канала
			for data := range chClosure {
				select {
				// выходим из горутины, если канал закрылся
				case <-doneCh:
					return
				case <-chClosure:
					return
				// если не закрылся, отправляем данные в конечный выходной канал
				case finalCh <- data:
				}
			}
		}()
	}

	// ждём завершения всех горутин
	wg.Wait()
	// когда все горутины завершились, закрываем результирующий канал
	close(finalCh)

	return finalCh
}
