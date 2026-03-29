package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	BotToken       string
	DatabaseDSN    string
	RedisAddr      string
	WorkerCount    int
	ScrapeInterval int // in seconds
}

func Load() (*Config, error) {
	cfg := &Config{
		BotToken:       os.Getenv("BOT_TOKEN"),
		RedisAddr:      getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		WorkerCount:    getEnvOrDefaultInt("WORKER_COUNT", 5),
		ScrapeInterval: getEnvOrDefaultInt("SCRAPE_INTERVAL", 60),
	}

	cfg.DatabaseDSN = fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		getEnvOrDefault("DB_HOST", "localhost"),
		getEnvOrDefault("DB_USER", "postgres"),
		getEnvOrDefault("DB_PASSWORD", "password"),
		getEnvOrDefault("DB_NAME", "olx_hunter"),
		getEnvOrDefault("DB_PORT", "5432"),
		getEnvOrDefault("DB_SSLMODE", "disable"),
	)

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvOrDefaultInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}
