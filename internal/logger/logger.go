package logger

import (
	"go.uber.org/zap"
)

type MyLogger struct {
	*zap.SugaredLogger
}

func NewLogger() MyLogger {
	var (
		logger    *zap.Logger
		errLogger error
	)

	logger, errLogger = zap.NewDevelopment()

	if errLogger != nil {
		panic("CAN'T INIT ZAP LOGGER")
	}

	defer logger.Sync() //nolint:errcheck // unnessesary error checking

	return MyLogger{SugaredLogger: logger.Sugar()}
}
