package models

import (
	"compress/gzip"
	"fmt"
	"net/http"

	"github.com/DmitryM7/short-url.git/internal/logger"
)

type (
	CustomResponseWriter struct {
		http.ResponseWriter
		ResponseData *ResponseData
		NeedGZip     bool
		Logger       logger.MyLogger
	}

	ResponseData struct {
		Status int
		Size   int
		Logger logger.MyLogger
	}
)

func (r *CustomResponseWriter) isContentTypeNeedZip() bool {
	needGZip := false

	headers := r.Header().Values("Content-type")

	for _, header := range headers {
		if header == "application/json" || header == "text/html" {
			needGZip = true
		}
	}
	return needGZip
}
func (r *CustomResponseWriter) Write(b []byte) (int, error) {
	var (
		size int
		err  error
		gz   *gzip.Writer
	)

	if r.NeedGZip && r.isContentTypeNeedZip() {
		gz, err = gzip.NewWriterLevel(r.ResponseWriter, gzip.BestSpeed)

		if err != nil {
			size = 0
			err = fmt.Errorf("CANT CREATE GZIP")
		} else {
			r.Logger.Debugln("DO ZIPPING")
			size, err = gz.Write(b)
		}
		defer gz.Close()
	} else {
		size, err = r.ResponseWriter.Write(b)
	}

	r.ResponseData.Size += size
	return size, err
}

func (r *CustomResponseWriter) WriteHeader(statusCode int) {
	if r.NeedGZip && r.isContentTypeNeedZip() {
		r.Header().Set("Content-encoding", "gzip")
	}
	r.ResponseWriter.WriteHeader(statusCode)
	r.ResponseData.Status = statusCode
}
