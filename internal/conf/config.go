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
	flag.StringVar(&BndAdd, "a", "localhost:8080", "адрес на котором запускается сервис")
	flag.StringVar(&RetAdd, "b", "http://localhost:8080", "адрес который возвращается после создания короткого алиаса")
	flag.StringVar(&FilePath, "f", "./repo.json", "путь к файлу в котором будет хранится таблица соответствия коротких и полных ссылок")
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
