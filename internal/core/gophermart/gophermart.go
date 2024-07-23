package gophermart

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/playmixer/gophermart/internal/adapters/store/model"
	"go.uber.org/zap"
)

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

var (
	workerCount     = 5
	delayUpdAccrual = time.Second * 10
)

type Config struct {
	AccrualAddress  string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"localhost:8081"`
	GorutineEnabled bool   `env:"GOROUTINE_ENABLED" envDefault:"true"`
}

type Gophermart struct {
	log    *zap.Logger
	cfg    *Config
	store  Store
	secret string
	wg     *sync.WaitGroup
}

type option func(*Gophermart)

func SetSecretKey(secret string) option {
	return func(g *Gophermart) {
		g.secret = secret
	}
}

func Logger(log *zap.Logger) option {
	return func(g *Gophermart) {
		g.log = log
	}
}

func New(ctx context.Context, cfg *Config, store Store, options ...option) *Gophermart {
	g := &Gophermart{
		store: store,
		cfg:   cfg,
		wg:    &sync.WaitGroup{},
	}

	for _, opt := range options {
		opt(g)
	}

	if g.cfg.GorutineEnabled {
		semaphore := NewSemaphore()
		g.wg.Add(1)
		outputCh := g.generatorUpdAccrual(ctx)
		for i := range workerCount {
			g.wg.Add(1)
			go g.workerUpdOrders(ctx, i, outputCh, semaphore)
		}
	}

	return g
}

func (g *Gophermart) Register(ctx context.Context, login, password string) error {
	if err := validatePassword(password); err != nil {
		return fmt.Errorf("password invalidate: %w", err)
	}

	if err := validateLogin(login); err != nil {
		return fmt.Errorf("login invalidate: %w", err)
	}

	hashPass, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed hash password: %w", err)
	}

	err = g.store.RegisterUser(ctx, login, hashPass)
	if err != nil {
		return fmt.Errorf("failed register user: %w", err)
	}

	return nil
}

func (g *Gophermart) Authorization(ctx context.Context, login, password string) (model.User, error) {
	var user model.User
	var err error
	if err := validatePassword(password); err != nil {
		return user, fmt.Errorf("password invalidate: %w", err)
	}

	if err := validateLogin(login); err != nil {
		return user, fmt.Errorf("login invalidate: %w", err)
	}

	user, err = g.store.GetUserByLogin(ctx, login)
	if err != nil {
		return user, fmt.Errorf("failed getting user `%s`: %w", login, err)
	}

	if ok := checkPasswordHash(password, user.PasswordHash); !ok {
		return user, ErrPasswordNotEquale
	}

	return user, nil
}

func (g *Gophermart) UploadOrder(ctx context.Context, userID uint, orderNumber string) error {
	if ok := checkLuhn(orderNumber); !ok {
		return ErrOrderNumberNotValid
	}

	err := g.store.UploadOrder(ctx, userID, orderNumber)
	if err != nil {
		return fmt.Errorf("failed upload order: %w", err)
	}

	return nil
}

func (g *Gophermart) GetUserOrders(ctx context.Context, userID uint) ([]*model.Order, error) {
	orders, err := g.store.GetUserOrders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed getting order by user: %w", err)
	}
	return orders, nil
}

func (g *Gophermart) GetUserBalance(ctx context.Context, userID uint) (model.Balance, error) {
	balance, err := g.store.GetUserBalance(ctx, userID)
	if err != nil {
		return balance, fmt.Errorf("failed getting balance by user: %w", err)
	}

	return balance, nil
}

func (g *Gophermart) WithdrawFromBalanceUser(ctx context.Context, userID uint, order string, sum float32) error {
	if ok := checkLuhn(order); !ok {
		return ErrOrderNumberNotValid
	}

	err := g.store.WithdrawFromUserBalance(ctx, userID, order, sum)
	if err != nil {
		return fmt.Errorf("failed with draw from user balance: %w", err)
	}

	return nil
}

func (g *Gophermart) GetWithdrawalsByUser(ctx context.Context, userID uint) ([]*model.WithdrawBalance, error) {
	balance, err := g.store.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed get user balance: %w", err)
	}
	withdrawals, err := g.store.GetWithdrawalsFromBalance(ctx, balance.ID)
	if err != nil {
		return withdrawals, fmt.Errorf("failed get withdrawals: %w", err)
	}

	return withdrawals, nil
}

func checkLuhn(ccn string) bool {
	sum := 0
	half := 2
	maxDigit := 9
	parity := len(ccn) % half

	for i := range len(ccn) {
		digit, _ := strconv.Atoi(string(ccn[i]))
		if i%2 == parity {
			digit *= 2
			if digit > maxDigit {
				digit -= maxDigit
			}
		}
		sum += digit
	}

	return sum%10 == 0
}

func (g *Gophermart) generatorUpdAccrual(ctx context.Context) <-chan *model.Order {
	outpuCh := make(chan *model.Order)
	go func() {
		g.log.Debug("start gorutin generatorUpdAccrual")
		defer g.log.Debug("stopped gorutin generatorUpdAccrual")
		defer g.wg.Done()
		tick := time.NewTicker(delayUpdAccrual)
		defer close(outpuCh)
		for {
			select {
			case <-ctx.Done():
				g.log.Debug("generator update accrual stopping")
				return
			case <-tick.C:
				orders, err := g.store.GetOrdersNotPrecessed(ctx)
				if err != nil {
					g.log.Error("failed getting orders without processed status", zap.Error(err))
					continue
				}
				for _, order := range orders {
					outpuCh <- order
				}
			}
		}
	}()
	return outpuCh
}

func (g *Gophermart) workerUpdOrders(ctx context.Context, id int, inputCh <-chan *model.Order, semaphore *semaphore) {
	g.log.Debug("start gorutin workerUpdOrders", zap.Int("id", id))
	defer g.log.Debug("stopped gorutin workerUpdOrders", zap.Int("id", id))
	defer g.wg.Done()
	for {
		select {
		case <-ctx.Done():
			g.log.Info("worker updating order stopping", zap.Int("id", id))
			return
		case o := <-inputCh:
			semaphore.Wait()
			func() {
				resp, err := http.Get(g.cfg.AccrualAddress + "/api/orders/" + o.Number)
				if err != nil {
					g.log.Error("request failed from accrual service", zap.String("orderNumber", o.Number))
					return
				}
				defer func() { _ = resp.Body.Close() }()
				bBody, err := io.ReadAll(resp.Body)
				if err != nil {
					g.log.Error("failed to read response body", zap.String("body", string(bBody)), zap.String("orderNumber", o.Number))
					return
				}
				if resp.StatusCode == http.StatusOK {
					jBody := tOrderAccrualBody{}
					err = json.Unmarshal(bBody, &jBody)
					if err != nil {
						g.log.Error("failed unmarshal accrual response body", zap.Error(err))
						return
					}
					order := o
					order.Accrual = jBody.Accrual
					order.Status = model.OrderStatus(jBody.Status)
					err = g.store.AddAccrual(ctx, order)
					if err != nil {
						g.log.Error("failed add accrual", zap.Error(err))
						return
					}
					return
				}
				if resp.StatusCode == http.StatusNoContent {
					g.log.Debug("no content", zap.String("order", o.Number))
					return
				}
				if resp.StatusCode == http.StatusTooManyRequests {
					sRetryAfter := resp.Header.Get("Retry-After")
					g.log.Debug("too many requests",
						zap.String("status", resp.Status),
						zap.String("Retry-After", sRetryAfter),
					)
					iRetryAfter, _ := strconv.Atoi(sRetryAfter)
					semaphore.Lock(time.Second * time.Duration(iRetryAfter))
					return
				}
				g.log.Info("not correct response",
					zap.String("status", resp.Status),
					zap.String("order", o.Number),
					zap.String("body", string(bBody)),
				)
			}()
		}
	}
}

func (g *Gophermart) Wait() {
	g.wg.Wait()
}
