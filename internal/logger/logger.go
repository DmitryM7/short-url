package logger

import (
	"go.uber.org/zap"
)

func NewLogger() *zap.SugaredLogger {

	var (
		logger    *zap.Logger
		errLogger error
	)

	logger, errLogger = zap.NewDevelopment()

	if errLogger != nil {
		panic("CAN'T INIT ZAP LOGGER")
	}

	defer logger.Sync() //nolint:errcheck // unnessesary error checking

	return logger.Sugar()
}
