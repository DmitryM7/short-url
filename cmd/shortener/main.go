package main

import (
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var (
	repo   linkRepo
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

type (
	responseData struct {
		status int
		size   int
	}

	logResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *logResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *logResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func createShortURL(url string) string {
	var shortURL string

	checksum := crc32.Checksum([]byte(url), crc32.MakeTable(crc32.IEEE))

	shortURL = fmt.Sprintf("%08x", checksum)

	return shortURL
}

func getURL(id string) (string, error) {
	if url, err := repo.Get(id); err == nil {
		return url, nil
	}

	return "", errors.New("NO REQUIRED PARAM ID")
}

func actionError(w http.ResponseWriter, e string) {
	w.WriteHeader(http.StatusBadRequest)
	_, err := w.Write([]byte(e))

	if err != nil {
		slog.Error("CAN'T WRITE ANSWER")
	}
}

func actionCreateURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)

	if err != nil {
		actionError(w, "Error read query request body")
		return
	}

	url := string(body)

	if url == "" {
		actionError(w, "Body was send, but empty")
		return
	}

	newURL := createShortURL(url)

	repo.Create(newURL, url)

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, errWrite := w.Write([]byte(retAdd + "/" + newURL))

	if errWrite != nil {
		slog.Error("CANT WRITE DATA TO RESPONSE")
	}
}

func actionRedirect(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")

	if id == "" {
		actionError(w, "No required param 'ID' or ID is empty")
		return
	}

	newURL, err := getURL(id)

	if err != nil {
		actionError(w, "Can't find short url by ID")
		return
	}

	http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
}

func start(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		begTime := time.Now()
		uri := r.RequestURI
		method := r.Method

		responseData := &responseData{
			status: 0,
			size:   0,
		}

		lw := logResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		next.ServeHTTP(&lw, r)

		duration := time.Since(begTime)

		sugar.Infoln(
			"uri", uri,
			"method", method,
			"duration", duration,
			"size", responseData.size,
			"status", responseData.status,
		)
	}
	return http.HandlerFunc(f)
}

func main() {
	var errLogger error

	parseFlags()
	flag.Parse()
	parseEnv()

	logger, errLogger = zap.NewDevelopment()

	if errLogger != nil {
		panic("CAN'T INIT ZAP LOGGER")
	}

	defer logger.Sync()

	sugar = logger.Sugar()

	repo = NewLinkRepo()

	r := chi.NewRouter()

	r.Use(start)

	r.Route("/", func(r chi.Router) {
		r.Post("/", actionCreateURL)
		r.Get("/{id}", actionRedirect)
	})
	/*****************************************************************************************
	  Инкеремент №6
	  Реализуйте логирование сведений о запросах и ответах на сервере для всех эндпоинтов,
	  которые у вас уже есть.
	  * Сведения о запросах должны содержать URI, метод запроса и время, затраченное на его выполнение.
	  Сведения об ответах должны содержать код статуса и размер содержимого ответа.
	  Эту функциональность нужно реализовать через middleware.
	  Используйте один из сторонних пакетов для логирования:
	  github.com/rs/zerolog,
	  go.uber.org/zap,
	  github.com/sirupsen/logrus.
	  Все сообщения логера должны быть на уровне Info.
	********************************************************************************************/
	sugar.Infow("Starting server", "bndAdd", bndAdd)

	server := &http.Server{
		Addr:         bndAdd,
		Handler:      r,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	if errServ := server.ListenAndServe(); errServ != nil {
		sugar.Fatalw(errServ.Error(), "event", "start server")
	}
}
