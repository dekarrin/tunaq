package dao

import "errors"

var (
	ErrConstraintViolation = errors.New("a uniqueness constraint was violated")
	ErrNotFound            = errors.New("The requested resource was not found")
)
