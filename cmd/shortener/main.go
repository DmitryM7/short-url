package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/DmitryM7/short-url.git/internal/conf"
	"github.com/DmitryM7/short-url.git/internal/controller"
	"github.com/DmitryM7/short-url.git/internal/repository"
	"go.uber.org/zap"
)

var (
	repo   repository.LinkRepo
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

func main() {
	var errLogger error

	conf.ParseFlags()
	flag.Parse()
	conf.ParseEnv()

	logger, errLogger = zap.NewDevelopment()

	if errLogger != nil {
		panic("CAN'T INIT ZAP LOGGER")
	}

	defer logger.Sync() //nolint:errcheck // unnessesary error checking

	sugar = logger.Sugar()

	repo = repository.NewLinkRepo()

	repo.SavePath = conf.FilePath
	repo.Logger = sugar

	err := repo.Load()

	if err != nil {
		sugar.Infoln("CAN'T LOAD STORAGE FROM FILE")
	}

	r := controller.NewRouter(sugar, repo)

	sugar.Infow("Starting server", "bndAdd", conf.BndAdd)

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
