package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"flag"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	parseFlags()
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestActionCreateURL(t *testing.T) {

	linkTable = make(map[string]string, 100)

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
				reqBody := strings.NewReader(tt.args.body)
				r = httptest.NewRequest(tt.args.method, tt.args.url, reqBody)
			} else {
				r = httptest.NewRequest(tt.args.method, tt.args.url, nil)
			}

			w := httptest.NewRecorder()
			actionCreateURL(w, r)
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

	linkTable = make(map[string]string, 100)

	linkTable["b8da4f2d"] = "www.ya.ru"

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
			actionRedirect(w, r)
			res := w.Result()
			body, _ := io.ReadAll(res.Body)
			t.Log(string(body))
			defer res.Body.Close()
		})
	}
}
