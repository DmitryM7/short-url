package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"flag"

	"github.com/DmitryM7/short-url.git/internal/conf"
	"github.com/DmitryM7/short-url.git/internal/logger"
	"github.com/DmitryM7/short-url.git/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var Logger logger.MyLogger
var Repo repository.StorageService

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
	_, err := Repo.Create("www.ya.ru")

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
