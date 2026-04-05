package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors used by all layers. Infrastructure adapters must map
// database/network errors to these before returning them.
var (
	ErrUserNotFound     = errors.New("user not found")
	ErrArtifactNotFound = errors.New("artifact not found")
	ErrDeviceNotFound   = errors.New("device token not found")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrNotAuthorized    = errors.New("not authorized")
)

// invalidInputError is a typed error for validation failures.
type invalidInputError struct{ msg string }

func (e *invalidInputError) Error() string { return fmt.Sprintf("invalid input: %s", e.msg) }

// ErrInvalidInput returns a domain validation error.
func ErrInvalidInput(msg string) error { return &invalidInputError{msg: msg} }

// IsInvalidInput reports whether err is an ErrInvalidInput.
func IsInvalidInput(err error) bool {
	var t *invalidInputError
	return errors.As(err, &t)
}
