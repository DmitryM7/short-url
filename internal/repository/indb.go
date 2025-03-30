package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/DmitryM7/short-url.git/internal/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type (
	// BatchDelMessage - структура обмена информацией между гоурутинами при пакетном удалении
	BatchDelMessage struct {
		Person int
		URL    string
	}

	// InDBStorage - хранилище в памяти
	InDBStorage struct {
		Logger      logger.MyLogger
		DatabaseDSN string
		db          *sql.DB

		cdata chan BatchDelMessage
		cend  chan int
		tx    *sql.Tx
	}
)

// NewInDBStorage - конструктор хранилища в памяти

func NewInDBStorage(lg logger.MyLogger, dsn string) (*InDBStorage, error) {
	lg.Infoln("CREATE NEW DB STORAGE")
	st := InDBStorage{
		DatabaseDSN: dsn,
		Logger:      lg,
		tx:          nil,
	}

	err := st.connect()

	if err != nil {
		return &st, fmt.Errorf("CANT CONNECT TO DB [%v]", err)
	}

	err = st.createSchema()

	if err != nil {
		return &st, fmt.Errorf("CAN'T CREATE SCHEMA [%v]", err)
	}

	st.cdata = make(chan BatchDelMessage)
	st.cend = make(chan int)

	go st.FlowDel(context.Background())

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

	row := l.db.QueryRowContext(context.Background(), `SELECT schemaname 
	                                                   FROM pg_stat_user_tables 
											           WHERE relname LIKE 'repo'`)

	err := row.Scan(&tableName)

	if err != nil {
		if err == sql.ErrNoRows {
			l.Logger.Debugln("TABLE repo NOT EXIST. CREATE IT.")
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

func (l *InDBStorage) Get(ctx context.Context, url string) (string, error) {
	var id int
	var shorturl string
	var isDeleted *bool
	row := l.db.QueryRowContext(ctx, "SELECT id,url,is_deleted FROM repo WHERE shorturl=$1", url)
	err := row.Scan(&id, &shorturl, &isDeleted)

	if err != nil {
		return "", fmt.Errorf("CAN'T GET RECORD: [%v]", err)
	}

	if isDeleted != nil && *isDeleted {
		return shorturl, ErrRecWasDelete
	}

	return shorturl, err
}

func (l *InDBStorage) GetByURL(ctx context.Context, url string) (string, error) {
	var shorturl string
	row := l.db.QueryRowContext(ctx, "SELECT shorturl FROM repo WHERE url=$1", url)
	err := row.Scan(&shorturl)
	return shorturl, err
}

func (l *InDBStorage) Create(ctx context.Context, lnkRec LinkRecord) error {
	_, err := l.db.ExecContext(ctx, `INSERT INTO repo (userid,shorturl,url) VALUES($1,$2,$3)`,
		lnkRec.UserID,
		lnkRec.ShortURL,
		lnkRec.URL)

	if err != nil {
		return err
	}
	return nil
}

func (l *InDBStorage) BatchCreate(ctx context.Context, lnkRecs []LinkRecord) error {
	tx, err := l.db.Begin()

	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO repo (shorturl,url) VALUES($1,$2)")

	if err != nil {
		return fmt.Errorf("CAN'T PREPARE CONTEXT IN BATCH: [%v]", err)
	}

	for _, lnk := range lnkRecs {
		_, err := stmt.ExecContext(ctx, lnk.ShortURL, lnk.URL)

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

func (l *InDBStorage) Urls(ctx context.Context, userid int) ([]LinkRecord, error) {
	res := []LinkRecord{}
	rows, err := l.db.QueryContext(ctx,
		`SELECT id,userid,shorturl,url FROM repo WHERE userid=$1`,
		userid)

	if err != nil {
		return nil, fmt.Errorf("CAN'T EXEC QUERY IN URLs [%v]", err)
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

func (l *InDBStorage) FlowDel(ctx context.Context) {
	var (
		err, err0 error
		stmt      *sql.Stmt
	)

	hasErrorInPacket := false

	for {
		select {
		case message := <-l.cdata:
			if l.tx == nil {
				l.tx, err = l.db.Begin()
				hasErrorInPacket = false

				if err != nil {
					l.Logger.Errorln("CAN'T OPEN TRANSACTION:" + err.Error())
				}

				stmt, err0 = l.tx.PrepareContext(ctx, "UPDATE repo SET is_deleted=true WHERE userid=$1 AND shorturl=$2")

				if err0 != nil {
					l.Logger.Errorln("CAN'T PREPARE CONTEXT: ", err.Error())
					hasErrorInPacket = true
				}
			}

			l.Logger.Infoln("Del id:" + message.URL)

			_, err1 := stmt.ExecContext(ctx, message.Person, message.URL)

			if err1 != nil {
				l.Logger.Errorln("CAN'T EXEC CONTEXT:" + err.Error())
				hasErrorInPacket = true
			}
		case <-l.cend:
			if l.tx != nil {
				if !hasErrorInPacket {
					l.Logger.Infoln("COMMIT")
					err := l.tx.Commit()
					if err != nil {
						l.Logger.Errorln("CAN'T COMMIT TRANSACTION: " + err.Error())
					}
					l.tx = nil
				} else {
					l.Logger.Infoln("ROLLBACK")

					err := l.tx.Rollback()
					if err != nil {
						l.Logger.Errorln("CAN'T ROLLBACK TRANSACTION: " + err.Error())
					}
					l.tx = nil
				}
			}
		}
	}
}

func (l *InDBStorage) BatchDel(ctx context.Context, userid int, urls []string) {
	for _, url := range urls {
		message := BatchDelMessage{
			Person: userid,
			URL:    url,
		}

		l.cdata <- message
	}

	l.cend <- userid
}
