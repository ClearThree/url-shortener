package main

import (
	"fmt"
	"log"

	"github.com/caarlos0/env/v6"

	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/server"
)

var (
	buildVersion string = "N/A"
	buildCommit  string = "N/A"
	buildDate    string = "N/A"
)

func main() {
	log.Printf("Build version: %s", buildVersion)
	log.Printf("Build date: %s", buildDate)
	log.Printf("Build commit: %s", buildCommit)
	config.ParseFlags()

	err := env.Parse(&config.Settings)
	if err != nil {
		fmt.Println("parsing env variables was not successful: ", err)
	}
	config.Settings.Sanitize()
	if err = server.Run(config.Settings.Address); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
