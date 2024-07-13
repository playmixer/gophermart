package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/playmixer/gophermart/internal/adapters/store/errstore"
	"github.com/playmixer/gophermart/internal/adapters/store/model"
	"github.com/playmixer/gophermart/internal/core/gophermart"
	"go.uber.org/zap"
)

var (
	msgErrorCloseBody = "failed close body request"
)

//	@Summary	Register user
//	@Schemes
//	@Description	registration user
//	@Tags			auth
//	@Accept			json
//	@Produce		plain
//	@Param			registration	body	tRegistration	true	"registration"
//	@Success		200				"пользователь успешно зарегистрирован и аутентифицирован"
//	@failure		400				"неверный формат запроса"
//	@failure		409				"логин уже занят"
//	@failure		500				"внутренняя ошибка сервера"
//	@Router			/api/user/register [post]
func (s *Server) handlerRegister(c *gin.Context) {
	ctx := c.Request.Context()

	unauthorize(c)

	bBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.log.Error("failed read body", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := c.Request.Body.Close(); err != nil {
			s.log.Error(msgErrorCloseBody, zap.Error(err))
		}
	}()

	jBody := tRegistration{}

	err = json.Unmarshal(bBody, &jBody)
	if err != nil {
		s.log.Error("failed parse body", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = s.service.Register(ctx, jBody.Login, jBody.Password); err != nil {
		if errors.Is(err, errstore.ErrLoginNotUnique) {
			c.Writer.WriteHeader(http.StatusConflict)
			return
		}
		if errors.Is(err, gophermart.ErrLoginNotValid) || errors.Is(err, gophermart.ErrPasswordNotValid) {
			c.Writer.WriteHeader(http.StatusBadRequest)
			return
		}

		s.log.Error("failed register user", zap.Error(err))
		// c.Writer.WriteHeader(http.StatusInternalServerError)
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err = s.authorization(c, jBody.Login, jBody.Password); err != nil {
		if errors.Is(err, gophermart.ErrLoginNotValid) || errors.Is(err, gophermart.ErrPasswordNotValid) {
			c.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
		if errors.Is(err, gophermart.ErrPasswordNotEquale) || errors.Is(err, errstore.ErrNotFoundData) {
			c.Writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		s.log.Error("authorization failed", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}

//	@Summary	Login user
//	@Schemes
//	@Description	authorization
//	@Tags			auth
//	@Accept			json
//	@Produce		plain
//	@Param			auth	body	tAuthorization	true	"auth"
//	@Success		200		"пользователь успешно аутентифицирован"
//	@failure		400		"неверный формат запроса"
//	@failure		401		"неверная пара логин/пароль"
//	@failure		500		"внутренняя ошибка сервера"
//	@Router			/api/user/login [post]
func (s *Server) handlerLogin(c *gin.Context) {
	unauthorize(c)

	bBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.log.Error("failed read body", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := c.Request.Body.Close(); err != nil {
			s.log.Error(msgErrorCloseBody, zap.Error(err))
		}
	}()

	jBody := tAuthorization{}

	err = json.Unmarshal(bBody, &jBody)
	if err != nil {
		s.log.Error("failed parse body", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = s.authorization(c, jBody.Login, jBody.Password); err != nil {
		if errors.Is(err, gophermart.ErrLoginNotValid) || errors.Is(err, gophermart.ErrPasswordNotValid) {
			c.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
		if errors.Is(err, gophermart.ErrPasswordNotEquale) || errors.Is(err, errstore.ErrNotFoundData) {
			c.Writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		s.log.Error("authorization failed", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}

//	@Summary	upload user order
//	@Schemes
//	@Description	upload order
//	@Tags			order
//	@Accept			plain
//	@Produce		plain
//	@Param			order_id	body	integer	true	"order_id"
//	@Success		200			"номер заказа уже был загружен этим пользователем"
//	@Success		202			"новый номер заказа принят в обработку"
//	@failure		400			"неверный формат запроса"
//	@failure		401			"пользователь не авторизован"
//	@failure		409			"номер заказа уже был загружен другим пользователем"
//	@failure		422			"неверный формат номера заказа"
//	@failure		500			"внутренняя ошибка сервера"
//	@Router			/api/user/orders [post]
func (s *Server) handlerLoadUserOrders(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := s.checkAuth(c)
	if err != nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	bBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := c.Request.Body.Close(); err != nil {
			s.log.Error(msgErrorCloseBody, zap.Error(err))
		}
	}()

	orderNumber := string(bBody)
	if orderNumber == "" {
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.service.UploadOrder(ctx, userID, orderNumber)
	if err != nil {
		if errors.Is(err, gophermart.ErrOrderNumberNotValid) {
			c.Writer.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, errstore.ErrOrderWasCreatedAnotherUser) {
			c.Writer.WriteHeader(http.StatusConflict)
			return
		}
		if errors.Is(err, errstore.ErrOrderWasCreatedByUser) {
			c.Writer.WriteHeader(http.StatusOK)
			return
		}

		s.log.Error("failed upload order number", zap.String("orderNumber", orderNumber), zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Writer.WriteHeader(http.StatusAccepted)
}

//	@Summary	List user orders
//	@Schemes
//	@Description	get user orders
//	@Tags			order
//	@Accept			plain
//	@Produce		json
//	@Success		200	"успешная обработка запроса"
//	@Success		204	"нет данных для ответа"
//	@failure		401	"пользователь не авторизован"
//	@failure		500	"внутренняя ошибка сервера"
//	@Router			/api/user/orders [get]
func (s *Server) handlerGetUserOrders(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := s.checkAuth(c)
	if err != nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := s.service.GetUserOrders(ctx, userID)
	if err != nil {
		if errors.Is(err, errstore.ErrNotFoundData) {
			c.Writer.WriteHeader(http.StatusNoContent)
			return
		}

		s.log.Error("failed get orders by user", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := []tOrderByUser{}
	for _, order := range orders {
		resOrder := tOrderByUser{
			Number:     order.Number,
			Status:     order.Status,
			uploadedAt: order.UpdatedAt,
		}
		if order.Status == model.OrderStateProcessed {
			_accrual := &order.Accrual
			resOrder.Accrual = _accrual
		}
		resOrder.Prepare()
		response = append(response, resOrder)
	}
	sort.Slice(response, func(i, j int) bool {
		return response[i].uploadedAt.Sub(response[j].uploadedAt) < 0
	})
	c.JSON(http.StatusOK, response)
}

//	@Summary	User balance
//	@Schemes
//	@Description	get user balance
//	@Tags			balance
//	@Accept			plain
//	@Produce		json
//	@Success		200	"успешная обработка запроса"
//	@failure		401	"пользователь не авторизован"
//	@failure		500	"внутренняя ошибка сервера"
//	@Router			/api/user/balance [get]
func (s *Server) handlerGetUserBalance(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := s.checkAuth(c)
	if err != nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	balance, err := s.service.GetUserBalance(ctx, userID)
	if err != nil {
		if errors.Is(err, errstore.ErrNotFoundData) {
			c.JSON(http.StatusOK, tBalanceByUser{
				Current:   0,
				Withdrawn: 0,
			})
		}
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, tBalanceByUser{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	})
}

//	@Summary	Withdraw from user balans
//	@Schemes
//	@Description	Withdraw from user balans
//	@Tags			balance
//	@Accept			json
//	@Param			withdraw	body	tWithdraw	true	"withdraw"
//	@Produce		json
//	@Success		200	"успешная обработка запроса"
//	@failure		401	"пользователь не авторизован"
//	@failure		402	"на счету недостаточно средств"
//	@failure		422	"неверный номер заказа"
//	@failure		500	"внутренняя ошибка сервера"
//	@Router			/api/user/balance/withdraw [post]
func (s *Server) handlerUserBalanceWithdraw(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := s.checkAuth(c)
	if err != nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	bBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.log.Error("failed read body", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := c.Request.Body.Close(); err != nil {
			s.log.Error(msgErrorCloseBody, zap.Error(err))
		}
	}()

	withdraw := tWithdraw{}
	err = json.Unmarshal(bBody, &withdraw)
	if err != nil {
		s.log.Error("failed marshal body", zap.String("body", string(bBody)), zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.service.WithdrawFromBalanceUser(ctx, userID, withdraw.Order, withdraw.Sum)
	if err != nil {
		if errors.Is(err, gophermart.ErrOrderNumberNotValid) {
			c.Writer.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, errstore.ErrBalansNotEnough) {
			c.Writer.WriteHeader(http.StatusPaymentRequired)
			return
		}

		s.log.Error("failed withdraw balance", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
}

//	@Summary	Withdraw from user balans
//	@Schemes
//	@Description	Withdraw from user balans
//	@Tags			balance
//	@Accept			plain
//	@Produce		json
//	@Success		200	{array}	tWithdrawBalance	"успешная обработка запроса"
//	@failure		204	"нет ни одного списания"
//	@failure		401	"пользователь не авторизован"
//	@failure		500	"внутренняя ошибка сервера"
//	@Router			/api/user/withdrawals [get]
func (s *Server) handlerUserWithdrawals(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := s.checkAuth(c)
	if err != nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	withdrawals, err := s.service.GetWithdrawalsByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, errstore.ErrNotFoundData) {
			c.Writer.WriteHeader(http.StatusNoContent)
			return
		}

		s.log.Error("failed getting withdrawals by user", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	result := []tWithdrawBalance{}
	for _, withdrawal := range withdrawals {
		wd := tWithdrawBalance{
			Order:       withdrawal.OderNumber,
			Sum:         withdrawal.Sum,
			processedAt: withdrawal.UpdatedAt,
		}
		result = append(result, *wd.Prepare())
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].processedAt.Sub(result[j].processedAt) < 0
	})

	c.JSON(http.StatusOK, result)
}
