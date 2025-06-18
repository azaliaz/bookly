package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/azaliaz/bookly/cart-service/internal/domain/models"
	"github.com/azaliaz/bookly/cart-service/internal/logger"
	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
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

func (s *DBStorage) AddBookToCart(cartID, BID string) error {
	ctx := context.Background()
	var exists bool

	err := s.conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM books WHERE bid = $1)`, BID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if book exists: %v", err)
	}
	if !exists {
		return fmt.Errorf("book with ID %s does not exist", BID)
	}

	err = s.conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM cart_items WHERE cart_id = $1 AND book_id = $2)", cartID, BID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if book is in cart: %v", err)
	}
	if exists {
		return nil
	}

	itemID := uuid.New().String()
	_, err = s.conn.Exec(ctx, `
		INSERT INTO cart_items (item_id, cart_id, book_id) 
		VALUES ($1, $2, $3)
	`, itemID, cartID, BID)
	if err != nil {
		return fmt.Errorf("failed to insert book into cart: %v", err)
	}

	_, err = s.conn.Exec(ctx, `
		UPDATE books
		SET rating = (
			SELECT COUNT(DISTINCT c.user_id)
			FROM cart_items ci
			JOIN cart c ON ci.cart_id = c.cart_id
			WHERE ci.book_id = $1
		)
		WHERE bid = $1
	`, BID)
	if err != nil {
		return fmt.Errorf("failed to update rating: %v", err)
	}

	return nil
}

func (s *DBStorage) GetCartItems(cartID string) ([]models.CartItem, error) {
	rows, err := s.conn.Query(context.Background(), "SELECT item_id, cart_id, book_id FROM cart_items WHERE cart_id = $1", cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.CartItem
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ItemID, &item.CartID, &item.BID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *DBStorage) RemoveBookFromCart(itemID string) error {
	ctx := context.Background()

	var bookID string
	err := s.conn.QueryRow(ctx, `
		SELECT book_id FROM cart_items WHERE item_id = $1
	`, itemID).Scan(&bookID)
	if err != nil {
		return fmt.Errorf("failed to find book_id for cart item: %v", err)
	}

	_, err = s.conn.Exec(ctx, `
		DELETE FROM cart_items WHERE item_id = $1
	`, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove item from cart: %v", err)
	}

	_, err = s.conn.Exec(ctx, `
		UPDATE books SET rating = rating - 1 WHERE bid = $1 AND rating > 0
	`, bookID)
	if err != nil {
		return fmt.Errorf("failed to update book favorites count: %v", err)
	}

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
