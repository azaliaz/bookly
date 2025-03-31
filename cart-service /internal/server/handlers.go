package server

import (
	// "context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/azaliaz/bookly/cart-service/internal/domain/models"
	"github.com/azaliaz/bookly/cart-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/cart-service/internal/storage/errors"
)

func (s *Server) addBookToCart(ctx *gin.Context) {
	log := logger.Get()

	// Проверка наличия uid
	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Извлечение идентификаторов книги и количества из тела запроса
	var request struct {
		BookID string `json:"book_id"`
		Count  int    `json:"count"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("invalid input data")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid input data"})
		return
	}

	// Добавление книги в корзину
	err := s.storage.AddBookToCart(ctx.GetString("uid"), request.BookID, request.Count)
	if err != nil {
		log.Error().Err(err).Msg("failed to add book to cart")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем успешный ответ
	ctx.JSON(http.StatusOK, gin.H{"message": "book added to cart"})
}

func (s *Server) getCartItems(ctx *gin.Context) {
	log := logger.Get()

	// Проверка наличия uid
	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Получение списка книг в корзине
	cartItems, err := s.storage.GetCartItems(ctx.GetString("uid"))
	if err != nil {
		log.Error().Err(err).Msg("failed to get cart items")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем успешный ответ с книгами
	ctx.JSON(http.StatusOK, cartItems)
}
func (s *Server) removeBookFromCart(ctx *gin.Context) {
	log := logger.Get()

	// Проверка наличия uid
	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Получение идентификатора книги из параметра запроса
	bookID := ctx.Param("id")
	if bookID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing book ID"})
		return
	}

	// Удаление книги из корзины
	err := s.storage.RemoveBookFromCart(ctx.GetString("uid"), bookID)
	if err != nil {
		log.Error().Err(err).Msg("failed to remove book from cart")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем успешный ответ
	ctx.JSON(http.StatusOK, gin.H{"message": "book removed from cart"})
}
