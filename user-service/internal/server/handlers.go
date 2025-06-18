package server

import (
	"errors"
	"github.com/google/uuid"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/azaliaz/bookly/user-service/internal/domain/models"
	"github.com/azaliaz/bookly/user-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/user-service/internal/storage/errors"
)

const adminSecretKey = "your-admin-secret-key"

type request struct {
	Email    string `json:"email" validate:"required,email"`
	CartID   string `json:"cart_id" validate:"required,uuid"`
	Name     string `json:"name"`
	LastName string `json:"lastname"`
	Pass     string `json:"pass" validate:"required,min=8"`
	Age      int    `json:"age" validate:"required,gte=16"`
	Role     string `json:"role" validate:"required"`
	AdminKey string `json:"adminKey"`
}

func (s *Server) Register(ctx *gin.Context) {
	var req request
	log := logger.Get()
	log.Debug().Any("register req", req).Msg("received register data")

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.String(http.StatusBadRequest, "incorrectly entered data")
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	if req.AdminKey == adminSecretKey {
		req.Role = "admin"
	} else {
		req.Role = "user"
	}
	cartID := uuid.New().String()

	uuid, err := s.storage.SaveUser(models.User{
		CartID:   cartID,
		Email:    req.Email,
		Name:     req.Name,
		LastName: req.LastName,
		Pass:     req.Pass,
		Age:      req.Age,
		Role:     req.Role,
	}, req.AdminKey)

	if err != nil {
		ctx.String(http.StatusConflict, "User already exists")
		return
	}
	user, err := s.storage.GetUser(uuid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user after registration"})
		return
	}
	token, err := createJWTToken(uuid, user.Role)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Authorization", token)
	ctx.String(http.StatusCreated, token)
}

func (s *Server) Login(ctx *gin.Context) {
	log := logger.Get()
	var req request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("unmarshal body failed")
		ctx.String(http.StatusBadRequest, "incorrectly entered data")
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	if req.AdminKey == "your-admin-secret-key" {
		req.Role = "admin"
	} else {
		req.Role = "user"
	}
	uuid, err := s.storage.ValidUser(models.User{
		CartID:   req.CartID,
		Email:    req.Email,
		Name:     req.Name,
		LastName: req.LastName,
		Pass:     req.Pass,
		Age:      req.Age,
		Role:     req.Role,
	})
	if err != nil {
		if errors.Is(err, storerrros.ErrUserNoExist) {
			log.Error().Err(err).Msg("user not found")
			ctx.String(http.StatusNotFound, "invalid login or password: %w", err)
			return
		}
		if errors.Is(err, storerrros.ErrInvalidPassword) {
			log.Error().Err(err).Msg("invalid pass")
			ctx.String(http.StatusUnauthorized, err.Error())
			return
		}
		log.Error().Err(err).Msg("validate user failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user, err := s.storage.GetUser(uuid)
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve user")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to fetch user details"})
		return
	}

	if user.CartID == "" {
		newCartID := uuid
		user.CartID = newCartID
		err := s.storage.UpdateUserCartID(user.UID, newCartID)
		if err != nil {
			log.Error().Err(err).Msg("failed to update user with cart ID")
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create cart"})
			return
		}
	}

	token, err := createJWTToken(uuid, user.Role)

	if err != nil {
		log.Error().Err(err).Msg("create jwt failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Header("Authorization", token)
	ctx.String(http.StatusOK, token)
}
func (s *Server) UserInfo(ctx *gin.Context) {
	log := logger.Get()
	uid := ctx.GetString("uid")
	user, err := s.storage.GetUser(uid)

	if err != nil {
		log.Error().Err(err).Msg("failed get user from db")
		if errors.Is(err, storerrros.ErrUserNotFound) {
			ctx.String(http.StatusNotFound, err.Error())
			return
		}
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, user)
}
