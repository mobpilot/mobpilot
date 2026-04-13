package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNotAuthorized        = errors.New("not authorized")
)

type invalidInputError struct{ msg string }

func (e *invalidInputError) Error() string { return fmt.Sprintf("invalid input: %s", e.msg) }

func ErrInvalidInput(msg string) error { return &invalidInputError{msg: msg} }

func IsInvalidInput(err error) bool {
	var t *invalidInputError
	return errors.As(err, &t)
}
