package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port     int
	DataFile string
}

func Load() (Config, error) {
	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		return Config{}, fmt.Errorf("PORT must be a valid integer: %w", err)
	}
	if port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
	}

	dataFile := getEnv("DATA_FILE", "data/subscriptions.json")
	absDataFile, err := filepath.Abs(dataFile)
	if err != nil {
		return Config{}, fmt.Errorf("resolve DATA_FILE: %w", err)
	}

	return Config{
		Port:     port,
		DataFile: absDataFile,
	}, nil
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
