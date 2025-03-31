package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"strings"

	"github.com/azaliaz/bookly/book-service/internal/config"
	"github.com/azaliaz/bookly/book-service/internal/domain/models"
	"github.com/azaliaz/bookly/book-service/internal/logger"
)

var SecretKey = "VerySecurKey2000Cat"

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	jwt.RegisteredClaims
	UserID string
	Role   string
}

type Storage interface {
	SaveBook(models.Book) error
	SaveBooks([]models.Book) error
	GetBooks() ([]models.Book, error)
	GetBook(string) (models.Book, error)
	SetDeleteStatus(string) error
	DeleteBooks(string) error
	DeleteBook(string) error
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
	books := router.Group("/books")
	{
		books.GET("/:id", s.JWTAuthMiddleware(), s.bookInfo)
		books.DELETE("/:id", s.JWTAuthMiddleware(), s.removeBook)
		books.GET("/", s.JWTAuthMiddleware(), s.allBooks)
	}
	router.POST("/add-book", s.JWTAuthRoleMiddleware("admin"), s.addBook)
	router.POST("/add-books", s.JWTAuthRoleMiddleware("admin"), s.addBooks)
	router.POST("/book-return", s.JWTAuthMiddleware(), s.bookReturn)

	s.serv.Handler = router
	log.Debug().Msg("start delete listener")
	// go s.deleter(ctx)
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
		fmt.Printf("role: %s\n", Role)
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
	fmt.Printf("Role %s uid: %s \n", claims.Role, claims.UserID)

	return claims.UserID, claims.Role, nil
}
