package main

import (
	"flag"
	"os"
)

var (
	bndAdd string
	retAdd string
)

func parseFlags() {
	flag.StringVar(&bndAdd, "a", "localhost:8080", "адрес на котором запускается сервис")
	flag.StringVar(&retAdd, "b", "http://localhost:8080", "адрес который возвращается после создания короткого алиаса")
}

func parseEnv() {
	if env := os.Getenv("SERVER_ADDRESS"); env != "" {
		bndAdd = env
	}

	if env := os.Getenv("BASE_URL"); env != "" {
		retAdd = env
	}
}
