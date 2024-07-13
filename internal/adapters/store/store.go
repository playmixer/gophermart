package store

import (
	"context"
	"fmt"

	"github.com/playmixer/gophermart/internal/adapters/store/database"
	"github.com/playmixer/gophermart/internal/adapters/store/model"
	"go.uber.org/zap"
)

type Config struct {
	Database *database.Config
}

type Store interface {
	RegisterUser(ctx context.Context, login, hashPassword string) error
	GetUserByLogin(ctx context.Context, login string) (model.User, error)
	UploadOrder(ctx context.Context, userID uint, orderNumber string) error
	GetUserOrders(ctx context.Context, userID uint) ([]*model.Order, error)
	GetUserBalance(ctx context.Context, userID uint) (model.Balance, error)
	WithdrawFromUserBalance(ctx context.Context, userID uint, order string, sum float32) error
	GetWithdrawalsFromBalance(ctx context.Context, balanceID uint) ([]*model.WithdrawBalance, error)
	GetOrdersNotPrecessed(ctx context.Context) ([]*model.Order, error)
	AddAccrual(ctx context.Context, order *model.Order) error
}

func New(ctx context.Context, cfg *Config, log *zap.Logger) (Store, error) {
	s, err := database.New(ctx, cfg.Database, database.Logger(log))
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return s, nil
}
