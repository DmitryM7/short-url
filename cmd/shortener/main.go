package main

import (
	"io"
	"net/http"
	"strings"
)

func createShortURL(url string) string {
	return url
}

func getURL(id string) (string, error) {
	return id, nil
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

	newURL := createShortURL(url)

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(newURL))

}

func actionRedirect(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		actionError(w, "Не смог получить обязательные параметры запроса")
	}

	id := strings.TrimPrefix(r.URL.Path, "/")

	newURL, err := getURL(id)

	if err != nil {
		actionError(w, "Не нашел ссылку по указанному id")
		return
	}

	http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
}

func actionError(w http.ResponseWriter, e string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(e))
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

	mux := http.NewServeMux()
	mux.HandleFunc("/", mainEndPoint)

	err := http.ListenAndServe(`:8080`, mux)

	if err != nil {
		panic(err)
	}

}
