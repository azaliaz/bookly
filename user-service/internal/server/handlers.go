package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/azaliaz/bookly/user-service/internal/domain/models"
	"github.com/azaliaz/bookly/user-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/user-service/internal/storage/errors"
)

func (s *Server) register(ctx *gin.Context) {
	log := logger.Get()
	var user models.User
	if err := ctx.ShouldBindBodyWithJSON(&user); err != nil {
		log.Error().Err(err).Msg("unmarshal body failed")
		ctx.String(http.StatusBadRequest, "incorrectly entered data")
		return
	}
	if err := s.valid.Struct(user); err != nil {
		log.Error().Err(err).Msg("validate user failed")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	uuid, err := s.storage.SaveUser(user)
	if err != nil {
		if errors.Is(err, storerrros.ErrUserExists) {
			log.Error().Msg(err.Error())
			ctx.String(http.StatusConflict, err.Error())
			return
		}
		log.Error().Err(err).Msg("save user failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Debug().Str("uuid", uuid).Send()
	token, err := createJWTToken(uuid)
	if err != nil {
		log.Error().Err(err).Msg("create jwt failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Header("Authorization", token)
	ctx.String(http.StatusCreated, token)
}

func (s *Server) login(ctx *gin.Context) {
	log := logger.Get()
	var user models.User
	if err := ctx.ShouldBindBodyWithJSON(&user); err != nil {
		log.Error().Err(err).Msg("unmarshal body failed")
		ctx.String(http.StatusBadRequest, "incorrectly entered data")
		return
	}
	if err := s.valid.Struct(user); err != nil {
		log.Error().Err(err).Msg("validate user failed")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	uuid, err := s.storage.ValidUser(user)
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
	token, err := createJWTToken(uuid)
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
