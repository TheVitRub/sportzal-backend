package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment string
	ServiceName string
	Server      ServerConfig
	Storage     StorageConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type StorageConfig struct {
	DataPath  string
	UploadDir string
}

func Load() (*Config, error) {
	cfg := &Config{
		Environment: envOr("ENVIRONMENT", "development"),
		ServiceName: envOr("SERVICE_NAME", "workout-service"),
		Server: ServerConfig{
			Host:         envOr("HOST", ""),
			Port:         envInt("PORT", 8080),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Storage: StorageConfig{
			DataPath:  envOr("APP_DATA_PATH", "data/app.json"),
			UploadDir: envOr("UPLOAD_DIR", "uploads"),
		},
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
