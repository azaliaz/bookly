package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/azaliaz/bookly/book-service/internal/domain/consts"
	"github.com/azaliaz/bookly/book-service/internal/domain/models"
	"github.com/azaliaz/bookly/book-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/book-service/internal/storage/errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

type DBStorage struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context, addr string) (*DBStorage, error) {
	config, err := pgxpool.ParseConfig(addr)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return &DBStorage{pool: pool}, nil
}

func (dbs *DBStorage) SaveBook(book models.Book) error {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	var bid string
	err := dbs.pool.QueryRow(ctx, `SELECT bid FROM books WHERE lable=$1 AND author=$2`, book.Lable, book.Author).Scan(&bid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			bid := uuid.New().String()
			_, err := dbs.pool.Exec(ctx,
				`INSERT INTO books (bid, lable, author, "desc", age, genre, rating, cover_url, pdf_url) 
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				bid, book.Lable, book.Author, book.Desc, book.Age, book.Genre, book.Rating, book.CoverURL, book.PDFURL)
			if err != nil {
				log.Error().Err(err).Msg("save book failed")
				return err
			}
			return nil
		}
		log.Error().Err(err).Msg("get book failed")
		return err
	}
	log.Debug().Str("bid", bid).Msg("book already exists")
	return nil
}

func (dbs *DBStorage) SaveBooks(books []models.Book) error {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	tx, err := dbs.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			_ = tx.Commit(ctx)
		}
	}()

	for _, book := range books {
		var bid string
		err = tx.QueryRow(ctx, `SELECT bid FROM books WHERE lable = $1 AND author = $2`, book.Lable, book.Author).Scan(&bid)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				bid := uuid.New().String()
				_, err = tx.Exec(ctx,
					`INSERT INTO books (bid, lable, author, "desc", age, genre, rating, cover_url, pdf_url) 
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
					bid, book.Lable, book.Author, book.Desc, book.Age, book.Genre, book.Rating, book.CoverURL, book.PDFURL)
				if err != nil {
					log.Error().Err(err).Msg("insert book failed")
					return err
				}
				continue
			}
			log.Error().Err(err).Msg("check book failed")
			return err
		}
		log.Debug().Str("bid", bid).Msg("book already exists")
	}
	return nil
}

func (dbs *DBStorage) GetBooks() ([]models.Book, error) {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	rows, err := dbs.pool.Query(ctx, `SELECT bid, lable, author, "desc", age, genre, rating, cover_url, pdf_url FROM books`)
	if err != nil {
		log.Error().Err(err).Msg("failed get all books from db")
		return nil, err
	}
	defer rows.Close()

	var books []models.Book
	for rows.Next() {
		var book models.Book
		if err := rows.Scan(&book.BID, &book.Lable, &book.Author, &book.Desc, &book.Age, &book.Genre, &book.Rating, &book.CoverURL, &book.PDFURL); err != nil {
			log.Error().Err(err).Msg("failed to scan data from db")
			return nil, err
		}
		books = append(books, book)
	}
	return books, nil
}

func (dbs *DBStorage) GetBook(bid string) (models.Book, error) {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	row := dbs.pool.QueryRow(ctx, `SELECT bid, lable, author, "desc", age, genre, rating, cover_url, pdf_url FROM books WHERE bid = $1`, bid)

	var book models.Book
	if err := row.Scan(&book.BID, &book.Lable, &book.Author, &book.Desc, &book.Age, &book.Genre, &book.Rating, &book.CoverURL, &book.PDFURL); err != nil {
		log.Error().Err(err).Msg("failed to scan data from db")
		return models.Book{}, err
	}
	return book, nil
}

func (dbs *DBStorage) DeleteBook(bid string) error {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	res, err := dbs.pool.Exec(ctx, "DELETE FROM books WHERE bid = $1", bid)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete book")
		return err
	}
	if res.RowsAffected() == 0 {
		log.Warn().Str("bid", bid).Msg("book not found")
		return errors.New("book not found")
	}
	log.Info().Str("bid", bid).Msg("book deleted successfully")
	return nil
}

//	func (dbs *DBStorage) GetBooksWithSearchAndSort(searchTerm string, sortBy string, ascending bool) ([]models.Book, error) {
//		log := logger.Get()
//		ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
//		defer cancel()
//
//		var orderDirection string
//		if ascending {
//			orderDirection = "ASC"
//		} else {
//			orderDirection = "DESC"
//		}
//
//		baseQuery := `SELECT bid, lable, author, "desc", age, genre, rating, cover_url, pdf_url FROM books`
//		searchQuery := ""
//		args := []interface{}{}
//
//		if searchTerm != "" {
//			searchQuery = ` WHERE lable ILIKE $1 OR author ILIKE $1`
//			args = append(args, "%"+searchTerm+"%")
//		}
//
//		var sortQuery string
//		switch sortBy {
//		case "lable":
//			sortQuery = ` ORDER BY lable ` + orderDirection
//		case "author":
//			sortQuery = ` ORDER BY author ` + orderDirection
//		case "rating":
//			sortQuery = ` ORDER BY rating ` + orderDirection
//		default:
//			sortQuery = ` ORDER BY bid ` + orderDirection
//		}
//
//		fullQuery := baseQuery + searchQuery + sortQuery
//
//		rows, err := dbs.pool.Query(ctx, fullQuery, args...)
//		if err != nil {
//			log.Error().Err(err).Msg("failed to get books from db")
//			return nil, err
//		}
//		defer rows.Close()
//
//		var books []models.Book
//		for rows.Next() {
//			var book models.Book
//			if err := rows.Scan(&book.BID, &book.Lable, &book.Author, &book.Desc, &book.Age, &book.Genre, &book.Rating, &book.CoverURL, &book.PDFURL); err != nil {
//				log.Error().Err(err).Msg("failed to scan data from db")
//				return nil, err
//			}
//			books = append(books, book)
//		}
//		return books, nil
//	}

func (dbs *DBStorage) GetBooksWithFilters(searchTerm string, genres []string, year string, sortBy string, ascending bool) ([]models.Book, error) {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	var orderDirection string
	if ascending {
		orderDirection = "ASC"
	} else {
		orderDirection = "DESC"
	}

	baseQuery := `SELECT bid, lable, author, "desc", age, genre, rating, cover_url, pdf_url FROM books`
	var conditions []string
	var args []interface{}
	argPos := 1

	if searchTerm != "" {
		conditions = append(conditions, fmt.Sprintf("(lable ILIKE $%d OR author ILIKE $%d)", argPos, argPos+1))
		args = append(args, "%"+searchTerm+"%", "%"+searchTerm+"%")
		argPos += 2
	}

	if len(genres) > 0 {
		var genreConds []string
		for _, g := range genres {
			genreConds = append(genreConds, fmt.Sprintf("genre ILIKE $%d", argPos))
			args = append(args, "%"+g+"%")
			argPos++
		}
		conditions = append(conditions, "("+strings.Join(genreConds, " OR ")+")")
	}

	if year != "" {
		conditions = append(conditions, fmt.Sprintf("age = $%d", argPos))
		args = append(args, year)
		argPos++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var sortQuery string
	switch sortBy {
	case "lable":
		sortQuery = ` ORDER BY lable ` + orderDirection
	case "author":
		sortQuery = ` ORDER BY author ` + orderDirection
	case "rating":
		sortQuery = ` ORDER BY rating ` + orderDirection
	case "genre":
		sortQuery = ` ORDER BY genre ` + orderDirection
	case "age":
		sortQuery = ` ORDER BY age ` + orderDirection
	default:
		sortQuery = ` ORDER BY bid ` + orderDirection
	}

	fullQuery := baseQuery + whereClause + sortQuery

	rows, err := dbs.pool.Query(ctx, fullQuery, args...)
	if err != nil {
		log.Error().Err(err).Msg("failed to get books from db")
		return nil, err
	}
	defer rows.Close()

	var books []models.Book
	for rows.Next() {
		var book models.Book
		if err := rows.Scan(&book.BID, &book.Lable, &book.Author, &book.Desc, &book.Age, &book.Genre, &book.Rating, &book.CoverURL, &book.PDFURL); err != nil {
			log.Error().Err(err).Msg("failed to scan data from db")
			return nil, err
		}
		books = append(books, book)
	}

	if len(books) == 0 {
		return nil, storerrros.ErrEmptyBooksList
	}

	return books, nil
}

func Migrations(dbDsn string, migrationsPath string) error {
	log := logger.Get()
	migratePath := fmt.Sprintf("file://%s", migrationsPath)
	m, err := migrate.New(migratePath, dbDsn)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info().Msg("no migrations apply")
			return nil
		}
		return err
	}
	log.Info().Msg("all migrations apply")
	return nil
}
