package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/playmixer/gophermart/internal/adapters/store/errstore"
	"github.com/playmixer/gophermart/internal/adapters/store/model"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Store struct {
	db  *gorm.DB
	log *zap.Logger
}

type option func(*Store)

func Logger(log *zap.Logger) option {
	return func(s *Store) {
		if log != nil {
			s.log = log
		}
	}
}

func New(ctx context.Context, cfg *Config, options ...option) (*Store, error) {
	var err error
	s := &Store{
		log: zap.NewNop(),
	}
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed connect to database: %w", err)
	}

	s.db = db.WithContext(ctx)

	for _, opt := range options {
		opt(s)
	}

	err = s.db.AutoMigrate(
		&model.User{},
		&model.Order{},
		&model.Balance{},
		&model.WithdrawBalance{},
	)

	if err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Store) CloseDB() error {
	db, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed getting database connection: %w", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed close database connection: %w", err)
	}

	return nil
}

func (s *Store) RegisterUser(ctx context.Context, login, hashPassword string) error {
	user := model.User{
		Login:        login,
		PasswordHash: hashPassword,
	}
	result := s.db.Create(&user)
	if err := result.Error; err != nil {
		var sqlError *pgconn.PgError
		if errors.As(err, &sqlError) && sqlError.Code == pgerrcode.UniqueViolation {
			return errstore.ErrLoginNotUnique
		}
		return fmt.Errorf("failed save user: %w", result.Error)
	}

	return nil
}
func (s *Store) GetUserByLogin(ctx context.Context, login string) (model.User, error) {
	tx := s.db.WithContext(ctx)
	user := model.User{}
	result := tx.Where(&model.User{Login: login}).First(&user)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, errors.Join(errstore.ErrNotFoundData, err)
		}
		return user, fmt.Errorf("error found user: %w", result.Error)
	}

	return user, nil
}

func (s *Store) UploadOrder(ctx context.Context, userID uint, orderNumber string) error {
	tx := s.db.WithContext(ctx)
	err := tx.Transaction(func(tx *gorm.DB) error {
		order := model.Order{}
		if err := tx.Where(&model.Order{Number: orderNumber}).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				order.UserID = userID
				order.Number = orderNumber
				if err := tx.Create(&order).Error; err != nil {
					return fmt.Errorf("failed create order: %w", err)
				}
				return nil
			}
			return fmt.Errorf("failed select order: %w", err)
		}
		if order.UserID != userID {
			return errstore.ErrOrderWasCreatedAnotherUser
		}
		return errstore.ErrOrderWasCreatedByUser
	})
	if err != nil {
		return fmt.Errorf("failed complite transaction: %w", err)
	}

	return nil
}

func (s *Store) GetUserBalance(ctx context.Context, userID uint) (model.Balance, error) {
	balance := model.Balance{}
	if err := s.db.Where(&model.Balance{UserID: userID}).First(&balance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return balance, errstore.ErrNotFoundData
		}
		return balance, fmt.Errorf("failed get balance: %w", err)
	}

	return balance, nil
}

func (s *Store) WithdrawFromUserBalance(ctx context.Context, userID uint, order string, sum float32) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		balance := model.Balance{}
		err := tx.Where(&model.Balance{UserID: userID}).First(&balance).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed get balance: %w", err)
		}

		if balance.Current < sum {
			return fmt.Errorf("%w: %f", errstore.ErrBalansNotEnough, sum)
		}

		balance.Current -= sum
		balance.Withdrawn += sum
		if err := tx.Save(&balance).Error; err != nil {
			return fmt.Errorf("failed save balance: %w", err)
		}

		withdraw := model.WithdrawBalance{
			OderNumber: order,
			Sum:        sum,
			BalanceID:  balance.ID,
		}
		if err := tx.Save(&withdraw).Error; err != nil {
			return fmt.Errorf("failed save withdraw: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed complite transaction: %w", err)
	}

	return nil
}

func (s *Store) GetUserOrders(ctx context.Context, userID uint) ([]*model.Order, error) {
	orders := []*model.Order{}
	if err := s.db.Where(&model.Order{UserID: userID}).Find(&orders).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return orders, errstore.ErrNotFoundData
		}
		return nil, fmt.Errorf("failed get orders: %w", err)
	}
	if len(orders) == 0 {
		return orders, errstore.ErrNotFoundData
	}

	return orders, nil
}

func (s *Store) GetWithdrawalsFromBalance(ctx context.Context, balanceID uint) ([]*model.WithdrawBalance, error) {
	withdrawals := []*model.WithdrawBalance{}
	if err := s.db.Where(&model.WithdrawBalance{BalanceID: balanceID}).Find(&withdrawals).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return withdrawals, errstore.ErrNotFoundData
		}
		return withdrawals, fmt.Errorf("failed get withdrawals: %w", err)
	}
	if len(withdrawals) == 0 {
		return withdrawals, errstore.ErrNotFoundData
	}

	return withdrawals, nil
}

func (s *Store) GetOrdersNotPrecessed(ctx context.Context) ([]*model.Order, error) {
	orders := []*model.Order{}
	db := s.db.WithContext(ctx)
	if err := db.Where("status = ?", model.OrderStateNew).Or("status = ?", model.OrderStateProcessing).
		Find(&orders).Error; err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return orders, fmt.Errorf("failed get orders: %w", err)
	}

	return orders, nil
}

func (s *Store) AddAccrual(ctx context.Context, order *model.Order) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(order).Error; err != nil {
			return fmt.Errorf("failed update order id=`%d`: %w", order.ID, err)
		}
		balance := model.Balance{UserID: order.UserID}
		if err := tx.Where(&balance).First(&balance).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed getting balance by user `%d`: %w", order.ID, err)
		}
		balance.Current += order.Accrual
		if err := tx.Save(&balance).Error; err != nil {
			return fmt.Errorf("failed update balance by user `%d`: %w", order.UserID, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed complite transaction: %w", err)
	}

	return nil
}
