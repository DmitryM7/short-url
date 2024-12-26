package main

import (
	"flag"
	"os"
)

var (
	bndAdd   string
	retAdd   string
	filePath string
)

func parseFlags() {
	flag.StringVar(&bndAdd, "a", "localhost:8080", "адрес на котором запускается сервис")
	flag.StringVar(&retAdd, "b", "http://localhost:8080", "адрес который возвращается после создания короткого алиаса")
	flag.StringVar(&filePath, "f", "./repo.json", "путь к файлу в котором будет хранится таблица соответствия коротких и полных ссылок")
}

func parseEnv() {
	if env := os.Getenv("SERVER_ADDRESS"); env != "" {
		bndAdd = env
	}

	if env := os.Getenv("BASE_URL"); env != "" {
		retAdd = env
	}

	if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
		filePath = env
	}
}
