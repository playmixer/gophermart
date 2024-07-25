package rest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/playmixer/gophermart/docs"
	"github.com/playmixer/gophermart/internal/adapters/store/errstore"
	"github.com/playmixer/gophermart/internal/adapters/store/model"
	"github.com/playmixer/gophermart/internal/core/gophermart"
	"github.com/playmixer/gophermart/pkg/jwt"
)

var (
	cookieName = "token"
	cookieKey  = "UserID"
)

type gophermartI interface {
	Register(ctx context.Context, login, password string) error
	Authorization(ctx context.Context, login, password string) (model.User, error)
	UploadOrder(ctx context.Context, userID uint, orderNumber string) error
	GetUserOrders(ctx context.Context, userID uint) ([]*model.Order, error)
	GetUserBalance(ctx context.Context, userID uint) (model.Balance, error)
	WithdrawFromBalanceUser(ctx context.Context, userID uint, order string, sum float32) error
	GetWithdrawalsByUser(ctx context.Context, userID uint) ([]*model.WithdrawBalance, error)
}

type Server struct {
	log     *zap.Logger
	engine  *gin.Engine
	service gophermartI
	address string
	secret  []byte
}

type Option func(*Server)

func Logger(log *zap.Logger) Option {
	return func(s *Server) {
		s.log = log
	}
}

func SetAddress(address string) Option {
	return func(s *Server) {
		s.address = address
	}
}

func SetSecretKey(key []byte) Option {
	return func(s *Server) {
		s.secret = key
	}
}

//	@title			«Гофермарт»
//	@version		1.0
//	@description	Накопительная система лояльности «Гофермарт».
//	@host			localhost:8080
//	@BasePath		/

func New(service gophermartI, options ...Option) (*Server, error) {
	s := &Server{
		log:     zap.NewNop(),
		service: service,
	}

	s.engine = gin.New()
	s.engine.Use(
		s.Logger(),
		s.GzipDecompress(),
	)
	apiUser := s.engine.Group("/api/user")
	apiUser.Use(s.GzipCompress())
	{
		apiUser.POST("/register", s.handlerRegister)
		apiUser.POST("/login", s.handlerLogin)

		authAPIUser := apiUser.Group("/")
		authAPIUser.Use(s.Authentication())
		{
			authAPIUser.POST("/orders", s.handlerLoadUserOrders)
			authAPIUser.GET("/orders", s.handlerGetUserOrders)
			authAPIUser.GET("/balance", s.handlerGetUserBalance)
			authAPIUser.POST("/balance/withdraw", s.handlerUserBalanceWithdraw)
			authAPIUser.GET("/withdrawals", s.handlerUserWithdrawals)
		}
	}
	s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	for _, opt := range options {
		opt(s)
	}

	return s, nil
}

func (s *Server) Engine() *gin.Engine {
	return s.engine
}

func (s *Server) Run() error {
	if err := s.engine.Run(s.address); err != nil {
		return fmt.Errorf("server stopped with error: %w", err)
	}

	return nil
}

func (s *Server) checkAuth(c *gin.Context) (userID uint, err error) {
	var ok bool
	var userIDS string
	cookieUserID, err := c.Request.Cookie(cookieName)
	if err != nil {
		return 0, fmt.Errorf("failed reade user cookie: %w %w", err, errUnauthorize)
	}

	jwtRest := jwt.New(s.secret)
	userIDS, ok, err = jwtRest.Verify(cookieUserID.Value, cookieKey)
	if err != nil {
		return 0, fmt.Errorf("failed verify token: %w %w", err, errUnauthorize)
	}

	if !ok {
		return 0, fmt.Errorf("unverify usercookie: %w", errUnauthorize)
	}

	userID64, err := strconv.ParseUint(userIDS, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("can't convert string userID to uint: %w", err)
	}

	return uint(userID64), nil
}

func unauthorize(c *gin.Context) {
	userCookie := &http.Cookie{
		Name:  cookieName,
		Value: "",
		Path:  "/",
	}
	c.Request.AddCookie(userCookie)
	http.SetCookie(c.Writer, userCookie)
}

func (s *Server) authorization(c *gin.Context, login, password string) error {
	ctx := c.Request.Context()
	var err error
	var user model.User
	if user, err = s.service.Authorization(ctx, login, password); err != nil {
		return fmt.Errorf("failed authorization: %w", err)
	}

	jwtRest := jwt.New(s.secret)
	signedCookie, err := jwtRest.Create(cookieKey, strconv.Itoa(int(user.ID)))
	if err != nil {
		return fmt.Errorf("can't create cookie data: %w", err)
	}

	userCookie := &http.Cookie{
		Name:  cookieName,
		Value: signedCookie,
		Path:  "/",
	}
	c.Request.AddCookie(userCookie)
	http.SetCookie(c.Writer, userCookie)

	return nil
}

func (s *Server) login(c *gin.Context, login, password string) (int, string) {
	if err := s.authorization(c, login, password); err != nil {
		if errors.Is(err, gophermart.ErrLoginNotValid) || errors.Is(err, gophermart.ErrPasswordNotValid) {
			if errors.Is(err, gophermart.ErrLoginNotValid) {
				return http.StatusBadRequest, "Не верный формат логина"
			}
			if errors.Is(err, gophermart.ErrPasswordNotValid) {
				return http.StatusBadRequest, "Не верный формат пароля"
			}
			return http.StatusInternalServerError, ""
		}
		if errors.Is(err, gophermart.ErrPasswordNotEquale) || errors.Is(err, errstore.ErrNotFoundData) {
			return http.StatusUnauthorized, ""
		}
		s.log.Error("authorization failed", zap.Error(err))
		return http.StatusInternalServerError, ""
	}
	return http.StatusOK, ""
}

func (s *Server) readBody(c *gin.Context) ([]byte, int) {
	bBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.log.Error("failed read body", zap.Error(err))
		return []byte{}, http.StatusInternalServerError
	}
	defer func() {
		if err := c.Request.Body.Close(); err != nil {
			s.log.Error(msgErrorCloseBody, zap.Error(err))
		}
	}()
	return bBody, 0
}
