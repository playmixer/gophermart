package rest_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/playmixer/gophermart/internal/adapters/api/rest"
	"github.com/playmixer/gophermart/internal/adapters/store/errstore"
	"github.com/playmixer/gophermart/internal/adapters/store/model"
	"github.com/playmixer/gophermart/internal/core/config"
	"github.com/playmixer/gophermart/internal/core/gophermart"
	"github.com/playmixer/gophermart/internal/mocks/store"
	"github.com/playmixer/gophermart/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	cookieKey = "UserID"
)

func TestServer_handlerRegister(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		login    string
		password string
		status   int
	}{
		{
			name:     "correct",
			login:    "user",
			password: "pass",
			status:   http.StatusOK,
		},
		{
			name:     "empty",
			login:    "",
			password: "",
			status:   http.StatusBadRequest,
		},
		{
			name:     "not unique",
			login:    "user",
			password: "pass",
			status:   http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg, err := config.Init()
			assert.NoError(t, err)
			cfg.Gophermart.GorutineEnabled = false

			storeMock := store.NewMockStore(ctrl)

			if tt.status != http.StatusBadRequest {
				if tt.status == http.StatusConflict {
					storeMock.EXPECT().
						RegisterUser(ctx, gomock.Any(), gomock.Any()).
						Return(errstore.ErrLoginNotUnique).
						Times(1)
				} else {
					storeMock.EXPECT().
						RegisterUser(ctx, gomock.Any(), gomock.Any()).
						Return(nil).
						Times(1)
					hashPass, err := gophermart.HashPassword(tt.password)
					assert.NoError(t, err)
					storeMock.EXPECT().
						GetUserByLogin(ctx, tt.login).
						Return(model.User{
							PasswordHash: hashPass,
						}, nil).
						Times(1)
				}
			}
			mart := gophermart.New(ctx, cfg.Gophermart, storeMock)
			server, err := rest.New(mart)
			assert.NoError(t, err)
			engin := server.Engine()

			w := httptest.NewRecorder()
			body := strings.NewReader(fmt.Sprintf(`{"login":%q, "password":%q}`, tt.login, tt.password))
			r := httptest.NewRequest(http.MethodPost, "/api/user/register", body)

			engin.ServeHTTP(w, r)

			result := w.Result()

			assert.Equal(t, tt.status, result.StatusCode)

			err = result.Body.Close()
			assert.NoError(t, err)
		})
	}
}

func TestServer_handlerLogin(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		login    string
		password string
		status   int
	}{
		{
			name:     "correct",
			login:    "user",
			password: "pass",
			status:   http.StatusOK,
		},
		{
			name:     "empty",
			login:    "",
			password: "",
			status:   http.StatusBadRequest,
		},
		{
			name:     "unauthorize",
			login:    "user",
			password: "pass",
			status:   http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg, err := config.Init()
			assert.NoError(t, err)
			cfg.Gophermart.GorutineEnabled = false

			storeMock := store.NewMockStore(ctrl)
			hashPass, err := gophermart.HashPassword(tt.password)
			assert.NoError(t, err)
			if tt.status != http.StatusBadRequest {
				if tt.status == http.StatusUnauthorized {
					storeMock.EXPECT().
						GetUserByLogin(ctx, tt.login).
						Return(model.User{
							PasswordHash: "wrong pass",
						}, nil).
						Times(1)
				} else {
					storeMock.EXPECT().
						GetUserByLogin(ctx, tt.login).
						Return(model.User{
							PasswordHash: hashPass,
						}, nil).
						Times(1)
				}
			}

			assert.NoError(t, err)
			mart := gophermart.New(ctx, cfg.Gophermart, storeMock)
			server, err := rest.New(mart)
			assert.NoError(t, err)
			engin := server.Engine()

			w := httptest.NewRecorder()
			body := strings.NewReader(fmt.Sprintf(`{"login":%q, "password":%q}`, tt.login, tt.password))
			r := httptest.NewRequest(http.MethodPost, "/api/user/login", body)

			engin.ServeHTTP(w, r)

			result := w.Result()

			assert.Equal(t, tt.status, result.StatusCode)

			err = result.Body.Close()
			assert.NoError(t, err)
		})
	}
}

func TestServer_handlerLoadUserOrders(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		userID   uint
		order    string
		status   int
		errstore error
	}{
		{
			name:     "ok",
			userID:   1,
			order:    "12345678903",
			status:   http.StatusOK,
			errstore: errstore.ErrOrderWasCreatedByUser,
		},
		{
			name:     "apply",
			userID:   1,
			order:    "12345678903",
			status:   http.StatusAccepted,
			errstore: nil,
		},
		{
			name:     "bad format order number",
			userID:   1,
			order:    "",
			status:   http.StatusBadRequest,
			errstore: nil,
		},
		{
			name:   "unauthorize",
			userID: 1,
			order:  "12345678903",
			status: http.StatusUnauthorized,
		},
		{
			name:     "order was upload by another user",
			userID:   1,
			order:    "12345678903",
			status:   http.StatusConflict,
			errstore: errstore.ErrOrderWasCreatedAnotherUser,
		},
		{
			name:   "not correct order number",
			userID: 1,
			order:  "1234567890312",
			status: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg, err := config.Init()
			assert.NoError(t, err)
			cfg.Gophermart.GorutineEnabled = false

			storeMock := store.NewMockStore(ctrl)
			if !(tt.errstore == nil) || tt.name == "apply" {
				storeMock.EXPECT().
					UploadOrder(ctx, tt.userID, tt.order).
					Return(tt.errstore).
					Times(1)
			}

			mart := gophermart.New(ctx, cfg.Gophermart, storeMock)
			server, err := rest.New(mart, rest.SetSecretKey([]byte(cfg.Rest.Secret)))
			assert.NoError(t, err)
			engin := server.Engine()

			w := httptest.NewRecorder()
			body := strings.NewReader(tt.order)
			r := httptest.NewRequest(http.MethodPost, "/api/user/orders", body)
			jwtRest := jwt.New([]byte(cfg.Rest.Secret))
			if tt.status != http.StatusUnauthorized {
				signedCookie, err := jwtRest.Create(cookieKey, strconv.Itoa(int(tt.userID)))
				assert.NoError(t, err)
				userCookie := &http.Cookie{
					Name:  "token",
					Value: signedCookie,
					Path:  "/",
				}
				r.AddCookie(userCookie)
				http.SetCookie(w, userCookie)
			}

			engin.ServeHTTP(w, r)

			result := w.Result()

			assert.Equal(t, tt.status, result.StatusCode)

			err = result.Body.Close()
			assert.NoError(t, err)
		})
	}
}

func TestServer_handlerGetUserOrders(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		userID   uint
		orders   []*model.Order
		status   int
		errstore error
	}{
		{
			name:   "ok",
			userID: 1,
			orders: []*model.Order{
				{ID: 1, Number: "9278923470", UserID: 1},
			},
			status: http.StatusOK,
		},
		{
			name:     "no content",
			userID:   1,
			status:   http.StatusNoContent,
			errstore: errstore.ErrNotFoundData,
		},
		{
			name:   "unauthorize",
			userID: 1,
			status: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg, err := config.Init()
			assert.NoError(t, err)
			cfg.Gophermart.GorutineEnabled = false

			storeMock := store.NewMockStore(ctrl)
			if tt.errstore != nil || tt.name == "ok" {
				storeMock.EXPECT().
					GetUserOrders(ctx, tt.userID).
					Return(tt.orders, tt.errstore).
					Times(1)
			}

			mart := gophermart.New(ctx, cfg.Gophermart, storeMock)
			server, err := rest.New(mart, rest.SetSecretKey([]byte(cfg.Rest.Secret)))
			assert.NoError(t, err)
			engin := server.Engine()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/user/orders", http.NoBody)

			jwtRest := jwt.New([]byte(cfg.Rest.Secret))
			if tt.status != http.StatusUnauthorized {
				signedCookie, err := jwtRest.Create(cookieKey, strconv.Itoa(int(tt.userID)))
				assert.NoError(t, err)
				userCookie := &http.Cookie{
					Name:  "token",
					Value: signedCookie,
					Path:  "/",
				}
				r.AddCookie(userCookie)
				http.SetCookie(w, userCookie)
			}

			engin.ServeHTTP(w, r)

			result := w.Result()

			assert.Equal(t, tt.status, result.StatusCode)

			err = result.Body.Close()
			assert.NoError(t, err)
		})
	}
}

func TestServer_handlerGetUserBalance(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		userID   uint
		balance  model.Balance
		status   int
		errstore error
	}{
		{
			name:    "ok",
			userID:  1,
			balance: model.Balance{ID: 1, UserID: 1, Current: 144, Withdrawn: 1},
			status:  http.StatusOK,
		},
		{
			name:   "unauthorize",
			userID: 1,
			status: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg, err := config.Init()
			assert.NoError(t, err)
			cfg.Gophermart.GorutineEnabled = false

			storeMock := store.NewMockStore(ctrl)
			if tt.name == "ok" {
				storeMock.EXPECT().
					GetUserBalance(ctx, tt.userID).
					Return(tt.balance, tt.errstore).
					Times(1)
			}

			mart := gophermart.New(ctx, cfg.Gophermart, storeMock)
			server, err := rest.New(mart, rest.SetSecretKey([]byte(cfg.Rest.Secret)))
			assert.NoError(t, err)
			engin := server.Engine()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/user/balance", http.NoBody)

			jwtRest := jwt.New([]byte(cfg.Rest.Secret))
			if tt.status != http.StatusUnauthorized {
				signedCookie, err := jwtRest.Create(cookieKey, strconv.Itoa(int(tt.userID)))
				assert.NoError(t, err)
				userCookie := &http.Cookie{
					Name:  "token",
					Value: signedCookie,
					Path:  "/",
				}
				r.AddCookie(userCookie)
				http.SetCookie(w, userCookie)
			}

			engin.ServeHTTP(w, r)

			result := w.Result()

			assert.Equal(t, tt.status, result.StatusCode)

			err = result.Body.Close()
			assert.NoError(t, err)
		})
	}
}

func TestServer_handlerUserBalanceWithdraw(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		userID   uint
		status   int
		order    string
		errstore error
	}{
		{
			name:   "ok",
			userID: 1,
			status: http.StatusOK,
			order:  "2377225624",
		},
		{
			name:   "unauthorize",
			userID: 1,
			status: http.StatusUnauthorized,
			order:  "2377225624",
		},
		{
			name:     "no money",
			userID:   1,
			status:   http.StatusPaymentRequired,
			errstore: errstore.ErrBalansNotEnough,
			order:    "2377225624",
		},
		{
			name:     "uncorrect order number",
			userID:   1,
			status:   http.StatusUnprocessableEntity,
			errstore: gophermart.ErrOrderNumberNotValid,
			order:    "2377225624123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg, err := config.Init()
			assert.NoError(t, err)
			cfg.Gophermart.GorutineEnabled = false

			storeMock := store.NewMockStore(ctrl)
			if tt.name == "ok" || tt.name == "no money" {
				storeMock.EXPECT().
					WithdrawFromUserBalance(ctx, tt.userID, tt.order, float32(1)).
					Return(tt.errstore).
					Times(1)
			}

			mart := gophermart.New(ctx, cfg.Gophermart, storeMock)
			server, err := rest.New(mart, rest.SetSecretKey([]byte(cfg.Rest.Secret)))
			assert.NoError(t, err)
			engin := server.Engine()

			w := httptest.NewRecorder()
			body := strings.NewReader(fmt.Sprintf(`{"order":%q, "sum":%d}`, tt.order, 1))
			r := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", body)

			jwtRest := jwt.New([]byte(cfg.Rest.Secret))
			if tt.status != http.StatusUnauthorized {
				signedCookie, err := jwtRest.Create(cookieKey, strconv.Itoa(int(tt.userID)))
				assert.NoError(t, err)
				userCookie := &http.Cookie{
					Name:  "token",
					Value: signedCookie,
					Path:  "/",
				}
				r.AddCookie(userCookie)
				http.SetCookie(w, userCookie)
			}

			engin.ServeHTTP(w, r)

			result := w.Result()

			assert.Equal(t, tt.status, result.StatusCode)

			err = result.Body.Close()
			assert.NoError(t, err)
		})
	}
}

func TestServer_handlerUserWithdrawals(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		userID   uint
		status   int
		errstore error
	}{
		{
			name:   "ok",
			userID: 1,
			status: http.StatusOK,
		},
		{
			name:     "no content",
			userID:   1,
			status:   http.StatusNoContent,
			errstore: errstore.ErrNotFoundData,
		},
		{
			name:   "unauthorize",
			userID: 1,
			status: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg, err := config.Init()
			assert.NoError(t, err)
			cfg.Gophermart.GorutineEnabled = false

			storeMock := store.NewMockStore(ctrl)
			if tt.name != "unauthorize" {
				storeMock.EXPECT().
					GetUserBalance(ctx, tt.userID).
					Return(model.Balance{ID: 1, UserID: tt.userID}, tt.errstore).
					Times(1)
				if tt.name != "no content" {
					storeMock.EXPECT().
						GetWithdrawalsFromBalance(ctx, uint(1)).
						Return([]*model.WithdrawBalance{{ID: 1, OderNumber: "123", Sum: float32(123)}}, tt.errstore).
						Times(1)
				}
			}

			mart := gophermart.New(ctx, cfg.Gophermart, storeMock)
			server, err := rest.New(mart, rest.SetSecretKey([]byte(cfg.Rest.Secret)))
			assert.NoError(t, err)
			engin := server.Engine()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", http.NoBody)

			jwtRest := jwt.New([]byte(cfg.Rest.Secret))
			if tt.status != http.StatusUnauthorized {
				signedCookie, err := jwtRest.Create(cookieKey, strconv.Itoa(int(tt.userID)))
				assert.NoError(t, err)
				userCookie := &http.Cookie{
					Name:  "token",
					Value: signedCookie,
					Path:  "/",
				}
				r.AddCookie(userCookie)
				http.SetCookie(w, userCookie)
			}

			engin.ServeHTTP(w, r)

			result := w.Result()

			assert.Equal(t, tt.status, result.StatusCode)

			err = result.Body.Close()
			assert.NoError(t, err)
		})
	}
}
