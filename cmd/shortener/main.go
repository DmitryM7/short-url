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

	"github.com/go-chi/chi/v5"
)

var repo linkRepo

func createShortURL(url string) (string, error) {
	var shortURL string

	checksum := crc32.Checksum([]byte(url), crc32.MakeTable(crc32.IEEE))

	shortURL = fmt.Sprintf("%08x", checksum)

	return shortURL, nil
}

func getURL(id string) (string, error) {

	if url, err := repo.Get(id); err == nil {
		return url, nil
	}

	return "", errors.New("NO REQUIRED PARAM ID")

}

func actionError(w http.ResponseWriter, e string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(e))
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

	newURL, err := createShortURL(url)

	if err != nil {
		actionError(w, "Can't create short url.")
	}

	_ = repo.Create(newURL, url)

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(retAdd + "/" + newURL))

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

func main() {

	parseFlags()
	flag.Parse()
	parseEnv()

	repo = NewLinkRepo()

	r := chi.NewRouter()

	slog.Info("Bind address:" + bndAdd)
	slog.Info("Return addres:" + retAdd)

	r.Route("/", func(r chi.Router) {
		r.Post("/", actionCreateURL)
		r.Get("/{id}", actionRedirect)
	})

	err := http.ListenAndServe(bndAdd, r)

	if err != nil {
		panic(err)
	}

}
