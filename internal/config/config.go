package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v6"
)

// Config содержит все настройки приложения
type Config struct {
	// Telegram
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN,required"`

	// Database
	DatabaseURL string `env:"DATABASE_URL,required"`

	// SStats API
	SStatsAPIKey  string `env:"SSTATS_API_KEY"`
	SStatsAPIBase string `env:"SSTATS_API_BASE" envDefault:"https://api.sstats.net"`

	// Logging
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

// Load загружает конфигурацию из переменных окружения
// и выполняет базовую валидацию
func Load() (*Config, error) {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	// Дополнительная валидация
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate проверяет корректность конфигурации
func (c *Config) validate() error {
	if c.TelegramBotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	// Проверка, что URL базы данных содержит postgresql://
	if !strings.HasPrefix(c.DatabaseURL, "postgresql://") {
		return fmt.Errorf("DATABASE_URL must start with 'postgresql://', got: %s", c.DatabaseURL)
	}

	return nil
}
