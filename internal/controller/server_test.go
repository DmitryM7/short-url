package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DmitryM7/short-url.git/internal/conf"
	"github.com/DmitryM7/short-url.git/internal/logger"
	mock_repository "github.com/DmitryM7/short-url.git/internal/mocks"
	"github.com/DmitryM7/short-url.git/internal/repository"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var Logger logger.MyLogger
var Repo repository.IStorage

func init() { //nolint: gochecknoinits //see chapter "Setting Up Test Data" in https://www.bytesizego.com/blog/init-function-golang#:~:text=Reasons%20to%20Avoid%20Using%20the%20init%20Function%20in%20Go&text=Since%20it%20runs%20automatically%2C%20any,state%20changes%20without%20explicit%20calls
	conf.ParseFlags()

	Logger = logger.NewLogger()
}

func TestMain(m *testing.M) {
	var err error
	flag.Parse()
	conf.ParseEnv()

	repoConf := repository.StorageConfig{Logger: Logger}

	if conf.DSN != "" {
		repoConf.StorageType = repository.DBType
		repoConf.DatabaseDSN = conf.DSN
	} else {
		repoConf.StorageType = repository.FileType
		repoConf.FilePath = conf.FilePath
	}

	Repo, err = repository.NewStorageService(repoConf)

	if err != nil {
		Logger.Fatalln("CAN'T CREATE REPO")
	}

	os.Exit(m.Run())
}

func TestActionCreateURL(t *testing.T) {
	Logger.Infoln("Запустился TestActionCreateUrl")

	type args struct {
		method string
		url    string
		body   string
	}

	type want struct {
		statusCode int
		body       string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Bad method",
			args: args{
				method: http.MethodGet,
				url:    "/",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "Bad url",
			args: args{
				method: http.MethodPost,
				url:    "/123",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "GOOD",
			args: args{
				method: http.MethodPost,
				url:    "/",
				body:   "www.ya.ru",
			},
			want: want{
				statusCode: http.StatusCreated,
				body:       "http://localhost:8080/b8da4f2d",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r *http.Request

			if tt.args.body != "" {
				Logger.Infoln("with body")
				r = httptest.NewRequest(tt.args.method, tt.args.url, strings.NewReader(tt.args.body))
			} else {
				r = httptest.NewRequest(tt.args.method, tt.args.url, nil)
			}

			w := httptest.NewRecorder()
			server, err := NewServer(Logger, Repo)
			assert.Nil(t, err, "NewServer create with error")
			server.actionCreateURL(w, r)
			res := w.Result()
			defer res.Body.Close()

			answer, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			if tt.want.body != "" {
				assert.Equal(t, tt.want.body, string(answer))
			}
		})
	}
}

func TestActionRedirect(t *testing.T) {
	ctx := context.Background()
	lnkRec := repository.LinkRecord{
		URL: "www.ya.ru",
	}
	_, err := Repo.Create(ctx, lnkRec)

	if err != nil {
		Logger.Fatalln("CAN'T CREATE RECORD")
	}

	type args struct {
		method string
		url    string
	}

	type want struct {
		statusCode int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Bad method",
			args: args{
				method: http.MethodPost,
				url:    "/",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "Bad url",
			args: args{
				method: http.MethodGet,
				url:    "/",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "GOOD",
			args: args{
				method: http.MethodGet,
				url:    "/b8da4f2d",
			},
			want: want{
				statusCode: http.StatusTemporaryRedirect,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.args.method, tt.args.url, nil)
			w := httptest.NewRecorder()
			server, err := NewServer(Logger, Repo)
			assert.Nil(t, err, "NewServer create with error")
			server.actionRedirect(w, r)
			res := w.Result()
			body, _ := io.ReadAll(res.Body)
			t.Log(string(body))
			defer res.Body.Close()
		})
	}
}

func TestActionShorten(t *testing.T) {
	type (
		Args struct {
			URL  string
			Body string
		}
		Want struct {
			Response   string
			StatusCode int
		}
		Tests struct {
			Name string
			Args Args
			Want Want
		}
	)

	tests := []Tests{
		{
			Name: "BAD_EMPTY_REQUEST",
			Args: Args{
				URL: "/api/shorten",
			},
			Want: Want{
				StatusCode: 400,
			},
		},
		{
			Name: "GOOD",
			Args: Args{
				URL:  "/api/shorten",
				Body: "{\"url\": \"https://practicum.yandex.ru\"} ",
			},
			Want: Want{
				StatusCode: 201,
				Response:   "{\"result\":\"http://localhost:8080/ba980180\"}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var r *http.Request
			if tt.Args.Body != "" {
				body := strings.NewReader(tt.Args.Body)
				r = httptest.NewRequest(http.MethodPost, tt.Args.URL, body)
			} else {
				r = httptest.NewRequest(http.MethodPost, tt.Args.URL, nil)
			}

			w := httptest.NewRecorder()

			server, err := NewServer(Logger, Repo)
			assert.Nil(t, err, "NewServer create with error")

			server.actionShorten(w, r)
			res := w.Result()
			assert.Equal(t, tt.Want.StatusCode, res.StatusCode)

			if tt.Want.Response != "" {
				b, e := io.ReadAll(res.Body)
				defer res.Body.Close()
				require.NoError(t, e, "CAN'T READ BODY")
				res := Response{}
				e = json.Unmarshal(b, &res)
				require.NoError(t, e, "CAN'T UNMARSHAL")
				assert.Equal(t, tt.Want.Response, string(b))
			}
		})
	}
}

func TestActionBatch(t *testing.T) {

	input :=
		[]RequestShortenBatchUnit{
			{
				CorrelationID: "123",
				OriginalURL:   "www.ya.ru",
			},
			{
				CorrelationID: "123",
				OriginalURL:   "www.mail.ru",
			},
			{
				CorrelationID: "123",
				OriginalURL:   "www.rambler.ru",
			},
			{
				CorrelationID: "123",
				OriginalURL:   "www.rbc.ru",
			},
		}

	output := []ResponseShortenBatchUnit{
		{
			CorrelationID: "123",
			ShortURL:      "http://localhost:8080/1qa",
		},
		{
			CorrelationID: "123",
			ShortURL:      "http://localhost:8080/2qa",
		},
		{
			CorrelationID: "123",
			ShortURL:      "http://localhost:8080/3qa",
		},
		{
			CorrelationID: "123",
			ShortURL:      "http://localhost:8080/4qa",
		},
	}

	lnkRecIn := []repository.LinkRecord{
		{
			CorrelationID: "123",
			URL:           "www.ya.ru",
		},
		{
			CorrelationID: "123",
			URL:           "www.mail.ru",
		},
		{
			CorrelationID: "123",
			URL:           "www.rambler.ru",
		},
		{
			CorrelationID: "123",
			URL:           "www.rbc.ru",
		},
	}

	lnkRecOut := []repository.LinkRecord{
		{
			CorrelationID: "123",
			URL:           "www.ya.ru",
			ShortURL:      "1qa",
		},
		{
			CorrelationID: "123",
			URL:           "www.mail.ru",
			ShortURL:      "2qa",
		},
		{
			CorrelationID: "123",
			URL:           "www.rambler.ru",
			ShortURL:      "3qa",
		},
		{
			CorrelationID: "123",
			URL:           "www.rbc.ru",
			ShortURL:      "4qa",
		},
	}
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mock_repository.NewMockIStorage(ctrl)
	storage.EXPECT().BatchCreate(gomock.Any(), lnkRecIn).MaxTimes(1).Return(lnkRecOut, nil)

	t.Run("Batch Shorten OK", func(t *testing.T) {
		b, err := json.Marshal(input)

		require.NoError(t, err, "WRONG PARAM. CAN'T MARSHAL IT")

		sendBody := bytes.NewReader(b)

		r := httptest.NewRequest(http.MethodPost, "/shorten/batch", sendBody)
		r = r.WithContext(ctx)

		server, err := NewServer(Logger, storage)

		require.NoError(t, err, "CAN'T CREATE SERVER")

		w := httptest.NewRecorder()
		server.actionBatch(w, r)

		res := w.Result()

		assert.Equal(t, http.StatusCreated, res.StatusCode)

		resBody, err := io.ReadAll(res.Body)
		require.NoError(t, err, "CAN'T READ BODY")
		defer res.Body.Close()

		response := []ResponseShortenBatchUnit{}
		err = json.Unmarshal(resBody, &response)
		require.NoError(t, err, "CAN'T UNMARSHAL ANSWER")

		assert.Equal(t, output, response)

	})

}

func TestActionAPIUrls(t *testing.T) {

	lnkRecOut := []repository.LinkRecord{
		{
			CorrelationID: "123",
			URL:           "www.ya.ru",
			ShortURL:      "1qa",
		},
		{
			CorrelationID: "123",
			URL:           "www.mail.ru",
			ShortURL:      "2qa",
		},
		{
			CorrelationID: "123",
			URL:           "www.rambler.ru",
			ShortURL:      "3qa",
		},
		{
			CorrelationID: "123",
			URL:           "www.rbc.ru",
			ShortURL:      "4qa",
		},
	}

	output := []repository.LinkRecord{
		{
			URL:      "www.ya.ru",
			ShortURL: "http://localhost:8080/1qa",
		},
		{
			URL:      "www.mail.ru",
			ShortURL: "http://localhost:8080/2qa",
		},
		{
			URL:      "www.rambler.ru",
			ShortURL: "http://localhost:8080/3qa",
		},
		{
			URL:      "www.rbc.ru",
			ShortURL: "http://localhost:8080/4qa",
		},
	}

	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mock_repository.NewMockIStorage(ctrl)

	storage.EXPECT().Urls(gomock.Any(), gomock.Any()).MaxTimes(1).Return(lnkRecOut, nil)

	s0, err := NewServer(Logger, storage)

	assert.NoError(t, err, "CAN'T CREATE SERVER")

	t.Run("API URLS", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/user/urls", nil)
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		s0.actionAPIUrls(w, r)

		res := w.Result()

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err, "CAN'T READ BODY")
		defer res.Body.Close()

		response := []repository.LinkRecord{}

		err = json.Unmarshal(body, &response)

		assert.NoError(t, err, "CAN'T UNMARSHAL BODY")

		assert.Equal(t, output, response)

	})

}
