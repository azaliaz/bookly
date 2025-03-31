package storage

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
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

func (dbs *DBStorage) SaveUser(user models.User, adminKey string) (string, error) {
	log := logger.Get()
	uuid := uuid.New().String()

	// Проверяем, существует ли пользователь с таким email
	var existingUser models.User
	row := dbs.conn.QueryRow(context.Background(), "SELECT email FROM users WHERE email = $1", user.Email)
	if err := row.Scan(&existingUser.Email); err == nil {
		return "", storerrros.ErrUserExists
	}

	// Генерация хеша пароля
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("save user failed")
		return "", err
	}

	// Логирование хеша
	log.Debug().Str("hash", string(hash)).Send()

	// Устанавливаем хеш пароля и UID
	user.Pass = string(hash)
	user.UID = uuid
	if adminKey == "your-admin-secret-key" {
		user.Role = "admin"
	} else if user.Role == "" {
		user.Role = "user"
	}

	// Сохраняем пользователя в БД
	_, err = dbs.conn.Exec(context.Background(), "INSERT INTO users (uid, email, pass, age, role) VALUES ($1, $2, $3, $4, $5)", uuid, user.Email, user.Pass, user.Age, user.Role)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert user")
		return "", err
	}

	return uuid, nil
}

func (dbs *DBStorage) ValidUser(user models.User) (string, error) {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	var usr models.User
	row := dbs.conn.QueryRow(ctx, "SELECT uid, email, pass, age, role FROM users WHERE email = $1", user.Email)
	if err := row.Scan(&usr.UID, &usr.Email, &usr.Pass, &usr.Age, &usr.Role); err != nil {
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
	row := dbs.conn.QueryRow(ctx, "SELECT uid, email, pass, age, role FROM users WHERE uid = $1", uid)
	var usr models.User
	if err := row.Scan(&usr.UID, &usr.Email, &usr.Pass, &usr.Age, &usr.Role); err != nil {
		log.Error().Err(err).Msg("failed scan db data")
		return models.User{}, err
	}

	//логирование успешного результата
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
