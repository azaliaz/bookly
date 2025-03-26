package storage

import (
	"context"
	"errors"
	"fmt"

	// "golang.org/x/crypto/bcrypt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	// "github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"

	// "github.com/jackc/pgx/v5/pgconn"
	"github.com/azaliaz/bookly/book-service/internal/domain/consts"
	"github.com/azaliaz/bookly/book-service/internal/domain/models"
	"github.com/azaliaz/bookly/book-service/internal/logger"
)

type DBStorage struct {
	conn *pgx.Conn
}

func NewDB(ctx context.Context, addr string) (*DBStorage, error) {
	conn, err := pgx.Connect(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &DBStorage{
		conn: conn,
	}, nil
}

//Метод SaveBook выполняет следующие действия:
// Ищет книгу в базе данных по ее метке и автору.
// Если книга не найдена, она добавляется в базу данных с новым уникальным идентификатором.
// Если книга найдена, увеличивается количество ее экземпляров на 1.
// В случае возникновения ошибок при работе с базой данных они логируются и возвращаются.
// В случае успешного завершения метода возвращается nil, что означает отсутствие ошибок.
func (dbs *DBStorage) SaveBook(book models.Book) error {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	//поиск существующей книги 
	var bid string
	var count int
	log.Debug().Msgf("search book %s %s", book.Author, book.Lable)
	err := dbs.conn.QueryRow(ctx, `SELECT bid, count FROM books 
		WHERE lable=$1 AND author=$2`, book.Lable, book.Author).Scan(&bid, &count)
	//обработка ошибки если книга не найдена 	
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			bid := uuid.New().String()
			_, err := dbs.conn.Exec(ctx,
				`INSERT INTO books (bid, lable, author, "desc", age, count) 
				VALUES ($1, $2, $3, $4, $5, $6)`,
				bid, book.Lable, book.Author, book.Desc, book.Age, book.Count)
			if err != nil {
				log.Error().Err(err).Msg("save book failed")
				return nil
			}
			return nil
		}
		log.Error().Err(err).Msg("get book count failed")
		return err
	}
	log.Debug().Int("book count", count).Msg("book count")
	//обновление количества экземпляров книги
	_, err = dbs.conn.Exec(ctx, "UPDATE books SET count=count + 1 WHERE bid=$1", bid)
	if err != nil {
		log.Error().Err(err).Msg("uodate book count failed")
		return err
	}
	return nil
}
//предназначен для сохранения множества книг в базу данных.
func (dbs *DBStorage) SaveBooks(books []models.Book) error {
	log := logger.Get()
	log.Debug().Any("books", books).Msg("check books")
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel() 
	tx, err := dbs.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // гарантирует откат транзакции, если метод завершится с ошибкой. Это помогает избежать частичных изменений в базе данных.
	
	//запрос для поиска книги по метке (lable) и автору (author).
	_, err = tx.Prepare(ctx, "saveBook", `SELECT bid, count FROM books 
		WHERE lable=$1 AND author=$2`)
	if err != nil {
		log.Error().Err(err).Msg("prepare save book req failed")
		return err
	}
	//запрос для вставки новой книги в таблицу.
	_, err = tx.Prepare(ctx, "insertBook", `INSERT INTO books (bid, lable, author, "desc", age, count) 
				VALUES ($1, $2, $3, $4, $5, $6)`)
	if err != nil {
		log.Error().Err(err).Msg("prepare insert book req failed")
		return err
	}
	// запрос для обновления количества экземпляров книги.
	_, err = tx.Prepare(ctx, "updateBook", `UPDATE books SET count=count + 1 WHERE bid=$1`)
	if err != nil {
		log.Error().Err(err).Msg("prepare update book req failed")
		return err
	}
	for _, book := range books {
		var bid string
		var count int
		log.Debug().Msgf("search book %s %s", book.Author, book.Lable)
		err := tx.QueryRow(ctx, "saveBook", book.Lable, book.Author).Scan(&bid, &count)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				bid := uuid.New().String()
				count := 1
				_, err := tx.Exec(ctx, `insertBook`,
					bid, book.Lable, book.Author, book.Desc, book.Age, count)
				if err != nil {
					log.Error().Err(err).Msg("save book failed")
					return nil
				}
				continue
			}
			log.Error().Err(err).Msg("get book count failed")
			return err
		}
		log.Debug().Int("book count", count).Msg("book count")
		_, err = tx.Exec(ctx, "updateBook", bid)
		if err != nil {
			log.Error().Err(err).Msg("uodate book count failed")
			return err
		}
	}
	return tx.Commit(ctx)
}

func (dbs *DBStorage) GetBooks() ([]models.Book, error) {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	//запрос к базе данных выбирает все книги из таблицы books, которые не помечены как удаленные
	rows, err := dbs.conn.Query(ctx, `SELECT bid, lable, author, "desc", age, count FROM books WHERE deleted=false`)
	if err != nil {
		log.Error().Err(err).Msg("failed get all books from db")
		return nil, err
	}
	//чтение данных из результата запроса
	var books []models.Book //создается пустой срез в который будут добавляться книги
	for rows.Next() { // цикл перебирает все строки в результате запроса гдк каждая строка представляет собой книгу.
		var book models.Book //данные с каждой строки (одна книга)
		if err := rows.Scan(&book.BID, &book.Lable, &book.Author, &book.Desc, &book.Age, &book.Count); err != nil {
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
	//запрос выбирает одну книга и проверяет не удалена ли она 
	row := dbs.conn.QueryRow(ctx, `SELECT bid, lable, author, "desc", age, count FROM books WHERE bid = $1 AND deleted=false`, bid)
	//извлечение данных из результата запроса
	var book models.Book
	if err := row.Scan(&book.BID, &book.Lable, &book.Author, &book.Desc, &book.Age, &book.Count); err != nil {
		log.Error().Err(err).Msg("failed to scan data from db")
		return models.Book{}, err
	}
	return book, nil
}
//изменение статуса (удаление)
func (dbs *DBStorage) SetDeleteStatus(bid string) error {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	//выполнение запроса на обновление записи
	_, err := dbs.conn.Exec(ctx, "UPDATE books SET deleted=true WHERE bid=$1", bid)
	if err != nil {
		log.Error().Msg("set deleted status failed")
		return err
	}
	return nil
}

func (dbs *DBStorage) DeleteBooks() error {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	//создание транзакции
	tx, err := dbs.conn.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to start tx")
		return err
	}
	defer tx.Rollback(ctx)
	//удаляет все книги которые помечены как удаленнные 
	if _, err = tx.Exec(ctx, "DELETE FROM books WHERE deleted=true"); err != nil {
		log.Error().Err(err).Msg("delete books failed")
		return err
	}
	return tx.Commit(ctx)
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
			log.Info().Msg("no mirations apply")
			return nil
		}
		return err
	}
	log.Info().Msg("all mirations apply")
	return nil
}
