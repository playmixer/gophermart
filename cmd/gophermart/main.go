package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/playmixer/gophermart/internal/adapters/api/rest"
	"github.com/playmixer/gophermart/internal/adapters/logger"
	"github.com/playmixer/gophermart/internal/adapters/store"
	"github.com/playmixer/gophermart/internal/core/config"
	"github.com/playmixer/gophermart/internal/core/gophermart"
	"go.uber.org/zap"
)

var (
	shutdownDelay = time.Second * 2
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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
		rest.SetAddress(cfg.Rest.Address),
		rest.SetSecretKey([]byte(cfg.Rest.Secret)),
	)
	if err != nil {
		return fmt.Errorf("failed initialize rest server: %w", err)
	}

	lgr.Info("Starting")
	go func() {
		if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			lgr.Error("failed run server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	lgr.Info("Stopping...")
	ctxShutdown, stop := context.WithTimeout(context.Background(), shutdownDelay)
	defer stop()

	// выключаем http сервер
	if err := server.Shutdown(ctxShutdown); err != nil {
		lgr.Error("Server Shutdown with error", zap.Error(err))
	}

	// ждем завершения горутин
	mart.Wait()

	// закрываем соединение с БД
	if err := storage.CloseDB(); err != nil {
		lgr.Error("Close Database with error", zap.Error(err))
	}

	<-ctxShutdown.Done()
	lgr.Info("Service stopped")

	return nil
}
