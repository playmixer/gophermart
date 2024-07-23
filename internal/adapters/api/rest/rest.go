package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/playmixer/gophermart/docs"
	"github.com/playmixer/gophermart/internal/adapters/store/model"
)

var (
	errInvalidAuthCookie = errors.New("invalid authorization cookie")

	cookieName = "token"
)

type Gophermart interface {
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
	service Gophermart
	address string
	secret  []byte
}

type Option func(*Server)

func Logger(log *zap.Logger) Option {
	return func(s *Server) {
		s.log = log
	}
}

func Configure(cfg *Config) Option {
	return func(s *Server) {
		if cfg != nil {
			s.secret = []byte(cfg.Secret)
			s.address = cfg.Address
		}
	}
}

func New(service Gophermart, options ...Option) (*Server, error) {
	s := &Server{
		log:     zap.NewNop(),
		service: service,
	}

	for _, opt := range options {
		opt(s)
	}

	return s, nil
}

//	@title			«Гофермарт»
//	@version		1.0
//	@description	Накопительная система лояльности «Гофермарт».

//	@host		localhost:8080
//	@BasePath	/

func (s *Server) CreateEngine() *gin.Engine {
	r := gin.New()
	r.Use(s.Logger())
	apiUser := r.Group("/api/user")
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
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

func (s *Server) Run() error {
	r := s.CreateEngine()
	if err := r.Run(s.address); err != nil {
		return fmt.Errorf("server stopped with error: %w", err)
	}

	return nil
}

func (s *Server) CreateJWT(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": strconv.Itoa(int(userID)),
	})
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed signe token: %w", err)
	}

	return tokenString, nil
}

func (s *Server) verifyJWT(signedData string) (string, bool) {
	token, err := jwt.Parse(signedData, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unknown signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		s.log.Debug("failed parse jwt token", zap.Error(err))
		return "", false
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if uniqueID, ok := claims["userID"].(string); ok {
			if uniqueID != "" {
				return uniqueID, true
			}
		}
	}

	return "", false
}

func (s *Server) checkAuth(c *gin.Context) (userID uint, err error) {
	var ok bool
	var userIDS string
	cookieUserID, err := c.Request.Cookie(cookieName)
	if err == nil {
		userIDS, ok = s.verifyJWT(cookieUserID.Value)
	}
	if err != nil {
		return 0, fmt.Errorf("failed reade user cookie: %w %w", errInvalidAuthCookie, err)
	}
	if !ok {
		return 0, fmt.Errorf("unverify usercookie: %w", errInvalidAuthCookie)
	}
	userID64, err := strconv.ParseUint(userIDS, 10, 32)
	if err != nil {
		s.log.Error("can't convert string userID to uint", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
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

	signedCookie, err := s.CreateJWT(user.ID)
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
