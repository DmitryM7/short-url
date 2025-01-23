package repository

import (
	"encoding/json"
	"io"
	"os"

	"github.com/DmitryM7/short-url.git/internal/logger"
)

const defFilePerm os.FileMode = 0644

type InFileStorage struct {
	InMemoryStorage
	SavePath string
}

func NewInFileStorage(lg logger.MyLogger, exportFile string) (*InFileStorage, error) {
	lg.Infoln("CREATE NEW IN FILE STORAGE")
	inmem, err := NewInMemoryStorage(lg)
	if err != nil {
		return &InFileStorage{SavePath: exportFile}, err
	}
	return &InFileStorage{
		InMemoryStorage: *inmem,
		SavePath:        exportFile,
	}, nil
}

func (r *InFileStorage) Create(lnkRec LinkRecord) error {
	err := r.InMemoryStorage.Create(lnkRec)

	if err != nil {
		return err
	}

	_, err = r.Unload()

	if err != nil {
		return err
	}

	return nil
}

func (r *InFileStorage) SetSavePath(p string) {
	r.SavePath = p
}

func (r *InFileStorage) Unload() (int, error) {
	j, err := json.Marshal(r.Repo)

	if err != nil {
		return 0, err
	}

	file, err := os.OpenFile(r.SavePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defFilePerm)

	if err != nil {
		r.Logger.Debugln("FIND PATH:" + r.SavePath)
		return 0, err
	}

	defer file.Close()

	return file.Write(j)
}

func (r *InFileStorage) Load() error {
	file, err := os.OpenFile(r.SavePath, os.O_RDONLY|os.O_CREATE, defFilePerm)

	if err != nil {
		r.Logger.Errorln("CANT CREATE AND OPEN STORAGE FILE:" + r.SavePath)
		return err
	}

	defer file.Close()

	buffer, err := io.ReadAll(file)

	if err != nil {
		r.Logger.Errorln("CANT READ STORAGE FROM FILE")
		return err
	}

	if string(buffer) != "" {
		err = json.Unmarshal(buffer, &r.Repo)

		if err != nil {
			r.Logger.Errorln("CANT UNMARSHAL STORAGE BODY:" + string(buffer))
			return err
		}
	} else {
		r.Logger.Infoln("EMPTY BUFFER. PROBABLY FIRST RUN")
	}

	return nil
}

func (r *InFileStorage) Ping() bool {
	return true
}
