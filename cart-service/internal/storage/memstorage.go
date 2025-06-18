package storage

import (
	"fmt"
	// "golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"

	"github.com/azaliaz/bookly/cart-service/internal/domain/models"
	//"github.com/azaliaz/bookly/cart-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/cart-service/internal/storage/errors"
)

type MemStorage struct {
	carts     map[string]models.Cart
	cartItems map[string][]models.CartItem
	books     map[string]models.Book
}

func New() *MemStorage {
	return &MemStorage{
		carts:     make(map[string]models.Cart),
		cartItems: make(map[string][]models.CartItem),
		books:     make(map[string]models.Book),
	}
}
func (ms *MemStorage) CreateCart(UID string) (string, error) {
	cartID := uuid.New().String()
	ms.carts[cartID] = models.Cart{CartID: cartID, UID: UID}
	return cartID, nil
}
func (ms *MemStorage) AddBookToCart(cartID, BID string) error {
	// Проверяем, существует ли книга
	if _, exists := ms.books[BID]; !exists {
		return fmt.Errorf("book not found")
	}

	// Создаем корзину, если ее нет
	if _, exists := ms.carts[cartID]; !exists {
		ms.carts[cartID] = models.Cart{}
	}

	// Проверяем, есть ли уже такая книга в корзине — если есть, ничего не делаем
	for _, item := range ms.cartItems[cartID] {
		if item.BID == BID {
			return nil
		}
	}

	// Добавляем книгу в корзину
	ms.cartItems[cartID] = append(ms.cartItems[cartID], models.CartItem{
		ItemID: uuid.New().String(),
		CartID: cartID,
		BID:    BID,
	})

	return nil
}

func (ms *MemStorage) GetCartItems(cartID string) ([]models.CartItem, error) {
	items, exists := ms.cartItems[cartID]
	if !exists {
		return nil, storerrros.ErrCartNotExist
	}
	return items, nil
}
func (ms *MemStorage) RemoveBookFromCart(itemID string) error {
	for cartID, items := range ms.cartItems {
		for i, item := range items {
			if item.ItemID == itemID {
				// Просто удаляем книгу из корзины
				ms.cartItems[cartID] = append(items[:i], items[i+1:]...)
				return nil
			}
		}
	}

	return storerrros.ErrBookNoExist
}

func (ms *MemStorage) ClearCart(cartID string) error {
	if _, exists := ms.carts[cartID]; !exists {
		return storerrros.ErrCartNotExist
	}

	delete(ms.cartItems, cartID)
	return nil
}
