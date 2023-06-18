package dao

import "errors"

var (
	ErrConstraintViolation = errors.New("a uniqueness constraint was violated")
	ErrNotFound            = errors.New("the requested resource was not found")
)
