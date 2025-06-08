package logger

import (
	"log"

	"go.uber.org/zap"

	"github.com/clearthree/url-shortener/internal/app/config"
)

var Log *zap.SugaredLogger

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	Log = logger.Sugar()
	return nil
}

func init() {
	err := Initialize(config.Settings.LogLevel)
	if err != nil {
		log.Fatal("error initializing logger")
	}
}
