package storerrros

import "errors"

var (
	ErrBookNoExist    = errors.New("book does not exists")
	ErrEmptyBooksList = errors.New("empty books list")
)
