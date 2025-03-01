package main

import (
	"github.com/clearthree/url-shortener/internal/app/server"
)

func main() {
	if err := server.Run(); err != nil {
		panic(err)
	}
}
