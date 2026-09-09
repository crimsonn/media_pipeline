package config

import (
	"fmt"
	"time"
)

type Config struct {
	APIAddr           string
	APIPort           string
	Environment       string
	TranscoderWorkers int
	DatabaseURL       string
	WatchDirectory    string
	OutputDirectory   string
	PollingInterval   time.Duration
}

func LoadConfig() *Config {
	apiAddr := GetEnvString("API_ADDR", "localhost")
	apiPort := GetEnvString("API_PORT", "8080")
	environment := GetEnvString("ENVIRONMENT", "development")
	databaseURL := GetEnvString(
		"DATABASE_URL",
		"postgres://media:media@localhost:5432/media_pipeline?sslmode=disable",
	)
	transcoderWorkers := GetEnvInt("TRANSCODER_WORKERS", 10)
	watchDirectory := GetEnvString("WATCH_DIRECTORY", "./watch")
	outputDirectory := GetEnvString("OUTPUT_DIRECTORY", "./done")
	pollingInterval := GetEnvString("POLLING_INTERVAL", "5s")
	pollingIntervalDuration, err := time.ParseDuration(pollingInterval)
	if err != nil {
		panic(fmt.Errorf("parse polling interval: %w", err))
	}
	return &Config{
		APIAddr:           apiAddr,
		APIPort:           apiPort,
		Environment:       environment,
		TranscoderWorkers: transcoderWorkers,
		DatabaseURL:       databaseURL,
		WatchDirectory:    watchDirectory,
		OutputDirectory:   outputDirectory,
		PollingInterval:   pollingIntervalDuration,
	}
}
