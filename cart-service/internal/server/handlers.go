package server

import (
	// "context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/azaliaz/bookly/cart-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/cart-service/internal/storage/errors"
)

//	func (s *Server) createCart(ctx *gin.Context) {
//		log := logger.Get()
//
//		// Проверка наличия uid
//		_, exist := ctx.Get("uid")
//		if !exist {
//			log.Error().Msg("user ID not found")
//			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
//			return
//		}
//
//		// Получаем ID пользователя
//		userID := ctx.Param("user_id")
//
//		// Создаем корзину
//		cartID, err := s.storage.CreateCart(userID)
//		if err != nil {
//			log.Error().Err(err).Msg("failed to create cart")
//			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//			return
//		}
//
//		// Возвращаем cartID
//		ctx.JSON(http.StatusCreated, gin.H{"cartID": cartID})
//	}
func (s *Server) addBookToCart(ctx *gin.Context) {
	log := logger.Get()

	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	cartID := ctx.Query("cart_id")
	bookID := ctx.Query("book_id")
	quantity := ctx.DefaultQuery("quantity", "1")

	if cartID == "" {
		log.Error().Msg("cart_id is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cart_id is required"})
		return
	}
	if bookID == "" {
		log.Error().Msg("book_id is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "book_id is required"})
		return
	}
	quantityInt, err := strconv.Atoi(quantity)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quantity"})
		return
	}

	if err := s.storage.AddBookToCart(cartID, bookID, quantityInt); err != nil {
		log.Error().Err(err).Msg("failed to add book to cart")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "book added to cart"})
}

func (s *Server) getCartItems(ctx *gin.Context) {
	log := logger.Get()

	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	cartID := ctx.Param("cart_id")

	items, err := s.storage.GetCartItems(cartID)
	if err != nil {
		if errors.Is(err, storerrros.ErrCartNotExist) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Cart not found"})
			return
		}
		log.Error().Err(err).Msg("failed to get cart items")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, items)
}
func (s *Server) removeBookFromCart(ctx *gin.Context) {
	log := logger.Get()

	itemID := ctx.Param("itemID")
	if itemID == "" {
		log.Error().Msg("item_id is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}

	if err := s.storage.RemoveBookFromCart(itemID); err != nil {
		if err.Error() == "book not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "book not found in cart"})
			return
		}
		log.Error().Err(err).Msg("failed to remove book from cart")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove book from cart"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "book removed from cart"})
}

func (s *Server) clearCart(ctx *gin.Context) {
	log := logger.Get()
	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}
	cartID := ctx.Param("cart_id")

	if err := s.storage.ClearCart(cartID); err != nil {
		log.Error().Err(err).Msg("failed to clear cart")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear cart"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "cart cleared"})
}
