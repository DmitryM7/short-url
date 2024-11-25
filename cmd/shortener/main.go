package main

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strings"
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
	w.Write([]byte(newURL))

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

func mainEndPoint(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodPost:
		actionCreateURL(w, r)

	case http.MethodGet:
		actionRedirect(w, r)

	default:
		actionError(w, "Отсутствует необходимый end-point.")
	}
}

func main() {

	linkTable = make(map[string]string, 100)

	mux := http.NewServeMux()
	mux.HandleFunc("/", mainEndPoint)

	err := http.ListenAndServe(`:8080`, mux)

	if err != nil {
		panic(err)
	}

}
