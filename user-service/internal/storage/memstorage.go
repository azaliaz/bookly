package storage

import (
	"github.com/azaliaz/bookly/internal/domain/models"
	"github.com/azaliaz/bookly/internal/logger"
	storerrros "github.com/azaliaz/bookly/internal/storage/errros"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type MemStorage struct {
	usersStor map[string]models.User
	// bookStor  map[string]models.Book
}

func New() *MemStorage {
	return &MemStorage{
		usersStor: make(map[string]models.User),
		// bookStor:  make(map[string]models.Book),
	}
}

func (ms *MemStorage) SaveUser(user models.User) (string, error) {
	log := logger.Get()
	uuid := uuid.New().String()
	if _, err := ms.findUser(user.Email); err == nil {
		return "", storerrros.ErrUserExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("save user failed")
		return "", err
	}
	log.Debug().Str("hash", string(hash)).Send()
	user.Pass = string(hash)
	user.UID = uuid
	ms.usersStor[uuid] = user
	log.Debug().Any("storage", ms.usersStor).Send()
	return uuid, nil
}

func (ms *MemStorage) ValidUser(user models.User) (string, error) {
	log := logger.Get()
	log.Debug().Any("storage", ms.usersStor).Send()
	memUser, err := ms.findUser(user.Email)
	if err != nil {
		return "", err
	}
	if err = bcrypt.CompareHashAndPassword([]byte(memUser.Pass), []byte(user.Pass)); err != nil {
		return "", storerrros.ErrInvalidPassword
	}
	return memUser.UID, nil
}

func (ms *MemStorage) GetUser(uid string) (models.User, error) {
	log := logger.Get()
	//поиск пользователя в хранилище по UID
	//ms.usersStor представляет собой карту (map), где ключом является uid, а значением — объект models.User
	user, ok := ms.usersStor[uid]
	//логирование и обработка ошибки, если пользователь не найден
	if !ok {
		log.Error().Str("uid", uid).Msg("user not found")
		return models.User{}, storerrros.ErrUserNotFound
	}
	return user, nil
}

func (ms *MemStorage) findUser(login string) (models.User, error) {
	for _, user := range ms.usersStor {
		if user.Email == login {
			return user, nil
		}
	}
	return models.User{}, storerrros.ErrUserNoExist
}
