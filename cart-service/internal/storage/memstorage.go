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
func (ms *MemStorage) AddBookToCart(cartID, BID string, quantity int) error {
	// Проверяем, есть ли книга в базе
	book, exists := ms.books[BID]
	if !exists {
		return fmt.Errorf("book not found")
	}

	// Проверяем, есть ли нужное количество
	if book.Count < quantity {
		return fmt.Errorf("not enough books in stock")
	}

	// Уменьшаем количество книг в хранилище
	book.Count -= quantity
	ms.books[BID] = book // Обновляем количество книги в хранилище

	// Проверяем, есть ли корзина для данного cartID, если нет - создаем
	if _, exists := ms.carts[cartID]; !exists {
		ms.carts[cartID] = models.Cart{} // Создаем пустую корзину
	}

	// Проверяем, есть ли книга в корзине
	for i, item := range ms.cartItems[cartID] {
		if item.BID == BID {
			// Если книга уже в корзине, увеличиваем количество
			ms.cartItems[cartID][i].Quantity += quantity
			return nil
		}
	}

	// Если книги в корзине нет, добавляем новую запись
	ms.cartItems[cartID] = append(ms.cartItems[cartID], models.CartItem{
		ItemID:   uuid.New().String(), // Генерируем новый ItemID
		CartID:   cartID,
		BID:      BID,
		Quantity: quantity,
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
				// Возвращаем книгу в общий список
				book := ms.books[item.BID]
				book.Count += item.Quantity
				ms.books[item.BID] = book

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
