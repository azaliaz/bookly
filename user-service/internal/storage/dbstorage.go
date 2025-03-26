package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/azaliaz/bookly/user-service/internal/domain/consts"
	"github.com/azaliaz/bookly/user-service/internal/domain/models"
	"github.com/azaliaz/bookly/user-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/user-service/internal/storage/errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
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

func (dbs *DBStorage) SaveUser(user models.User) (string, error) {
	log := logger.Get()
	uuid := uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("save user failed")
		return "", err
	}
	log.Debug().Str("hash", string(hash)).Send()
	user.Pass = string(hash)
	user.UID = uuid
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	_, err = dbs.conn.Exec(ctx, "INSERT INTO users (uid, email, pass, age) VALUES ($1, $2, $3, $4)",
		user.UID, user.Email, user.Pass, user.Age)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				return "", storerrros.ErrUserExists
			}
		}
		return "", err
	}
	return user.UID, nil
}

func (dbs *DBStorage) ValidUser(user models.User) (string, error) {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	row := dbs.conn.QueryRow(ctx, "SELECT uid, email, pass FROM users WHERE email = $1", user.Email)
	var usr models.User
	if err := row.Scan(&usr.UID, &usr.Email, &usr.Pass); err != nil {
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
	row := dbs.conn.QueryRow(ctx, "SELECT uid, email, pass, age FROM users WHERE uid = $1", uid)
	var usr models.User //сюда будут записаны данные о пользователе
	if err := row.Scan(&usr.UID, &usr.Email, &usr.Pass, &usr.Age); err != nil {
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
