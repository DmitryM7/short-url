package conf

import (
	"flag"
	"os"
)

var (
	// BndAdd - адрес на котором будет размещаться сервер
	BndAdd string
	// RetAdd - адрес который будет присоединяться к результируюему ответу
	RetAdd string
	// FilePath - путь к файлу в котором будет храниться база с ссылками
	FilePath string
	// DSN - строка подключения к СУБД
	DSN string
)

// ParseFlags - парсит опции параметров запуска
func ParseFlags() {
	flag.StringVar(&BndAdd, "a", "localhost:8080", "host where server is run")
	flag.StringVar(&RetAdd, "b", "http://localhost:8080", "host that add to short link")
	flag.StringVar(&FilePath, "f", "./repo.json", "the path to the file where the matching table of short and full links will be stored")
	flag.StringVar(&DSN, "d", "", "database dsn")
}

// ParseEnv - парсит опции из переменных окружения
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

	if env := os.Getenv("DATABASE_DSN"); env != "" {
		DSN = env
	}
}
