package controller

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DmitryM7/short-url.git/internal/conf"
	"github.com/DmitryM7/short-url.git/internal/logger"
	"github.com/DmitryM7/short-url.git/internal/models"
	"github.com/DmitryM7/short-url.git/internal/repository"
	"github.com/go-chi/chi"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type (
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

	MyServer struct {
		Logger logger.MyLogger
		Repo   repository.LinkRepoDB
	}
)

func (s *MyServer) actionError(w http.ResponseWriter, e string) {
	s.Logger.Infoln(e)
	w.WriteHeader(http.StatusBadRequest)
	_, err := w.Write([]byte(e))

	if err != nil {
		s.Logger.Error("CAN'T WRITE ANSWER")
	}
}

func (s *MyServer) actionCreateURL(w http.ResponseWriter, r *http.Request) {
	var answerStatus = http.StatusCreated
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		s.actionError(w, "Error read query request body")
		return
	}

	url := string(body)

	if url == "" {
		s.actionError(w, "Body was send, but empty")
		return
	}

	newURL, err := s.Repo.CalcAndCreate(url)

	var perr *pgconn.PgError

	if errors.As(err, &perr) && perr.Code == pgerrcode.UniqueViolation {
		/***********************************************************************
		 * Бесмысленная история для стратегии №2 задания итерации 13           *
		 * (мы можем сделать вставку т. и т. к., мы уже знаем сокращенный URL) *
		 * но чтобы выполнить букву задания                                    *
		 * делаем повторное получение shorturl из БД.                          *
		 ***********************************************************************/
		newURL, err = s.Repo.GetByURL(url)
		if err != nil {
			s.actionError(w, "CAN'T RECEIVE SHORTURL FROM DB")
			return
		}
		answerStatus = http.StatusConflict
	}

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(answerStatus)
	_, errWrite := w.Write([]byte(conf.RetAdd + "/" + newURL))

	if errWrite != nil {
		s.Logger.Errorln("CANT WRITE DATA TO RESPONSE")
	}

	_, err = s.Repo.Unload()

	if err != nil {
		s.Logger.Errorln("CANT SAVE REPO:" + fmt.Sprintf("%s", err))
	}
}

func (s *MyServer) actionRedirect(w http.ResponseWriter, r *http.Request) {
	s.Logger.Debugln("Start Redirect")

	id := strings.TrimPrefix(r.URL.Path, "/")

	if id == "" {
		s.actionError(w, "No required param 'ID' or ID is empty")
		return
	}

	newURL, err := s.Repo.Get(id)

	if err != nil {
		s.actionError(w, "CAN'T GET SHORT LINK FROM REPO")
		return
	}

	http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
}

func (s *MyServer) actionPing(w http.ResponseWriter, r *http.Request) {
	err := s.Repo.Ping()

	if err != nil {
		s.Logger.Infoln("CAN'T OPEN DATABASE CONNECT")
		s.Logger.Infoln(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *MyServer) actionTest(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")

	if id == "" {
		s.actionError(w, "No required param 'ID' or ID is empty")
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	s.Logger.Debugln(string(body))

	if err != nil {
		s.actionError(w, "CAN'T READ BODY")
		return
	}

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, errWrite := w.Write(body)

	if errWrite != nil {
		s.actionError(w, "CAN'T WRITE BODY")
		return
	}
}
func (s *MyServer) actionShorten(w http.ResponseWriter, r *http.Request) {
	var answerStatus = http.StatusCreated
	s.Logger.Debugln("Start Shorten")

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		s.actionError(w, "CAN'T READ BODY FROM REQUEST")
		return
	}

	if string(body) == "" {
		s.actionError(w, "EMPTY BODY")
		return
	}

	request := Request{}
	response := Response{}

	err = json.Unmarshal(body, &request)

	if err != nil {
		s.actionError(w, "CAN'T UNMARSHAL JSON BODY.")
		return
	}

	newURL, err := s.Repo.CalcAndCreate(request.URL)

	var perr *pgconn.PgError

	if errors.As(err, &perr) && perr.Code == pgerrcode.UniqueViolation {
		/***********************************************************************
		 * Бесмысленная история для стратегии №2 задания итерации 13           *
		 * (мы можем сделать вставку т. и т. к., мы уже знаем сокращенный URL) *
		 * но чтобы выполнить букву задания                                    *
		 * делаем повторное получение shorturl из БД.                          *
		 ***********************************************************************/
		newURL, err = s.Repo.GetByURL(request.URL)
		if err != nil {
			s.actionError(w, "CAN'T RECEIVE SHORTURL FROM DB")
			return
		}
		answerStatus = http.StatusConflict
	}

	_, err = s.Repo.Unload()

	if err != nil {
		s.Logger.Errorln("CANT SAVE REPO TO FILE")
	}

	response.Result = conf.RetAdd + "/" + newURL

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(answerStatus)

	res, err := json.Marshal(response)
	if err != nil {
		s.actionError(w, "CAN'T UNMARSHAL JSON RESULT.")
		return
	}

	_, errRes := w.Write(res)

	if errRes != nil {
		s.actionError(w, "CAN'T WRITE RESULT BODY.")
		return
	}
}

func (s *MyServer) actionBatch(w http.ResponseWriter, r *http.Request) {
	var batchError error = nil

	body, err := io.ReadAll(r.Body)

	if err != nil {
		s.actionError(w, "CAN'T READ BODY FROM REQUEST")
		return
	}

	defer r.Body.Close()

	if string(body) == "" {
		s.actionError(w, "EMPTY BODY")
		return
	}

	s.Logger.Debugln(string(body))

	input := []RequestShortenBatchUnit{}
	output := []ResponseShortenBatchUnit{}

	err = json.Unmarshal(body, &input)

	if err != nil {
		s.actionError(w, "CAN'T UNMARSHAL JSON BODY.")
		return
	}

	for _, v := range input {
		shorturl, err := s.Repo.CalcAndCreateManualCommit(v.OriginalURL)
		if err != nil {
			batchError = err
			break
		}

		output = append(output, ResponseShortenBatchUnit{
			CorrelationID: v.CorrelationID,
			ShortURL:      conf.RetAdd + "/" + shorturl,
		})
	}

	if batchError != nil {
		s.Repo.RollBack()
		s.actionError(w, "CAN'T BATCH LOAD"+fmt.Sprintf("%s", batchError))
	}

	s.Repo.Commit()

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)

	res, err := json.Marshal(output)
	if err != nil {
		s.actionError(w, "CAN'T UNMARSHAL JSON RESULT.")
		return
	}

	_, errRes := w.Write(res)

	if errRes != nil {
		s.actionError(w, "CAN'T WRITE RESULT BODY.")
		return
	}
}

func (s *MyServer) actionStart(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		s.Logger.Debugln(fmt.Sprintf("Req: %s %s\n", r.Host, r.URL.Path))

		begTime := time.Now()
		uri := r.RequestURI
		method := r.Method

		responseData := &models.ResponseData{
			Status: 0,
			Size:   0,
		}

		lw := models.CustomResponseWriter{
			ResponseWriter: w,
			ResponseData:   responseData,
			NeedGZip:       false,
		}

		acceptEncodings := r.Header.Values("Accept-Encoding")

		for _, encodingLine := range acceptEncodings {
			acceptEncoding := strings.Split(encodingLine, ",")
			for _, encoding := range acceptEncoding {
				if encoding == "gzip" {
					lw.NeedGZip = true
					break
				}
			}
		}

		s.Logger.Debugln(r.Header.Get("Content-Encoding"))

		if r.Header.Get("Content-Encoding") == "gzip" {
			buf, err := io.ReadAll(r.Body) // handle the error

			if err != nil {
				s.actionError(w, "CAN'T CREATE NEW BUFFER")
				return
			}
			readedBody := io.NopCloser(bytes.NewBuffer(buf))

			gz, err := gzip.NewReader(readedBody)

			if err != nil {
				s.actionError(w, "CAN'T CREATE GZ READER")
				return
			}

			r.Body = gz
		}
		next.ServeHTTP(&lw, r)

		duration := time.Since(begTime)

		s.Logger.Infoln(
			"uri", uri,
			"method", method,
			"duration", duration,
			"size", responseData.Size,
			"status", responseData.Status,
		)
	}
	return http.HandlerFunc(f)
}

func NewServer(log logger.MyLogger, repo repository.LinkRepoDB) (*MyServer, error) {
	return &MyServer{
		Logger: log,
		Repo:   repo,
	}, nil
}

func NewRouter(log logger.MyLogger, repo repository.LinkRepoDB) *chi.Mux {
	R := chi.NewRouter()
	server, err := NewServer(log, repo)

	if err != nil {
		log.Errorln("CAN'T CREATE SERVER")
	}

	R.Use(server.actionStart)

	R.Route("/", func(r chi.Router) {
		r.Post("/", server.actionCreateURL)
		r.Post("/api/shorten", server.actionShorten)
		r.Post("/api/shorten/batch", server.actionBatch)
		r.Get("/{id}", server.actionRedirect)
		r.Get("/ping", server.actionPing)
		r.Get("/tst", server.actionTest)
		r.Post("/tst", server.actionTest)
	})

	return R
}
