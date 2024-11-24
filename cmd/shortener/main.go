package main

import "net/http"

func createShortUrl(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)

}

func redirectByAlias(w http.ResponseWriter, r *http.Request) {

	http.Redirect(w, r, "http://www.ya.ru", http.StatusTemporaryRedirect)

}

func mainEndPoint(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodPost:
		createShortUrl(w, r)

	case http.MethodGet:
		redirectByAlias(w, r)

	default:
		w.WriteHeader(http.StatusBadRequest)

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
