package service

import "errors"

var (
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("subscription not found")
)
