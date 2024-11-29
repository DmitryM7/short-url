package main

import (
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

var linkTable map[string]string

func createShortURL(url string) (string, error) {
	var shortURL string

	checksum := crc32.Checksum([]byte(url), crc32.MakeTable(crc32.IEEE))

	shortURL = fmt.Sprintf("%08x", checksum)

	return shortURL, nil
}

func getURL(id string) (string, error) {

	if url, ok := linkTable[id]; ok {
		return url, nil
	}

	return "", errors.New("НЕТ ССЫЛКИ ДЛЯ ID")

}

func actionError(w http.ResponseWriter, e string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(e))
}

func actionCreateURL(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)

	if err != nil {
		actionError(w, "Ошибка чтения запроса.")
		return
	}

	url := string(body)

	if url == "" {
		actionError(w, "Обязательный пар-р пустой.")
		return
	}

	newURL, err := createShortURL(url)

	if err != nil {
		actionError(w, "Не удалось создать короткую ссылку")
	}

	linkTable[newURL] = url

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(retAdd + "/" + newURL))

}

func actionRedirect(w http.ResponseWriter, r *http.Request) {

	id := strings.TrimPrefix(r.URL.Path, "/")

	if id == "" {
		actionError(w, "Отсутствует id короткой ссылки")
		return
	}

	newURL, err := getURL(id)

	if err != nil {
		actionError(w, "Не нашел ссылку по указанному id")
		return
	}

	http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
}

func main() {

	parseFlags()
	flag.Parse()
	parseEnv()

	linkTable = make(map[string]string, 100)

	r := chi.NewRouter()

	log.Println("Bind address:" + bndAdd)
	log.Println("Return addres:" + retAdd)

	r.Route("/", func(r chi.Router) {
		r.Post("/", actionCreateURL)
		r.Get("/{id}", actionRedirect)
	})

	err := http.ListenAndServe(bndAdd, r)

	if err != nil {
		panic(err)
	}

}
