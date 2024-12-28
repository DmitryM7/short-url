package conf

import (
	"flag"
	"os"
)

var (
	BndAdd   string
	RetAdd   string
	FilePath string
)

func ParseFlags() {
	flag.StringVar(&BndAdd, "a", "localhost:8080", "host where server is run")
	flag.StringVar(&RetAdd, "b", "http://localhost:8080", "host that add to short link")
	flag.StringVar(&FilePath, "f", "./repo.json", "the path to the file where the matching table of short and full links will be stored")
}

func ParseEnv() {
	if env := os.Getenv("SERVER_ADDRESS"); env != "" {
		BndAdd = env
	}

	if env := os.Getenv("BASE_URL"); env != "" {
		RetAdd = env
	}

	if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
		FilePath = env
	}
}
