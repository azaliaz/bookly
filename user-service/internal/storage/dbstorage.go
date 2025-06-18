package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	// "github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"

	// "github.com/jackc/pgx/v5/pgconn"
	"github.com/azaliaz/bookly/user-service/internal/domain/consts"
	"github.com/azaliaz/bookly/user-service/internal/domain/models"
	"github.com/azaliaz/bookly/user-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/user-service/internal/storage/errors"
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

func (dbs *DBStorage) UpdateUserCartID(uid, cartID string) error {
	log := logger.Get()
	_, err := dbs.conn.Exec(context.Background(), "UPDATE users SET cart_id = $1 WHERE uid = $2", cartID, uid)
	if err != nil {
		log.Error().Err(err).Msg("failed to update cart ID")
		return err
	}
	return nil
}
func (dbs *DBStorage) SaveUser(user models.User, adminKey string) (string, error) {
	log := logger.Get()

	userUUID := uuid.New().String()
	var existingUser models.User
	row := dbs.conn.QueryRow(context.Background(), "SELECT email FROM users WHERE email = $1", user.Email)
	if err := row.Scan(&existingUser.Email); err == nil {
		return "", storerrros.ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("save user failed")
		return "", err
	}
	cartUUID := uuid.New().String()

	user.Pass = string(hash)
	user.UID = userUUID
	user.CartID = cartUUID

	if adminKey == "your-admin-secret-key" {
		user.Role = "admin"
	} else if user.Role == "" {
		user.Role = "user"
	}

	_, err = dbs.conn.Exec(context.Background(), "INSERT INTO users (uid, cart_id, email, name, lastname, pass, age, role) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		userUUID, cartUUID, user.Email, user.Name, user.LastName, user.Pass, user.Age, user.Role)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert user")
		return "", err
	}

	_, err = dbs.conn.Exec(context.Background(), "INSERT INTO cart (cart_id, user_id) VALUES ($1, $2)", cartUUID, userUUID)
	if err != nil {
		log.Error().Err(err).Msg("failed to create cart")
		return "", err
	}
	log.Debug().Any("user to insert", user).Msg("about to save user")

	return userUUID, nil
}

func (dbs *DBStorage) ValidUser(user models.User) (string, error) {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	var usr models.User
	row := dbs.conn.QueryRow(ctx, "SELECT uid, cart_id, email, name, lastname, pass, age, role FROM users WHERE email = $1", user.Email)
	if err := row.Scan(&usr.UID, &usr.CartID, &usr.Email, &usr.Name, &usr.LastName, &usr.Pass, &usr.Age, &usr.Role); err != nil {
		log.Error().Err(err).Msg("failed scan db data")
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(usr.Pass), []byte(user.Pass)); err != nil {
		log.Error().Err(err).Msg("failed compare hash and password")
		return "", storerrros.ErrInvalidPassword
	}
	log.Debug().Any("db user", usr).Msg("user form data base")
	return usr.UID, nil
}

// извлечение данных пользователя
func (dbs *DBStorage) GetUser(uid string) (models.User, error) {
	log := logger.Get()
	//контекст с тайм-аутом будет использован для выполнения запроса к бд
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	row := dbs.conn.QueryRow(ctx, "SELECT uid, cart_id, email, name, lastname, pass, age, role FROM users WHERE uid = $1", uid)
	var usr models.User
	if err := row.Scan(&usr.UID, &usr.CartID, &usr.Email, &usr.Name, &usr.LastName, &usr.Pass, &usr.Age, &usr.Role); err != nil {
		log.Error().Err(err).Msg("failed scan db data")
		return models.User{}, err
	}

	log.Debug().Any("db user", usr).Msg("user form data base")
	return usr, nil
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
