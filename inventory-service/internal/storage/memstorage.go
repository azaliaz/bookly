package storage

import (
	// "golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"

	"github.com/azaliaz/bookly/book-service/internal/domain/models"
	"github.com/azaliaz/bookly/book-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/book-service/internal/storage/errors"
)

type MemStorage struct {
	bookStor  map[string]models.Book
}

func New() *MemStorage {
	return &MemStorage{
		bookStor:  make(map[string]models.Book),
	}
}

func (ms *MemStorage) SaveBook(book models.Book) error {
	memBook, err := ms.findBook(book)
    if err == nil {
        memBook.Count += book.Count 
        ms.bookStor[memBook.BID] = memBook
        return nil
    }
    // используем значение из запроса на добавление книги
    bid := uuid.New().String()
    ms.bookStor[bid] = book
    return nil
}

func (ms *MemStorage) SaveBooks(_ []models.Book) error {
	return nil
}

func (ms *MemStorage) GetBooks() ([]models.Book, error) {
	var books []models.Book
	for _, book := range ms.bookStor {
		books = append(books, book)
	}
	if len(books) < 1 {
		return nil, storerrros.ErrEmptyBooksList
	}
	return books, nil
}

func (ms *MemStorage) GetBook(bid string) (models.Book, error) {
	log := logger.Get()
	book, ok := ms.bookStor[bid]
	if !ok {
		log.Error().Str("bid", bid).Msg("user not found")
		return models.Book{}, storerrros.ErrBookNoExist
	}
	return book, nil
}

func (ms *MemStorage) findBook(value models.Book) (models.Book, error) {
	for _, book := range ms.bookStor {
		if book.Lable == value.Lable && book.Author == value.Author {
			return book, nil
		}
	}
	return models.Book{}, storerrros.ErrBookNoExist
}

func (ms *MemStorage) DeleteBook(bid string) error {
	log := logger.Get()

	
	if _, exists := ms.bookStor[bid]; !exists {
		log.Warn().Str("bid", bid).Msg("book not found")
		return storerrros.ErrBookNoExist
	}
	
	delete(ms.bookStor, bid)
	log.Info().Str("bid", bid).Msg("book deleted successfully")

	return nil
}
func (ms *MemStorage) DeleteBook(bid string) error {
    log := logger.Get()
    book, ok := ms.bookStor[bid]
    if !ok {
        log.Warn().Str("bid", bid).Msg("book not found")
        return storerrros.ErrBookNoExist
    }

    if book.Count > 1 {
        book.Count-- 
        ms.bookStor[bid] = book
        log.Info().Str("bid", bid).Msg("one instance of the book deleted")
    } else {
        delete(ms.bookStor, bid) 
        log.Info().Str("bid", bid).Msg("book deleted completely")
    }

    return nil
}


func (ms *MemStorage) SetDeleteStatus(bid string) error {
	log := logger.Get()
	
	book, ok := ms.bookStor[bid]
	if !ok {
		log.Error().Str("bid", bid).Msg("book not found")
		return storerrros.ErrBookNoExist
	}
	book.Deleted = true
	ms.bookStor[bid] = book
	log.Info().Str("bid", bid).Msg("book marked as deleted")
	return nil
}
