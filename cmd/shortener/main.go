package main

import (
	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/server"
)

func main() {
	config.ParseFlags()
	if err := server.Run(config.Config.Address.String()); err != nil {
		panic(err)
	}
}
