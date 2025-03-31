package storage

import (
	// "golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"

	"github.com/azaliaz/bookly/cart-service/internal/domain/models"
	"github.com/azaliaz/bookly/cart-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/cart-service/internal/storage/errors"
)

type UserCart struct {
	UserID    string
	Books     []models.Book
	TotalCost float64
}

type MemStorage struct {
	bookStor map[string]models.Book
	carts    map[string]*UserCart // Карта для хранения корзин пользователей
}

func New() *MemStorage {
	return &MemStorage{
		bookStor: make(map[string]models.Book), // Инициализация хранилища книг
		carts:    make(map[string]*UserCart),   // Инициализация карты для корзин
	}
}

func (ms *MemStorage) SaveBook(book models.Book) {
	ms.bookStor[book.ID] = book
}

// Получение книги по ID
func (ms *MemStorage) GetBook(bookID string) (models.Book, error) {
	book, exists := ms.bookStor[bookID]
	if !exists {
		return models.Book{}, errors.New("book not found")
	}
	return book, nil
}

// Создание или получение корзины пользователя
func (ms *MemStorage) GetUserCart(userID string) (*UserCart, error) {
	cart, exists := ms.carts[userID]
	if !exists {
		// Если корзина не существует, создаем новую
		cart = &UserCart{
			UserID: userID,
			Books:  []models.Book{},
		}
		ms.carts[userID] = cart
	}
	return cart, nil
}

// Добавление книги в корзину пользователя
func (ms *MemStorage) AddBookToCart(userID string, bookID string, count int) error {
	// Получаем книгу по ID
	book, err := ms.GetBook(bookID)
	if err != nil {
		return err
	}

	// Получаем корзину пользователя
	cart, err := ms.GetUserCart(userID)
	if err != nil {
		return err
	}

	// Проверяем, есть ли книга уже в корзине
	for i, cartBook := range cart.Books {
		if cartBook.ID == book.ID {
			// Если книга уже в корзине, обновляем количество
			cart.Books[i].Count += count
			cart.TotalCost += float64(count) * book.Price
			return nil
		}
	}

	// Если книги нет в корзине, добавляем новую книгу
	book.Count = count
	cart.Books = append(cart.Books, book)
	cart.TotalCost += float64(count) * book.Price
	return nil
}

// Удаление книги из корзины пользователя
func (ms *MemStorage) RemoveBookFromCart(userID string, bookID string) error {
	// Получаем корзину пользователя
	cart, err := ms.GetUserCart(userID)
	if err != nil {
		return err
	}

	// Ищем книгу в корзине
	for i, cartBook := range cart.Books {
		if cartBook.ID == bookID {
			// Если книга найдена, удаляем её из корзины
			cart.TotalCost -= float64(cartBook.Count) * cartBook.Price
			cart.Books = append(cart.Books[:i], cart.Books[i+1:]...)
			return nil
		}
	}

	return errors.New("book not found in cart")
}

// Получение всех книг в корзине пользователя
func (ms *MemStorage) GetCartItems(userID string) ([]models.Book, error) {
	cart, err := ms.GetUserCart(userID)
	if err != nil {
		return nil, err
	}
	return cart.Books, nil
}
