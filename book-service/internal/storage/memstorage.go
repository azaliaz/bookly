package storage

import (
	// "golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	"sort"
	"strconv"
	"strings"

	"github.com/azaliaz/bookly/book-service/internal/domain/models"
	"github.com/azaliaz/bookly/book-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/book-service/internal/storage/errors"
)

type MemStorage struct {
	bookStor map[string]models.Book
}

func New() *MemStorage {
	return &MemStorage{
		bookStor: make(map[string]models.Book),
	}
}

func (ms *MemStorage) SaveBook(book models.Book) error {
	memBook, err := ms.findBook(book)
	if err == nil {
		ms.bookStor[memBook.BID] = memBook
		return nil
	}
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
func (ms *MemStorage) GetBooksWithFilters(search string, genres []string, year string, sortBy string, ascending bool) ([]models.Book, error) {
	search = strings.ToLower(search)
	var result []models.Book

	for _, book := range ms.bookStor {
		if search != "" && !(strings.Contains(strings.ToLower(book.Author), search) || strings.Contains(strings.ToLower(book.Lable), search)) {
			continue
		}

		if len(genres) > 0 {
			matched := false
			for _, g := range genres {
				if strings.EqualFold(book.Genre, g) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if year != "" && strconv.Itoa(book.Age) != year {
			continue
		}

		result = append(result, book)
	}

	if len(result) == 0 {
		return nil, storerrros.ErrEmptyBooksList
	}

	sort.Slice(result, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "author":
			less = result[i].Author < result[j].Author
		case "lable", "label":
			less = result[i].Lable < result[j].Lable
		case "rating":
			less = result[i].Rating < result[j].Rating
		case "age":
			less = result[i].Age < result[j].Age
		case "genre":
			less = result[i].Genre < result[j].Genre
		default:
			return true
		}

		if ascending {
			return less
		}
		return !less
	})

	return result, nil
}
