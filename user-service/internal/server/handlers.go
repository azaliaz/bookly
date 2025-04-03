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
	Pass     string `json:"pass" validate:"required,min=8"`
	Age      int    `json:"age" validate:"required,gte=16"`
	Role     string `json:"role" validate:"required"`
	AdminKey string `json:"adminKey"`
}

func (s *Server) register(ctx *gin.Context) {
	var req request
	if err := ctx.ShouldBindJSON(&req); err != nil {
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
	cartID := uuid.New().String()
	uuid, err := s.storage.SaveUser(models.User{
		CartID: cartID,
		Email:  req.Email,
		Pass:   req.Pass,
		Age:    req.Age,
		Role:   req.Role,
	}, req.AdminKey)

	if err != nil {
		ctx.String(http.StatusConflict, "User already exists")
		return
	}

	token, err := createJWTToken(uuid, req.Role)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Authorization", token)
	ctx.String(http.StatusCreated, token)
}

func (s *Server) login(ctx *gin.Context) {
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
		Email: req.Email,
		Pass:  req.Pass,
		Age:   req.Age,
		Role:  req.Role,
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
		// Создание новой корзины
		newCartID := uuid
		user.CartID = newCartID
		err := s.storage.UpdateUserCartID(user.UID, newCartID) // Передаем правильный UUID пользователя
		if err != nil {
			log.Error().Err(err).Msg("failed to update user with cart ID")
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create cart"})
			return
		}
	}

	token, err := createJWTToken(uuid, req.Role)
	if err != nil {
		log.Error().Err(err).Msg("create jwt failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Header("Authorization", token)
	ctx.String(http.StatusOK, "user %s are logined", uuid)
}
func (s *Server) userInfo(ctx *gin.Context) {

	log := logger.Get()
	//извлечение uid
	uid := ctx.GetString("uid")
	//получение информации о пользователе
	user, err := s.storage.GetUser(uid)

	//обработка ошибок
	if err != nil {
		log.Error().Err(err).Msg("failed get user from db")
		if errors.Is(err, storerrros.ErrUserNotFound) { //пользователь не найден
			ctx.String(http.StatusNotFound, err.Error())
			return
		}
		ctx.String(http.StatusInternalServerError, err.Error()) //Если это другая ошибка (например, проблемы с БД)
		return
	}
	//возвращение данных пользователя
	ctx.JSON(http.StatusFound, user)
	//ctx.JSON(http.StatusOK, user)

}
