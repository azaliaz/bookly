package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/azaliaz/bookly/cart-service/internal/domain/consts"
	"github.com/azaliaz/bookly/cart-service/internal/domain/models"
	"github.com/azaliaz/bookly/cart-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/cart-service/internal/storage/errors"
	"github.com/golang-migrate/migrate/v4"

	// "github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
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

func (s *DBStorage) CreateCart(UID string) (string, error) {
	var cartID int
	// Убираем ctx, используем дефолтный контекст
	err := s.conn.QueryRow(context.Background(), "INSERT INTO cart (user_id) VALUES ($1) RETURNING cart_id", UID).Scan(&cartID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", cartID), nil
}
func (s *DBStorage) AddBookToCart(cartID, BID string, quantity int) error {
	// Проверка наличия книги в корзине
	var existingQuantity int
	err := s.conn.QueryRow(context.Background(), "SELECT quantity FROM cart_items WHERE cart_id = $1 AND book_id = $2", cartID, BID).Scan(&existingQuantity)

	// Проверка наличия книги в таблице books
	var bookQuantity int
	err = s.conn.QueryRow(context.Background(), "SELECT count FROM books WHERE bid = $1", BID).Scan(&bookQuantity)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("book not found in books table")
		}
		return fmt.Errorf("failed to check book availability in books table: %v", err)
	}

	if bookQuantity < quantity {
		return fmt.Errorf("not enough books in stock")
	}

	_, err = s.conn.Exec(context.Background(), "UPDATE books SET count = count - $1 WHERE bid = $2", quantity, BID)
	if err != nil {
		return fmt.Errorf("failed to update book quantity in books table: %v", err)
	}

	if err == nil && existingQuantity > 0 {
		_, err = s.conn.Exec(context.Background(), "UPDATE cart_items SET quantity = quantity + $1 WHERE cart_id = $2 AND book_id = $3", quantity, cartID, BID)
		if err != nil {
			return fmt.Errorf("failed to update book quantity in cart: %v", err)
		}
	} else {

		_, err := s.conn.Exec(context.Background(), "INSERT INTO cart_items (cart_id, book_id, quantity) VALUES ($1, $2, $3)", cartID, BID, quantity)
		if err != nil {
			return fmt.Errorf("failed to add book to cart: %v", err)
		}
	}

	return nil
}

func (s *DBStorage) GetCartItems(cartID string) ([]models.CartItem, error) {
	rows, err := s.conn.Query(context.Background(), "SELECT item_id, cart_id, book_id, quantity FROM cart_items WHERE cart_id = $1", cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.CartItem
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ItemID, &item.CartID, &item.BID, &item.Quantity); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *DBStorage) RemoveBookFromCart(itemID string) error {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	var bookID, label, author, desc string
	var age, quantity, count int

	err := s.conn.QueryRow(ctx, `SELECT ci.book_id, b.lable, b.author, b."desc", b.age, ci.quantity, b.count
		FROM cart_items ci 
		LEFT JOIN books b ON ci.book_id = b.bid
		WHERE ci.item_id = $1`, itemID).
		Scan(&bookID, &label, &author, &desc, &age, &quantity, &count)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storerrros.ErrBookNotFound
		}
		return fmt.Errorf("failed to check book in cart: %w", err)
	}

	if quantity > 1 {
		_, err := s.conn.Exec(ctx, "UPDATE cart_items SET quantity = quantity - 1 WHERE item_id = $1", itemID)
		if err != nil {
			log.Error().Err(err).Msg("failed to decrement book count in cart")
			return err
		}

	} else {
		_, err = s.conn.Exec(ctx, "DELETE FROM cart_items WHERE item_id = $1", itemID)

		if err != nil {
			return fmt.Errorf("failed to delete book from cart: %w", err)
		}
	}
	if count == 0 {
		_, err = s.conn.Exec(ctx, "DELETE FROM cart_items WHERE item_id = $1", itemID)
		if err != nil {
			log.Error().Err(err).Str("itemID", itemID).Msg("Failed to delete book from cart")
			return fmt.Errorf("failed to delete book from cart: %w", err)
		}
		log.Debug().Str("itemID", itemID).Msg("Deleted book from cart")

	} else {
		_, err = s.conn.Exec(ctx, "UPDATE books SET count = count + 1 WHERE bid = $1", bookID)
		if err != nil {
			log.Error().Err(err).Msg("failed to update book count in books")
			return err
		}
	}
	log.Debug().Str("itemID", itemID).Msg("Book removal process completed successfully")

	return nil
}

func (s *DBStorage) ClearCart(cartID string) error {
	_, err := s.conn.Exec(context.Background(), "DELETE FROM cart_items WHERE cart_id = $1", cartID)
	return err
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
