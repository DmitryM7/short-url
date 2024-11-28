package main

import "flag"

var (
	bndAdd string
	retAdd string
)

func parseFlags() {
	flag.StringVar(&bndAdd, "a", "localhost:8080", "адрес на котором запускается сервис")
	flag.StringVar(&retAdd, "b", "localhost:8080", "адрес который возвращается после создания короткого алиаса")
	flag.Parse()
}
