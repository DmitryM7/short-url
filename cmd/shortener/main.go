package main

import (
	"io"
	"net/http"
	"strings"
)

func createShortUrl(url string) string {
	return url
}

func getUrl(id string) (string, error) {
	return id, nil
}

func actionCreateUrl(w http.ResponseWriter, r *http.Request) {

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

	newUrl := createShortUrl(url)

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(newUrl))

}

func actionRedirect(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		actionError(w, "Не смог получить обязательные параметры запроса")
	}

	id := strings.TrimPrefix(r.URL.Path, "/")

	newUrl, err := getUrl(id)

	if err != nil {
		actionError(w, "Не нашел ссылку по указанному id")
		return
	}

	http.Redirect(w, r, newUrl, http.StatusTemporaryRedirect)
}

func actionError(w http.ResponseWriter, e string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(e))
}

func mainEndPoint(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodPost:
		actionCreateUrl(w, r)
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
