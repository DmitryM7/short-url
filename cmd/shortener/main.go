package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

	CustomResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
		needGZip     bool
	}

	Request struct {
		URL string `json:"url"`
	}

	Response struct {
		Result string `json:"result"`
	}
)

func (r *CustomResponseWriter) isContentTypeNeedZip() bool {
	var needGZip bool = false

	headers := r.Header().Values("Content-type")

	for _, header := range headers {
		if header == "application/json" || header == "text/html" || header == "text/plain" {
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

	if r.needGZip && r.isContentTypeNeedZip() {
		gz, err = gzip.NewWriterLevel(r.ResponseWriter, gzip.BestSpeed)
		defer gz.Close()

		if err != nil {
			size = 0
			err = fmt.Errorf("CANT CREATE GZIP")
		} else {
			sugar.Infoln("DO ZIPPING")
			size, err = gz.Write(b)
		}
	} else {
		size, err = r.ResponseWriter.Write(b)
	}

	r.responseData.size += size
	return size, err
}

func (r *CustomResponseWriter) WriteHeader(statusCode int) {
	if r.needGZip && r.isContentTypeNeedZip() {
		r.Header().Set("Content-encoding", "gzip")
	}
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func getURL(id string) (string, error) {
	if url, err := repo.Get(id); err == nil {
		return url, nil
	}

	return "", errors.New("NO REQUIRED PARAM ID")
}

func actionError(w http.ResponseWriter, e string) {
	sugar.Infoln(e)
	w.WriteHeader(http.StatusBadRequest)
	_, err := w.Write([]byte(e))

	if err != nil {
		slog.Error("CAN'T WRITE ANSWER")
	}
}

func actionCreateURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		actionError(w, "Error read query request body")
		return
	}

	url := string(body)

	if url == "" {
		actionError(w, "Body was send, but empty")
		return
	}

	newURL := repo.CreateAndSave(url)

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

func actionTest(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")

	if id == "" {
		actionError(w, "No required param 'ID' or ID is empty")
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	sugar.Infoln(string(body))

	if err != nil {
		actionError(w, "CAN'T READ BODY")
		return
	}

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, errWrite := w.Write(body)

	if errWrite != nil {
		actionError(w, "CAN'T WRITE BODY")
		return
	}
}
func actionShorten(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		actionError(w, "CAN'T READ BODY FROM REQUEST")
		return
	}

	if string(body) == "" {
		actionError(w, "EMPTY BODY")
		return
	}

	request := Request{}
	response := Response{}

	err = json.Unmarshal(body, &request)

	if err != nil {
		actionError(w, "CAN'T UNMARSHAL JSON BODY.")
		return
	}

	newURL := repo.CreateAndSave(request.URL)

	response.Result = retAdd + "/" + newURL

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)

	res, err := json.Marshal(response)
	if err != nil {
		actionError(w, "CAN'T UNMARSHAL JSON RESULT.")
		return
	}

	_, errRes := w.Write(res)

	if errRes != nil {
		actionError(w, "CAN'T WRITE RESULT BODY.")
		return
	}
}

/****************************************************************************************
 *                                                                                      *
 * Задание по треку «Сервис сокращения URL»                                             *
 * Добавьте поддержку gzip в ваш сервис. Научите его:                                   *
 * Принимать запросы в сжатом формате (с HTTP-заголовком Content-Encoding).             *
 * +Отдавать сжатый ответ клиенту, который поддерживает обработку сжатых ответов         *
 * +(с HTTP-заголовком Accept-Encoding).                                                 *
 * +Функция сжатия должна работать для контента с типами application/json и text/html.   *
 * +Вспомните middleware из урока про HTTP-сервер, это может вам помочь.                 *
 *                                                                                      *
 ****************************************************************************************/
func actionStart(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {

		begTime := time.Now()
		uri := r.RequestURI
		method := r.Method

		responseData := &responseData{
			status: 0,
			size:   0,
		}

		lw := CustomResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
			needGZip:       false,
		}

		acceptEncodings := r.Header.Values("Accept-Encoding")

		for _, encodingLine := range acceptEncodings {

			acceptEncoding := strings.Split(encodingLine, ",")
			for _, encoding := range acceptEncoding {
				if encoding == "gzip" {
					lw.needGZip = true
					break
				}
			}
		}

		sugar.Infoln(r.Header.Get("Content-Encoding"))

		if r.Header.Get("Content-Encoding") == "gzip" {

			buf, err := io.ReadAll(r.Body) // handle the error

			if err != nil {
				actionError(w, "CAN'T CREATE NEW BUFFER")
				return
			}
			readedBody := io.NopCloser(bytes.NewBuffer(buf))

			gz, err := gzip.NewReader(readedBody)

			if err != nil {
				actionError(w, "CAN'T CREATE GZ READER")
				return
			}

			r.Body = gz

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

	defer logger.Sync() //nolint:errcheck // unnessesary error checking

	sugar = logger.Sugar()

	repo = NewLinkRepo()

	r := chi.NewRouter()

	r.Use(actionStart)

	r.Route("/", func(r chi.Router) {
		r.Post("/", actionCreateURL)
		r.Post("/api/shorten", actionShorten)
		r.Get("/{id}", actionRedirect)
		r.Get("/tst", actionTest)
		r.Post("/tst", actionTest)
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
