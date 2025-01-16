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
	"go.uber.org/zap"
)

var (
	sugar *zap.SugaredLogger
)

func main() {
	sugar = logger.NewLogger()

	sugar.Infoln("RUN...")

	conf.ParseFlags()
	flag.Parse()
	conf.ParseEnv()

	repo := repository.NewLinkRepoDB(sugar, conf.FilePath, conf.DSN)

	err := repo.Load()

	if err != nil {
		sugar.Infoln("CAN'T LOAD STORAGE. " + fmt.Sprintf("%s", err) + " . USE EMPTY REPO.")
	}

	r := controller.NewRouter(sugar, repo)

	sugar.Infoln("Starting server", "bndAdd", conf.BndAdd)

	server := &http.Server{
		Addr:         conf.BndAdd,
		Handler:      r,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	if errServ := server.ListenAndServe(); errServ != nil {
		sugar.Fatalw(errServ.Error(), "event", "start server")
	}
}
