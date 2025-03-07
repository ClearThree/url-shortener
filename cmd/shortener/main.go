package main

import (
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/server"
)

func main() {
	config.ParseFlags()
	err := env.Parse(&config.Settings)
	config.Settings.Sanitize()
	if err != nil {
		fmt.Println("parsing env variables was not successful: ", err)
	}
	if err := server.Run(config.Settings.Address); err != nil {
		panic(err)
	}
}
