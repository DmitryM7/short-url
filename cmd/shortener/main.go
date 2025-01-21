package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/DmitryM7/short-url.git/internal/conf"
	"github.com/DmitryM7/short-url.git/internal/controller"
	"github.com/DmitryM7/short-url.git/internal/logger"
	"github.com/DmitryM7/short-url.git/internal/repository"
)

func main() {
	lg := logger.NewLogger()

	lg.Infoln("RUN...")

	conf.ParseFlags()
	flag.Parse()
	conf.ParseEnv()

	repoConf := repository.StorageConfig{Logger: lg}

	if conf.DSN != "" {
		repoConf.StorageType = repository.DBType
		repoConf.DatabaseDSN = conf.DSN
	} else {
		repoConf.StorageType = repository.FileType
		repoConf.FilePath = conf.FilePath
	}

	repo, err := repository.NewStorageService(repoConf)

	if err != nil {
		lg.Fatalln("CANT INIT REPO" + fmt.Sprintf("%#v", err))
	}

	r := controller.NewRouter(lg, repo)

	lg.Infoln("Starting server", "bndAdd", conf.BndAdd)

	server := &http.Server{
		Addr:         conf.BndAdd,
		Handler:      r,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	if errServ := server.ListenAndServe(); errServ != nil {
		lg.Fatalw(errServ.Error(), "event", "start server")
	}
}
