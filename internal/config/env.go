package config

import (
	"log/slog"
	"os"
	"strconv"
)

func GetEnvString(key string, fallback string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	return v
}

func GetEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	vNum, err := strconv.Atoi(v)
	if err != nil {
		slog.Warn("%s is not an integer", key)
		return fallback
	}

	return vNum
}
