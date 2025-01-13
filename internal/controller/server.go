package controller

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DmitryM7/short-url.git/internal/conf"
	"github.com/DmitryM7/short-url.git/internal/repository"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

var (
	R      *chi.Mux
	Logger *zap.SugaredLogger
	Repo   repository.LinkRepoDB
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

	RequestShortenBatchUnit struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}

	ResponseShortenBatchUnit struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
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

	if r.needGZip && r.isContentTypeNeedZip() {
		gz, err = gzip.NewWriterLevel(r.ResponseWriter, gzip.BestSpeed)

		if err != nil {
			size = 0
			err = fmt.Errorf("CANT CREATE GZIP")
		} else {
			Logger.Debugln("DO ZIPPING")
			size, err = gz.Write(b)
		}
		defer gz.Close()
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

func actionError(w http.ResponseWriter, e string) {
	Logger.Infoln(e)
	w.WriteHeader(http.StatusBadRequest)
	_, err := w.Write([]byte(e))

	if err != nil {
		Logger.Error("CAN'T WRITE ANSWER")
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

	newURL := Repo.CalcAndCreate(url)

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, errWrite := w.Write([]byte(conf.RetAdd + "/" + newURL))

	if errWrite != nil {
		Logger.Errorln("CANT WRITE DATA TO RESPONSE")
	}

	_, err = Repo.Unload()

	if err != nil {
		Logger.Errorln("CANT SAVE REPO:" + fmt.Sprintf("%s", err))
	}
}

func actionRedirect(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")

	if id == "" {
		actionError(w, "No required param 'ID' or ID is empty")
		return
	}

	newURL, err := Repo.Get(id)

	if err != nil {
		actionError(w, "CAN'T GET SHORT LINK FROM REPO")
		return
	}

	http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
}

func actionPing(w http.ResponseWriter, r *http.Request) {

	err := Repo.Ping()

	if err != nil {
		Logger.Infoln("CAN'T OPEN DATABASE CONNECT")
		Logger.Infoln(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

}

func actionTest(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")

	if id == "" {
		actionError(w, "No required param 'ID' or ID is empty")
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	Logger.Debugln(string(body))

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

	newURL := Repo.CalcAndCreate(request.URL)

	_, err = Repo.Unload()

	if err != nil {
		Logger.Errorln("CANT SAVE REPO TO FILE")
	}

	response.Result = conf.RetAdd + "/" + newURL

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

func actionBatch(w http.ResponseWriter, r *http.Request) {
	var batchError error = nil

	body, err := io.ReadAll(r.Body)

	if err != nil {
		actionError(w, "CAN'T READ BODY FROM REQUEST")
		return
	}

	defer r.Body.Close()

	if string(body) == "" {
		actionError(w, "EMPTY BODY")
		return
	}

	input := []RequestShortenBatchUnit{}
	output := []ResponseShortenBatchUnit{}

	err = json.Unmarshal(body, &input)

	if err != nil {
		actionError(w, "CAN'T UNMARSHAL JSON BODY.")
		return
	}

	for _, v := range input {

		shorturl, err := Repo.CalcAndCreateManualCommit(v.OriginalURL)
		if err != nil {
			batchError = err
			break
		}

		output = append(output, ResponseShortenBatchUnit{
			CorrelationID: v.CorrelationID,
			ShortURL:      shorturl,
		})

	}

	if batchError != nil {
		Repo.RollBack()
		actionError(w, "CAN'T BATCH LOAD"+fmt.Sprintf("%s", batchError))
	}

	Repo.Commit()

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)

	res, err := json.Marshal(output)
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

func actionStart(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {

		Logger.Debugln(fmt.Sprintf("Req: %s %s\n", r.Host, r.URL.Path))

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

		Logger.Debugln(r.Header.Get("Content-Encoding"))

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

		Logger.Infoln(
			"uri", uri,
			"method", method,
			"duration", duration,
			"size", responseData.size,
			"status", responseData.status,
		)
	}
	return http.HandlerFunc(f)
}

func NewRouter(logger *zap.SugaredLogger, repo repository.LinkRepoDB) *chi.Mux {

	Logger = logger
	Repo = repo

	R := chi.NewRouter()

	R.Use(actionStart)

	R.Route("/", func(r chi.Router) {
		r.Post("/", actionCreateURL)
		r.Post("/api/shorten", actionShorten)
		r.Post("/api/shorten/batch", actionBatch)
		r.Get("/{id}", actionRedirect)
		r.Get("/ping", actionPing)
		r.Get("/tst", actionTest)
		r.Post("/tst", actionTest)
	})

	return R
}
