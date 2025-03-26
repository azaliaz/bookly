package server

import (
	"context"
	"errors"
	"time"
	"net/http"
	"strings" 

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/golang-jwt/jwt/v4"

	"github.com/azaliaz/bookly/user-service/internal/config"
	"github.com/azaliaz/bookly/user-service/internal/domain/models"
	"github.com/azaliaz/bookly/user-service/internal/logger"
)

var SecretKey = "VerySecurKey2000Cat" //nolint:gochecknoglobals //demo var

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

type Storage interface {
	SaveUser(models.User) (string, error)
	ValidUser(models.User) (string, error)
	GetUser(string) (models.User, error)
}

type Server struct {
	serv    *http.Server
	valid   *validator.Validate
	storage Storage
	delChan chan struct{}
	ErrChan chan error
}

func New(cfg config.Config, stor Storage) *Server {
	server := http.Server{ //nolint:gosec // not today
		Addr: cfg.Addr,
	}
	valid := validator.New()
	return &Server{
		serv:    &server,
		valid:   valid,
		storage: stor,
		delChan: make(chan struct{}, 10), //nolint:mnd //todo
		ErrChan: make(chan error),
	}
}

func (s *Server) ShutdownServer() error {
	return s.serv.Shutdown(context.Background())
}

func (s *Server) Run(ctx context.Context) error {
	log := logger.Get()
	router := gin.Default()
	router.GET("/", func(ctx *gin.Context) { ctx.String(http.StatusOK, "Hello") })
	users := router.Group("/users")
	{
		users.GET("/info", s.JWTAuthMiddleware(), s.userInfo)
		users.POST("/register", s.register)
		users.POST("/login", s.login)
	}
	
	s.serv.Handler = router
	log.Debug().Msg("start delete listener")
	//go s.deleter(ctx)
	log.Info().Str("host", s.serv.Addr).Msg("server started")
	if err := s.serv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func (s *Server) Close() error {
	return s.serv.Shutdown(context.TODO())
}

func (s *Server) JWTAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		log := logger.Get()

		// Извлечение токена из заголовка Authorization
		tokenHeader := ctx.GetHeader("Authorization")
		if tokenHeader == "" {
			ctx.String(http.StatusUnauthorized, "Authorization header is required")
			return
		}

		// Разделение токена по пробелу, ожидаем формат "Bearer <token>"
		tokenParts := strings.Split(tokenHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			ctx.String(http.StatusUnauthorized, "Invalid token format")
			return
		}
		tokenStr := tokenParts[1]

		// Проверка валидности токена
		UID, err := validToken(tokenStr)
		if err != nil {
			log.Error().Err(err).Msg("validate jwt failed")
			ctx.String(http.StatusUnauthorized, "Invalid token")
			return
		}

		// Сохранение UID пользователя в контексте
		ctx.Set("uid", UID)
		// Передача управления следующему обработчику
		ctx.Next()
	}
}

func validToken(tokenStr string) (string, error) {
	claims := &Claims{}
	// Разбираем JWT токен и проверяем его
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", ErrInvalidToken
	}
	return claims.UserID, nil
}

func createJWTToken(uid string) (string, error) {
	// Создаем новый токен сClaims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 3)), // Устанавливаем время жизни токена
		},
		UserID: uid,
	})
	// Подписываем токен
	tokenStr, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

