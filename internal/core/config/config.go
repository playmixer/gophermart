package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/playmixer/gophermart/internal/adapters/api/rest"
	"github.com/playmixer/gophermart/internal/adapters/store"
	"github.com/playmixer/gophermart/internal/adapters/store/database"
	"github.com/playmixer/gophermart/internal/core/gophermart"
)

type Config struct {
	Rest       *rest.Config
	Store      *store.Config
	Gophermart *gophermart.Config
	Secret     string `env:"SECRET_KEY" envDefault:"secret_key"`
	LogLevel   string `env:"LOG_LEVEL" envDefault:"info"`
}

func Init() (Config, error) {
	cfg := Config{
		Rest: &rest.Config{},
		Store: &store.Config{
			Database: &database.Config{},
		},
		Gophermart: &gophermart.Config{
			GorutineEnabled: true,
		},
	}
	_ = godotenv.Load(".env")
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("failed parse env: %w", err)
	}

	return cfg, nil
}
