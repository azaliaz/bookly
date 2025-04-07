package storerrros

import "errors"

var (
	ErrBookNoExist  = errors.New("book does not exists")
	ErrCartNotExist = errors.New("cart does not exists")
	ErrBookNotFound = errors.New("book not found in cart")
)
