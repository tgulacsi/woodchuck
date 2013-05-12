package main

import (
    "github.com/robertkowalski/graylog-golang"
    "log"
)

func main() {
	g := gelf.New(gelf.Config{})
	g.Log("start")
	res := g.ParseJson("aa")
    log.Printf("res=%v", res)
}
