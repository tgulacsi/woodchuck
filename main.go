package main

import (
	"github.com/tgulacsi/woodchuck/loglib"
	"log"
)

func main() {
	s, err := loglib.LoadConfig("config.toml", "filters.toml")
	if err != nil {
		log.Fatalf("error loading config: %s", err)
	}
	s.Serve()
}
