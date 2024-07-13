package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/playmixer/gophermart/internal/adapters/api/rest"
	"github.com/playmixer/gophermart/internal/adapters/logger"
	"github.com/playmixer/gophermart/internal/adapters/store"
	"github.com/playmixer/gophermart/internal/core/config"
	"github.com/playmixer/gophermart/internal/core/gophermart"
)

func main() {
	if err := run(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Init()
	if err != nil {
		return fmt.Errorf("failed initilize config: %w", err)
	}

	lgr, err := logger.New(cfg.LogLevel, logger.OutputPath(cfg.LogPath))
	if err != nil {
		return fmt.Errorf("failed initialize logger: %w", err)
	}

	storage, err := store.New(ctx, cfg.Store, lgr)
	if err != nil {
		return fmt.Errorf("failed initilize storage: %w", err)
	}

	mart := gophermart.New(ctx, cfg.Gophermart, storage, gophermart.SetSecretKey(cfg.Secret), gophermart.Logger(lgr))

	server, err := rest.New(
		mart,
		rest.Logger(lgr),
		rest.Configure(cfg.Rest),
	)
	if err != nil {
		return fmt.Errorf("failed initialize rest server: %w", err)
	}

	err = server.Run()
	if err != nil {
		return fmt.Errorf("stop server, %w", err)
	}
	return nil
}
