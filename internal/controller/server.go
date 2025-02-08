package controller

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

const maxDBExecuteTime = 60 * time.Second

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
		Logger        logger.MyLogger
		Repo          repository.StorageService
		userIDCounter int
		secretKey     string
	}
)

const CookieLiveMinutes = 25

func (s *MyServer) actionError(w http.ResponseWriter, e string) {
	s.Logger.Infoln(e)
	w.WriteHeader(http.StatusBadRequest)
	_, err := w.Write([]byte(e))

	if err != nil {
		s.Logger.Error("CAN'T WRITE ANSWER")
	}
}

func (s *MyServer) actionCreateURL(w http.ResponseWriter, r *http.Request) {
	s.Logger.Debugln("Start ActionCreateUrl")

	var answerStatus = http.StatusCreated
	var userid int

	ctx, cancel := context.WithTimeout(context.Background(), maxDBExecuteTime)

	defer cancel()

	userid, err := s.getUser(r)

	if err != nil {
		userid, err = s.sendAuthToken(w)

		if err != nil {
			s.actionError(w, "AUTH NEED BUT CAN'T:"+fmt.Sprintf("%s", err))
			return
		}
	}

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

	lnkRec := repository.LinkRecord{
		UserID: userid,
		URL:    url,
	}

	newURL, err := s.Repo.Create(ctx, lnkRec)

	var perr *pgconn.PgError

	if errors.As(err, &perr) && perr.Code == pgerrcode.UniqueViolation {
		/***********************************************************************
		 * Бесмысленная история для стратегии №2 задания итерации 13           *
		 * (мы можем сделать вставку т. и т. к., мы уже знаем сокращенный URL) *
		 * но чтобы выполнить букву задания                                    *
		 * делаем повторное получение shorturl из БД.                          *
		 ***********************************************************************/
		newURL, err = s.Repo.GetByURL(ctx, url)
		if err != nil {
			s.actionError(w, "CAN'T RECEIVE SHORTURL FROM DB")
			return
		}
		answerStatus = http.StatusConflict
	}

	s.Logger.Infoln("CURR USER IS = " + strconv.Itoa(userid))
	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(answerStatus)
	_, errWrite := w.Write([]byte(conf.RetAdd + "/" + newURL))

	if errWrite != nil {
		s.Logger.Errorln("CANT WRITE DATA TO RESPONSE")
	}

	if err != nil {
		s.Logger.Errorln("CANT SAVE REPO:" + fmt.Sprintf("%s", err))
	}
}

func (s *MyServer) actionRedirect(w http.ResponseWriter, r *http.Request) {
	s.Logger.Debugln("Start Redirect")

	ctx, cancel := context.WithTimeout(r.Context(), maxDBExecuteTime)

	defer cancel()

	id := strings.TrimPrefix(r.URL.Path, "/")

	if id == "" {
		s.actionError(w, "No required param 'ID' or ID is empty")
		return
	}

	newURL, err := s.Repo.Get(ctx, id)

	if err != nil {
		if errors.Is(err, repository.ErrRecWasDelete) {
			w.WriteHeader(http.StatusGone)
			return
		}

		s.actionError(w, fmt.Sprintf("CAN'T GET SHORT LINK FROM REPO: [%v]", err))
		return
	}

	http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
}

func (s *MyServer) actionPing(w http.ResponseWriter, r *http.Request) {
	if !s.Repo.Ping() {
		s.Logger.Infoln("NO DATABASE PING")
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

	ctx, cancel := context.WithTimeout(r.Context(), maxDBExecuteTime)

	defer cancel()

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

	lnkRec := repository.LinkRecord{
		UserID: 100,
		URL:    request.URL,
	}

	newURL, err := s.Repo.Create(ctx, lnkRec)

	var perr *pgconn.PgError

	if errors.As(err, &perr) && perr.Code == pgerrcode.UniqueViolation {
		/***********************************************************************
		 * Бесмысленная история для стратегии №2 задания итерации 13           *
		 * (мы можем сделать вставку т. и т. к., мы уже знаем сокращенный URL) *
		 * но чтобы выполнить букву задания                                    *
		 * делаем повторное получение shorturl из БД.                          *
		 ***********************************************************************/
		newURL, err = s.Repo.GetByURL(ctx, request.URL)
		if err != nil {
			s.actionError(w, "CAN'T RECEIVE SHORTURL FROM DB")
			return
		}
		answerStatus = http.StatusConflict
	}

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
	s.Logger.Debugln("Start Batch")
	body, err := io.ReadAll(r.Body)

	ctx, cancel := context.WithTimeout(r.Context(), maxDBExecuteTime)

	defer cancel()

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

	lnkRecs := []repository.LinkRecord{}

	for _, v := range input {
		lnkRecs = append(lnkRecs, repository.LinkRecord{URL: v.OriginalURL, CorrelationID: v.CorrelationID})
	}

	lnkResRecs, err := s.Repo.BatchCreate(ctx, lnkRecs)

	if err != nil {
		s.actionError(w, "CANT SAVE DATA IN REPO")
		return
	}

	for _, v := range lnkResRecs {
		output = append(output, ResponseShortenBatchUnit{
			CorrelationID: v.CorrelationID,
			ShortURL:      conf.RetAdd + "/" + v.ShortURL,
		})
	}

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

func (s *MyServer) actionAPIUrls(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), maxDBExecuteTime)

	defer cancel()

	userid, err := s.getUser(r)

	if err != nil {
		userid, err = s.sendAuthToken(w)

		if err != nil {
			s.actionError(w, "AUTH NEED BUT CAN'T:"+fmt.Sprintf("%s", err))
			return
		}
	}

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	s.Logger.Infoln("CURR USER IS = " + strconv.Itoa(userid))
	lnkRecords, err := s.Repo.Urls(ctx, userid)

	if err != nil {
		s.actionError(w, "CAN'T GET URLS")
		return
	}

	if len(lnkRecords) == 0 {
		s.Logger.Debugln("CAN'T FIND LINKS FOR USER")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Добавляем к короткому адресу указание текущего хоста
	for k := range lnkRecords {
		lnkRecords[k].ShortURL = conf.RetAdd + "/" + lnkRecords[k].ShortURL
	}

	answ, err := json.Marshal(&lnkRecords)

	if err != nil {
		s.actionError(w, "CAN'T MARSHAL ANSWER")
		return
	}

	s.Logger.Infoln("JSON:" + string(answ))
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(answ)

	if err != nil {
		s.actionError(w, "CAN'T WRITE ANSWER TO BODY")
		return
	}
}

func (s *MyServer) actionAPIUrlsDelete(w http.ResponseWriter, r *http.Request) {
	s.Logger.Infoln("URLS DELETE START")

	//ctx, cancel := context.WithTimeout(context.Background(), maxDBExecuteTime)
	ctx := context.Background()

	//defer cancel()

	userid, err := s.getUser(r)

	s.Logger.Infoln("URLS DELETE USER IS CHECKED")

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	s.Logger.Infoln("CURR USER IS = " + strconv.Itoa(userid))

	body, err := io.ReadAll(r.Body)

	if err != nil {
		s.actionError(w, "CAN'T READ BODY")
		return
	}
	defer r.Body.Close()

	if string(body) == "" {
		s.actionError(w, "BODY IS EMPTY")
		return
	}

	s.Logger.Infoln("URLS DELETE:" + string(body))

	idsToDel := []string{}

	err = json.Unmarshal(body, &idsToDel)

	if err != nil {
		s.actionError(w, "CAN'T LOAD BODY TO SLICE.")
	}

	go func() {
		err = s.Repo.BatchDel(ctx, userid, idsToDel)
		if err != nil {
			s.Logger.Errorln(err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (s *MyServer) sendAuthToken(w http.ResponseWriter) (int, error) {
	userid := s.userIDCounter
	jwtProvider := NewJwtProvider(time.Hour, s.secretKey)

	tokenStr, err := jwtProvider.GetStr(s.secretKey, userid)

	if err != nil {
		return 0, fmt.Errorf("CAN'T CREATE JWT TOKEN: [%v]", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenStr,
		Expires: time.Now().Add(CookieLiveMinutes * time.Minute),
	})

	s.userIDCounter++

	return userid, nil
}

func (s *MyServer) getUser(r *http.Request) (int, error) {
	cookie, err := r.Cookie("token")

	if err != nil {
		s.Logger.Infoln("NO COOKIE")
		return 0, fmt.Errorf("CAN'T READ COOKIE [%v]", err)
	}

	jwtProvider := NewJwtProvider(time.Hour, s.secretKey)

	userid, err := jwtProvider.GetUserID(cookie.Value)

	if err != nil {
		return userid, fmt.Errorf("CAN'T GETUSER ID [%v]", err)
	}

	return userid, nil
}

func (s *MyServer) actionStart(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		s.Logger.Debugln(fmt.Sprintf("Req: %s %s", r.Host, r.URL.Path))

		begTime := time.Now()
		uri := r.RequestURI
		method := r.Method

		responseData := &models.ResponseData{
			Status: 0,
			Size:   0,
			Logger: s.Logger,
		}

		lw := models.CustomResponseWriter{
			ResponseWriter: w,
			ResponseData:   responseData,
			NeedGZip:       false,
			Logger:         s.Logger,
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

		s.Logger.Debugln(
			"uri", uri,
			"method", method,
			"duration", duration,
			"size", responseData.Size,
			"status", responseData.Status,
		)
	}
	return http.HandlerFunc(f)
}

func NewServer(log logger.MyLogger, repo repository.StorageService) (*MyServer, error) {
	b := make([]byte, 2)
	_, err := rand.Read(b)
	if err != nil {
		return &MyServer{}, err
	}

	return &MyServer{
		Logger:    log,
		Repo:      repo,
		secretKey: "KEY_FOR_SECRET",
		//userIDCounter: int(time.Now().Unix()),
		userIDCounter: int(b[0] + b[1]),
	}, nil
}

func NewRouter(log logger.MyLogger, repo repository.StorageService) *chi.Mux {
	R := chi.NewRouter()
	server, err := NewServer(log, repo)

	if err != nil {
		log.Errorln("CAN'T CREATE SERVER")
	}

	R.Use(server.actionStart)

	R.Route("/", func(r chi.Router) {
		r.Route("/api", func(r chi.Router) {
			r.Post("/shorten", server.actionShorten)
			r.Post("/shorten/batch", server.actionBatch)
			r.Get("/user/urls", server.actionAPIUrls)
			r.Delete("/user/urls", server.actionAPIUrlsDelete)
		})
		r.Post("/", server.actionCreateURL)
		r.Get("/{id}", server.actionRedirect)
		r.Get("/ping", server.actionPing)
		r.Get("/tst", server.actionTest)
		r.Post("/tst", server.actionTest)
	})

	return R
}
