package logger

import (
	"github.com/clearthree/url-shortener/internal/app/config"
	"go.uber.org/zap"
	"log"
)

var Log *zap.SugaredLogger

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	log, err := cfg.Build()
	if err != nil {
		return err
	}
	Log = log.Sugar()
	return nil
}

func init() {
	err := Initialize(config.Settings.LogLevel)
	if err != nil {
		log.Fatal("error initializing logger")
	}
}
