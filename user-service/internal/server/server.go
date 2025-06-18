package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-contrib/cors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/golang-jwt/jwt/v4"

	"github.com/azaliaz/bookly/user-service/internal/config"
	"github.com/azaliaz/bookly/user-service/internal/domain/models"
	"github.com/azaliaz/bookly/user-service/internal/logger"
)

//go:generate mockgen -source=server.go -destination=./mocks/service_mock.go -package=mocks
var SecretKey = "VerySecurKey2000Cat" //nolint:gochecknoglobals //demo var

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	jwt.RegisteredClaims
	UserID string
	Role   string
}

type Storage interface {
	SaveUser(models.User, string) (string, error)
	ValidUser(models.User) (string, error)
	GetUser(string) (models.User, error)
	UpdateUserCartID(string, string) error
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
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	router.GET("/", func(ctx *gin.Context) { ctx.String(http.StatusOK, "Hello") })
	users := router.Group("/users")
	{
		users.GET("/info", s.JWTAuthMiddleware(), s.UserInfo)
		users.POST("/register", s.Register)
		users.POST("/login", s.Login)
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
		tokenHeader := ctx.GetHeader("Authorization")
		if tokenHeader == "" {
			ctx.String(http.StatusUnauthorized, "Authorization header is required")
			return
		}

		tokenParts := strings.Split(tokenHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			ctx.String(http.StatusUnauthorized, "Invalid token format")
			return
		}

		UID, Role, err := validToken(tokenParts[1])
		if err != nil {
			ctx.String(http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx.Set("uid", UID)
		ctx.Set("role", Role)
		ctx.Next()
	}
}
func (s *Server) JWTAuthRoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenHeader := ctx.GetHeader("Authorization")
		if tokenHeader == "" {
			ctx.String(http.StatusUnauthorized, "Authorization header is required")
			ctx.Abort()
			return
		}

		tokenParts := strings.Split(tokenHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			ctx.String(http.StatusUnauthorized, "Invalid token format")
			ctx.Abort()
			return
		}

		UID, Role, err := validToken(tokenParts[1])
		if err != nil {
			ctx.String(http.StatusUnauthorized, "Invalid token")
			ctx.Abort()
			return
		}

		if len(roles) > 0 {
			isAllowed := false
			for _, allowedRole := range roles {
				if Role == allowedRole {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				ctx.String(http.StatusForbidden, "Access denied")
				ctx.Abort()
				return
			}
		}
		fmt.Printf("jwtadmin uid: %s, role: %s\n", UID, Role)
		ctx.Set("uid", UID)
		ctx.Set("role", Role)
		ctx.Next()
	}
}

func validToken(tokenStr string) (string, string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		fmt.Printf("Parsed Claims - Role: %s, UID: %s\n", claims.Role, claims.UserID)
		return []byte(SecretKey), nil
	})
	if err != nil || !token.Valid {
		return "", "", ErrInvalidToken
	}
	fmt.Printf("Claims - Role: %s, UID: %s\n", claims.Role, claims.UserID)
	return claims.UserID, claims.Role, nil
}
func createJWTToken(uid, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 3)),
		},
		UserID: uid,
		Role:   role,
	})
	return token.SignedString([]byte(SecretKey))
}
