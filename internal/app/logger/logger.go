// Package logger initializes logger and contains global logger object.
package logger

import (
	"log"

	"go.uber.org/zap"

	"github.com/clearthree/url-shortener/internal/app/config"
)

// Log is the global logger object used for the logging.
var Log *zap.SugaredLogger

// Initialize is a function that sets up the logger according to the given level.
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
