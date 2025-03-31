package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"

	"github.com/azaliaz/bookly/cart-service/internal/domain/consts"
	"github.com/azaliaz/bookly/cart-service/internal/domain/models"
	"github.com/azaliaz/bookly/cart-service/internal/logger"
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
func (dbs *DBStorage) CreateCart(uid string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	var cid string
	err := dbs.conn.QueryRow(ctx, `SELECT cid FROM carts WHERE uid=$1`, uid).Scan(&cid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			cid = uuid.New().String()
			_, err := dbs.conn.Exec(ctx, `INSERT INTO carts (cid, uid) VALUES ($1, $2)`, cid, uid)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return cid, nil
}
func (dbs *DBStorage) AddBookToCart(uid, bid string, count int) error {
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	cid, err := dbs.CreateCart(uid)
	if err != nil {
		return err
	}

	var existingCount int
	err = dbs.conn.QueryRow(ctx, `SELECT count FROM cart_items WHERE cid=$1 AND bid=$2`, cid, bid).Scan(&existingCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Если книги нет в корзине, добавляем новую запись
			_, err := dbs.conn.Exec(ctx, `INSERT INTO cart_items (id, cid, bid, count) VALUES ($1, $2, $3, $4)`,
				uuid.New().String(), cid, bid, count)
			return err
		}
		return err
	}

	// Если книга уже в корзине, обновляем её количество
	_, err = dbs.conn.Exec(ctx, `UPDATE cart_items SET count=count+$1 WHERE cid=$2 AND bid=$3`, count, cid, bid)
	return err
}
func (dbs *DBStorage) GetCartItems(uid string) ([]models.CartItems, error) {
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	cid, err := dbs.CreateCart(uid)
	if err != nil {
		return nil, err
	}

	rows, err := dbs.conn.Query(ctx, `SELECT id, cid, bid, count FROM cart_items WHERE cid=$1`, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.CartItems
	for rows.Next() {
		var item models.CartItems
		if err := rows.Scan(&item.ID, &item.CID, &item.BID, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}
func (dbs *DBStorage) RemoveBookFromCart(uid, bid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	cid, err := dbs.CreateCart(uid)
	if err != nil {
		return err
	}

	var count int
	err = dbs.conn.QueryRow(ctx, `SELECT count FROM cart_items WHERE cid=$1 AND bid=$2`, cid, bid).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("book not found in cart")
		}
		return err
	}

	if count > 1 {
		_, err = dbs.conn.Exec(ctx, `UPDATE cart_items SET count = count - 1 WHERE cid=$1 AND bid=$2`, cid, bid)
	} else {
		_, err = dbs.conn.Exec(ctx, `DELETE FROM cart_items WHERE cid=$1 AND bid=$2`, cid, bid)
	}

	return err
}
