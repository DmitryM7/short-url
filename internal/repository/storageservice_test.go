package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/DmitryM7/short-url.git/internal/logger"
)

func BenchmarkGet(b *testing.B) {
	lg := logger.NewLogger()
	repoConf := StorageConfig{Logger: lg}
	repoConf.StorageType = FileType
	repoConf.FilePath = "./repo.json"

	storage, err := NewStorageService(repoConf)

	if err != nil {
		lg.Errorln(err)
	}

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_, err := storage.Get(ctx, "qwer")

		if err != nil {
			lg.Errorln(err)
		}
	}
}

func ExampleNewStorageService() {
	lg := logger.NewLogger()
	repoConf := StorageConfig{Logger: lg}
	repoConf.StorageType = FileType
	repoConf.FilePath = "./repo.json"
	ctx := context.Background()

	storage, err := NewStorageService(repoConf)

	if err != nil {
		lg.Errorln(err)
	}

	short_url, err := storage.Create(ctx, LinkRecord{UserID: 1, URL: "www.ya.ru"})

	fmt.Println(short_url)

}
