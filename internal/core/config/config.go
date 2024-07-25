package config

import (
	"errors"
	"flag"
	"fmt"
	"os"

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
	LogPath    string `env:"LOG_PATH"`
}

func Init() (*Config, error) {
	cfg := &Config{
		Rest: &rest.Config{},
		Store: &store.Config{
			Database: &database.Config{},
		},
		Gophermart: &gophermart.Config{},
	}

	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return cfg, fmt.Errorf("failed load enviorements from file: %w", err)
	}

	if err := env.Parse(cfg); err != nil {
		return cfg, fmt.Errorf("failed parse env: %w", err)
	}

	regStringFlag(&cfg.Rest.Address, "a", cfg.Rest.Address, "address listen")
	regStringFlag(&cfg.Store.Database.DSN, "d", cfg.Store.Database.DSN, "database dsn")
	regStringFlag(&cfg.Gophermart.AccrualAddress, "r", cfg.Gophermart.AccrualAddress, "address accrual system")
	flag.Parse()

	return cfg, nil
}

func regStringFlag(p *string, name string, value string, usage string) {
	if flag.Lookup(name) == nil {
		flag.StringVar(p, name, value, usage)
	}
}
