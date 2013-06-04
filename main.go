// Copyright 2013 Tamás Gulácsi. All rights reserved.
// Use of this source code is governed by an Apache 2.0
// license that can be found in the LICENSE file.

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
