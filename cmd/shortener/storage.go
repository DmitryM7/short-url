package main

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

const (
	repoLength  int64       = 100
	defFilePerm os.FileMode = 0644
)

type linkRepo struct {
	repo     map[string]string
	SavePath string
}

func NewLinkRepo() linkRepo {
	return linkRepo{repo: make(map[string]string, repoLength)}
}

func (r *linkRepo) Create(h, l string) {
	r.repo[h] = l
}

func (r *linkRepo) Get(h string) (string, error) {
	l, err := r.repo[h]

	if !err {
		return "", fmt.Errorf("CAN'T FIND LINK BY HASH")
	}

	return l, nil
}

func (r *linkRepo) SetSavePath(p string) {
	r.SavePath = p
}

func (r *linkRepo) CreateAndSave(url string) string {
	var shortURL string

	checksum := crc32.Checksum([]byte(url), crc32.MakeTable(crc32.IEEE))

	shortURL = fmt.Sprintf("%08x", checksum)

	r.Create(shortURL, url)

	return shortURL
}

func (r *linkRepo) Unload() (int, error) {
	j, err := json.Marshal(r.repo)

	if err != nil {
		return 0, err
	}

	file, err := os.OpenFile(r.SavePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defFilePerm)

	if err != nil {
		sugar.Infoln("FIND PATH:" + r.SavePath)
		return 0, err
	}

	defer file.Close()

	return file.Write(j)
}
func (r *linkRepo) Load() error {
	file, err := os.OpenFile(r.SavePath, os.O_RDONLY|os.O_CREATE, defFilePerm)

	if err != nil {
		sugar.Errorln("CANT OPEN STORAGE FILE:" + r.SavePath)
		return err
	}

	defer file.Close()

	buffer, err := io.ReadAll(file)

	if err != nil {
		sugar.Errorln("CANT READ STORAGE FROM FILE")
		return err
	}

	if string(buffer) != "" {
		err = json.Unmarshal(buffer, &r.repo)

		if err != nil {
			sugar.Errorln("CANT UNMARSHAL STORAGE BODY:" + string(buffer))
			return err
		}
	} else {
		sugar.Infoln("EMPTY BUFFER. PROBABLY FIRST RUN")
	}

	return nil
}
